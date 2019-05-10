package foundation.icon.test.cases;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.GovScore;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.util.HashMap;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;

import static foundation.icon.test.common.Env.LOG;
import static foundation.icon.test.common.Utils.getMicroTime;
import static org.junit.jupiter.api.Assertions.assertEquals;

@Tag(Constants.TAG_GOVERNANCE)
public class StepTest {
    private static KeyWallet[]testWallets;
    private static IconService iconService;
    private static Env.Chain chain;
    private static final int testWalletNum = 5;
    private static GovScore govScore;
    private static Map<String, BigInteger> defStepCosts = new HashMap<>();
    private static BigInteger defStepPrice;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        govScore = new GovScore(iconService, chain);
        initTransfer();
    }

    @AfterAll
    public static void destroy() throws Exception {
        govScore.setStepPrice(defStepPrice);
        setSteps(defStepCosts);
    }

    public static void initTransfer() throws Exception {
        LOG.infoEntering("initTransfer");

        RpcObject rpcObject = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS,"getStepCosts", null)
                .asObject();
        for(String key : rpcObject.keySet()) {
            defStepCosts.put(key, rpcObject.getItem(key).asInteger());
        }
        defStepPrice = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS,"getStepPrice", null)
                .asInteger();

        testWallets = new KeyWallet[testWalletNum];
        Address[]addrs = new Address[testWalletNum];
        for(int i = 0; i < testWalletNum; i++){
            KeyWallet wallet = KeyWallet.create();
            testWallets[i] = wallet;
            addrs[i] = wallet.getAddress();
        }

        Utils.transferAndCheck(iconService, chain, chain.godWallet, addrs, Constants.DEFAULT_BALANCE);
        Utils.transferAndCheck(iconService, chain, chain.godWallet, chain.governorWallet.getAddress(), Constants.DEFAULT_BALANCE);

        govScore.setMaxStepLimit("invoke", new BigInteger("10000000000"));
        govScore.setMaxStepLimit("query", new BigInteger("10000000000"));
        govScore.setStepPrice(BigInteger.ONE);
        Map<String, BigInteger> steps = new HashMap<>();
        steps.put("default", BigInteger.valueOf(100));
        steps.put("input", BigInteger.valueOf(1));
        steps.put("contractCreate", BigInteger.valueOf(1000));
        steps.put("contractSet", BigInteger.valueOf(1));
        steps.put("contractCall", BigInteger.valueOf(100));
        setSteps(steps);
        LOG.infoExiting();
    }

    private static void setSteps(Map<String, BigInteger> stepsMap) throws Exception {
        List<Bytes> txList = new LinkedList<>();
        for(String type : stepsMap.keySet()) {
            RpcObject params = new RpcObject.Builder()
                    .put("type", new RpcValue(type))
                    .put("cost", new RpcValue(stepsMap.get(type)))
                    .build();
            txList.add(Utils.sendTransactionWithCall(iconService, chain.networkId,
                    chain.governorWallet, Constants.GOV_ADDRESS, "setStepCost", params, 0, false));
        }

        for(Bytes tx : txList) {
            TransactionResult result =
                    Utils.getTransactionResult(iconService, tx, Constants.DEFAULT_WAITING_TIME);
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        }
    }

    class StepTransaction {
        Map<String, BigInteger> steps = new HashMap<>();
        BigInteger stepPrice;
        BigInteger usedStep;
        BigInteger treasuryFee;
        Address scoreAddr;
        StepTransaction() throws Exception {
            RpcObject rpcObject = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS,"getStepCosts", null)
                    .asObject();
            for(String key : rpcObject.keySet()) {
                steps.put(key, rpcObject.getItem(key).asInteger());
            }
            stepPrice = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS,"getStepPrice", null)
                    .asInteger();
        }

        BigInteger usedCoin() {
            return usedStep.multiply(stepPrice);
        }

        BigInteger calcTransactionStep(Transaction tx) {
            // default + input * data
            BigInteger stepUsed = steps.get("default");
            if(tx.getDataType().equals("message")) {
                // tx.getData() returns message with no quotes
                long dataSize = tx.getData().asString().length() + 2;
                stepUsed = stepUsed.add(BigInteger.valueOf(dataSize).multiply(steps.get("input")));
            }
            else {
                int dataLen = 2; // curly brace
                RpcObject rpcObject = tx.getData().asObject();
                for(String key : rpcObject.keySet()) {
                    // Quotes for key(2) + colon(1) + comma(1)
                    dataLen += 4;
                    dataLen += key.length();
                    if("params".equals(key)) {
                        RpcObject paramObj = rpcObject.getItem(key).asObject();
                        dataLen += 2; // curly brace
                        for(String param : paramObj.keySet()) {
                            dataLen += paramObj.getItem(param).asString().length();
                            dataLen += param.length();
                            // Quotes for key(2) + Quotes for value(2) + colon(1) + comma(1)
                            dataLen += 6;
                        }
                        dataLen -= 1; // subtract last comma
                    }
                    else {
                        dataLen += rpcObject.getItem(key).asString().length();
                        dataLen += 2;// add Quotes for value
                    }
                }
                dataLen -= 1; // subtract last comma
                stepUsed = stepUsed.add(BigInteger.valueOf(dataLen));
            }
            return stepUsed;
        }

        BigInteger calcDeployStep(Transaction tx, byte[] content, boolean update) {
            // contractCreate or contractUpdate
            // contractSet * codeLen
            BigInteger stepUsed;
            if(Utils.isAudit(iconService)) {
                stepUsed = calcTransactionStep(tx);
            }
            else {
                // add step for calling on_install
                stepUsed = calcCallStep(tx);
            }
            BigInteger codeLen = BigInteger.valueOf(content.length);
            if(update) {
                stepUsed = steps.get("contractUpdate").add(stepUsed);
            }
            else {
                stepUsed = steps.get("contractCreate").add(stepUsed);
            }
            stepUsed = stepUsed.add(steps.get("contractSet").multiply(codeLen));
            return stepUsed;
        }

        BigInteger calcCallStep(Transaction tx) {
            BigInteger stepUsed = calcTransactionStep(tx);
            // contractCall
            stepUsed = steps.get("contractCall").add(stepUsed);
            return stepUsed;
        }

        // return used coin
        BigInteger transfer(KeyWallet from, Address to, BigInteger value, String msg) throws Exception {
            BigInteger prevTresury = iconService.getBalance(Constants.TREASURY_ADDRESS).execute();
            BigInteger prevBal = iconService.getBalance(from.getAddress()).execute();
            Transaction transaction = TransactionBuilder.newBuilder()
                    .nid(BigInteger.valueOf(chain.networkId))
                    .from(from.getAddress())
                    .to(to)
                    .value(value)
                    .stepLimit(new BigInteger("100000"))
                    .timestamp(getMicroTime())
                    .nonce(new BigInteger("1"))
                    .message(msg)
                    .build();
            this.usedStep = calcTransactionStep(transaction);

            SignedTransaction signedTransaction = new SignedTransaction(transaction, from);
            Bytes txHash = iconService.sendTransaction(signedTransaction).execute();
            TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            BigInteger bal = iconService.getBalance(from.getAddress()).execute();
            BigInteger treasury = iconService.getBalance(Constants.TREASURY_ADDRESS).execute();
            treasuryFee = treasury.subtract(prevTresury);
            return prevBal.subtract(bal.add(value));
        }

        BigInteger deploy(KeyWallet from, Address to, String contentPath, RpcObject params) throws Exception {
            BigInteger prevTresury = iconService.getBalance(Constants.TREASURY_ADDRESS).execute();
            BigInteger prevBal = iconService.getBalance(from.getAddress()).execute();
            byte[] content = Utils.zipContent(contentPath);
            TransactionBuilder.Builder builder = TransactionBuilder.newBuilder()
                    .nid(BigInteger.valueOf(chain.networkId))
                    .from(from.getAddress())
                    .stepLimit(new BigInteger("10000000"))
                    .timestamp(getMicroTime())
                    .nonce(new BigInteger("1"));
            if(to != null) {
                builder = builder.to(to);
            }
            else {
                builder = builder.to(Constants.CHAINSCORE_ADDRESS);
            }

            Transaction transaction = builder.deploy(Constants.CONTENT_TYPE, content)
                    .params(params).build();
            this.usedStep = calcDeployStep(transaction, content, to != null);

            SignedTransaction signedTransaction = new SignedTransaction(transaction, from);
            Bytes txHash = iconService.sendTransaction(signedTransaction).execute();
            TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());

            try {
                Utils.acceptIfAuditEnabled(iconService, chain, txHash);
            }
            catch(TransactionFailureException ex) {
                LOG.infoExiting();
                throw ex;
            }

            this.scoreAddr = new Address(result.getScoreAddress());
            BigInteger bal = iconService.getBalance(from.getAddress()).execute();
            BigInteger treasury = iconService.getBalance(Constants.TREASURY_ADDRESS).execute();
            treasuryFee = treasury.subtract(prevTresury);
            return prevBal.subtract(bal);
        }

        BigInteger call(KeyWallet from, Address to, String method, RpcObject params, BigInteger value) throws Exception {
            BigInteger prevTresury = iconService.getBalance(Constants.TREASURY_ADDRESS).execute();
            BigInteger prevBal = iconService.getBalance(from.getAddress()).execute();
            TransactionBuilder.Builder builder = TransactionBuilder.newBuilder()
                    .nid(BigInteger.valueOf(chain.networkId))
                    .from(from.getAddress())
                    .to(to)
                    .stepLimit(BigInteger.valueOf(1000000));

            if ((value != null) && value.bitLength() != 0) {
                builder = builder.value(value);
            }

            Transaction transaction;
            if (params != null) {
                transaction = builder.call(method).params(params).build();
            } else {
                transaction = builder.call(method).build();
            }
            this.usedStep = calcCallStep(transaction);

            Bytes txHash = iconService.sendTransaction(new SignedTransaction(transaction, from)).execute();
            TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            BigInteger bal = iconService.getBalance(from.getAddress()).execute();
            BigInteger treasury = iconService.getBalance(Constants.TREASURY_ADDRESS).execute();
            treasuryFee = treasury.subtract(prevTresury);
            return prevBal.subtract(bal);
        }
    }

    @Test
    public void transferStep() throws Exception {
        LOG.infoEntering("transferStep" );
        StepTransaction sTx = new StepTransaction();
        LOG.infoEntering("transfer" );
        BigInteger usedCoin = sTx.transfer(testWallets[0], testWallets[1].getAddress(), BigInteger.valueOf(1), "HELLO");
        LOG.infoExiting();
        LOG.infoExiting();
        assertEquals(sTx.usedCoin(), usedCoin);
        assertEquals(sTx.usedCoin(), sTx.treasuryFee);
    }

    @Test
    public void deployStep() throws Exception {
        LOG.infoEntering("deployStep" );
        final String installPath = Constants.SCORE_HELLOWORLD_PATH;
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        StepTransaction sTx = new StepTransaction();
        LOG.infoEntering("deploy" );
        BigInteger usedCoin = sTx.deploy(testWallets[0], null, installPath, params);
        LOG.infoExiting();
        assertEquals(sTx.usedCoin(), usedCoin);
        if(!Utils.isAudit(iconService)) {
            assertEquals(sTx.usedCoin(), sTx.treasuryFee);
        }

        final String updatePath = Constants.SCORE_HELLOWORLD_UPDATE_PATH;
        Address socreAddr = sTx.scoreAddr;
        params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        sTx = new StepTransaction();
        LOG.infoEntering("update" );
        usedCoin = sTx.deploy(testWallets[0], socreAddr, updatePath, params);
        LOG.infoExiting();
        assertEquals(sTx.usedCoin(), usedCoin);
        if(!Utils.isAudit(iconService)) {
            assertEquals(sTx.usedCoin(), sTx.treasuryFee);
        }
        LOG.infoExiting();
    }

    @Test
    public void callStep() throws Exception {
        LOG.infoEntering("callStep");
        KeyWallet scoreOwner = testWallets[2];
        KeyWallet caller = testWallets[3];
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        LOG.infoEntering("install score" );
        Bytes txHash = Utils.deployScore(iconService, chain.networkId, scoreOwner,
                Constants.CHAINSCORE_ADDRESS, Constants.SCORE_HELLOWORLD_PATH, params);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        try {
            Utils.acceptIfAuditEnabled(iconService, chain, txHash);
        }
        catch(TransactionFailureException ex) {
            LOG.infoExiting();
            throw ex;
        }
        Address scoreAddr = new Address(result.getScoreAddress());
        StepTransaction sTx = new StepTransaction();
        LOG.infoEntering("invoke" );
        BigInteger usedCoin = sTx.call(caller, scoreAddr, "hello", null, null);
        LOG.infoExiting();
        assertEquals(sTx.usedCoin(), usedCoin);
        assertEquals(sTx.usedCoin(), sTx.treasuryFee);
        LOG.infoExiting();
    }
}
