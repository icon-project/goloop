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
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;

import static foundation.icon.test.common.Env.LOG;
import static foundation.icon.test.common.Utils.getMicroTime;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;

/*
test methods
    transferStep
    deployStep
    callStep
 */
@Tag(Constants.TAG_GOVERNANCE)
public class StepTest {
    private static KeyWallet[]testWallets;
    private static IconService iconService;
    private static Env.Chain chain;
    private static final int testWalletNum = 5;
    private static GovScore govScore;
    private static Map<String, BigInteger> originStepCosts = new HashMap<>();
    private static BigInteger originStepPrice;
    private static GovScore.Fee fee;

    private final static String STEP_DEFAULT = "default";
    private final static String STEP_INPUT = "input";
    private final static String STEP_CCREATE = "contractCreate";
    private final static String STEP_CUPDATE = "contractUpdate";
    private final static String STEP_CSET = "contractSet";
    private final static String STEP_CCALL = "contractCall";
    private final static String STEP_GET = "get";
    private final static String STEP_SET = "set";
    private final static String STEP_REPLACE = "replace";
    private final static String STEP_DELETE = "delete";

    private final static int TYPE_INT = 0;
    private final static int TYPE_STR = 1;
    private final static int TYPE_BYTES = 2;
    private final static int TYPE_ADDR = 3;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        govScore = new GovScore(iconService, chain);
        fee = govScore.getFee();
        initTransfer();
    }

    @AfterAll
    public static void destroy() throws Exception {
        govScore.setFee(fee);
    }

    public static void initTransfer() throws Exception {
        LOG.infoEntering("initTransfer");

        RpcObject rpcObject = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS,"getStepCosts", null)
                .asObject();
        for(String key : rpcObject.keySet()) {
            originStepCosts.put(key, rpcObject.getItem(key).asInteger());
        }
        originStepPrice = Utils.icxCall(iconService,
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
        Map<String, BigInteger> stepCosts = new HashMap<>();
        stepCosts.put(STEP_DEFAULT, BigInteger.valueOf(100));
        stepCosts.put(STEP_INPUT, BigInteger.valueOf(1));
        stepCosts.put(STEP_CCREATE, BigInteger.valueOf(1000));
        stepCosts.put(STEP_CSET, BigInteger.valueOf(1));
        stepCosts.put(STEP_CCALL, BigInteger.valueOf(100));
        stepCosts.put(STEP_GET, BigInteger.valueOf(200));
        stepCosts.put(STEP_SET, BigInteger.valueOf(200));
        stepCosts.put(STEP_REPLACE, BigInteger.valueOf(200));
        stepCosts.put(STEP_DELETE, BigInteger.valueOf(200));
        govScore.setStepCosts(stepCosts);
        LOG.infoExiting();
    }

    class StepTransaction {
        Map<String, BigInteger> steps = new HashMap<>();
        BigInteger stepPrice;
        BigInteger expectedStep;
        BigInteger usedCoin;
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

        void addOperation(String op, int type, String val) {
            BigInteger stepUsed = steps.get(op);
            long valSize = 0;
            switch(type) {
                case TYPE_INT:
                    int v = Integer.valueOf(val);
                    long bitLen = BigInteger.valueOf(v).bitLength();
                    valSize = bitLen / 8 + 1;
                    // int to byte
                    break;
                case TYPE_BYTES:
                    if (!val.startsWith("0x")) {
                        // error
                    }
                    valSize = (val.length() - 2) / 2;
                    break;
                case TYPE_ADDR:
                    valSize = 21; // address should be 21 bytes
                    break;
                default:
                    valSize = val.length();
                    break;
            }
            LOG.info("addOperation val : " + val + ", valSize : " + valSize);
            expectedStep = expectedStep.add(BigInteger.valueOf(valSize).multiply(stepUsed));
        }

        BigInteger estimatedCoin() {
            return expectedStep.multiply(stepPrice);
        }

        BigInteger calcTransactionStep(Transaction tx) {
            // default + input * data
            BigInteger stepUsed = steps.get(STEP_DEFAULT);
            if(tx.getDataType().equals("message")) {
                // tx.getData() returns message with no quotes
                long dataSize = tx.getData().asString().getBytes(StandardCharsets.UTF_8).length + 2;
                stepUsed = stepUsed.add(BigInteger.valueOf(dataSize).multiply(steps.get(STEP_INPUT)));
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
                            dataLen += paramObj.getItem(param).asString().getBytes(StandardCharsets.UTF_8).length;
                            dataLen += param.getBytes(StandardCharsets.UTF_8).length;
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
                stepUsed = calcTransactionStep(tx);
            }
            BigInteger codeLen = BigInteger.valueOf(content.length);
            if(update) {
                stepUsed = steps.get(STEP_CUPDATE).add(stepUsed);
            }
            else {
                stepUsed = steps.get(STEP_CCREATE).add(stepUsed);
            }
            stepUsed = stepUsed.add(steps.get(STEP_CSET).multiply(codeLen));
            return stepUsed;
        }

        BigInteger calcCallStep(Transaction tx) {
            BigInteger stepUsed = calcTransactionStep(tx);
            // contractCall
            stepUsed = steps.get(STEP_CCALL).add(stepUsed);
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
            this.expectedStep = calcTransactionStep(transaction);

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

            Transaction transaction = builder.deploy(Constants.CONTENT_TYPE_ZIP, content)
                    .params(params).build();
            this.expectedStep = calcDeployStep(transaction, content, to != null);

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

        BigInteger getSpentCoinByLastTx() {
            return usedCoin;
        }

        BigInteger call(KeyWallet from, Address to, String method, RpcObject params, BigInteger value, BigInteger stepLimit) throws Exception {
            BigInteger prevTresury = iconService.getBalance(Constants.TREASURY_ADDRESS).execute();
            BigInteger prevBal = iconService.getBalance(from.getAddress()).execute();
            TransactionBuilder.Builder builder = TransactionBuilder.newBuilder()
                    .nid(BigInteger.valueOf(chain.networkId))
                    .from(from.getAddress())
                    .to(to)
                    .stepLimit(stepLimit);

            if ((value != null) && value.bitLength() != 0) {
                builder = builder.value(value);
            }

            Transaction transaction;
            if (params != null) {
                transaction = builder.call(method).params(params).build();
            } else {
                transaction = builder.call(method).build();
            }
            this.expectedStep = calcCallStep(transaction);

            Bytes txHash = iconService.sendTransaction(new SignedTransaction(transaction, from)).execute();
            TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
            BigInteger bal = iconService.getBalance(from.getAddress()).execute();
            BigInteger treasury = iconService.getBalance(Constants.TREASURY_ADDRESS).execute();
            treasuryFee = treasury.subtract(prevTresury);

            usedCoin = prevBal.subtract(bal);
            if(Constants.STATUS_SUCCESS.compareTo(result.getStatus()) != 0) {
                LOG.info("prop : " + result.getProperties());
                throw new TransactionFailureException(result.getFailure());
            }
            return usedCoin;
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
        assertEquals(sTx.estimatedCoin(), usedCoin);
        assertEquals(sTx.estimatedCoin(), sTx.treasuryFee);
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
        assertEquals(sTx.estimatedCoin(), usedCoin);
        if(!Utils.isAudit(iconService)) {
            assertEquals(sTx.estimatedCoin(), sTx.treasuryFee);
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
        assertEquals(sTx.estimatedCoin(), usedCoin);
        if(!Utils.isAudit(iconService)) {
            assertEquals(sTx.estimatedCoin(), sTx.treasuryFee);
        }
        LOG.infoExiting();
    }

    enum VarTest {
        VAR_SET("setToVar", STEP_SET),
        VAR_GET("getFromVar", STEP_GET),
        VAR_REPLACE("setToVar", STEP_REPLACE),
        VAR_EDGE("setToVar", STEP_REPLACE),
        VAR_DELETE("delFromVar", STEP_DELETE);

        String [][]params;
        String method;
        String op;

        VarTest(String method, String op) {
            this.method = method;
            this.op = op;
        }

        public RpcObject getParams(int type) {
            RpcObject.Builder paramObj = new RpcObject.Builder();
            // set, get, replace, edge, delete
            if(params == null) {
                paramObj = paramObj.put("type", new RpcValue(BigInteger.valueOf(type)));
            }
            else {
                for(int i = 0; i < type + 1; i++) {
                    paramObj = paramObj.put(params[i][0], new RpcValue(params[i][1]));
                }
            }
            return paramObj.build();
        }
    }

    @Test
    public void varDb() throws Exception {
        LOG.infoEntering("callStep");
        KeyWallet scoreOwner = testWallets[2];
        KeyWallet caller = testWallets[3];
        LOG.infoEntering("install score" );
        Bytes txHash = Utils.deployScore(iconService, chain.networkId, scoreOwner,
                Constants.CHAINSCORE_ADDRESS, Constants.SCORE_DB_STEP_PATH, null);
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

        String [][]params = {
                {"v_int", "128"},
                {"v_str", "LOOP"},
                {"v_bytes", new Bytes("BYTES PARAM".getBytes()).toString()},
                {"v_addr", testWallets[0].getAddress().toString()},
        };

        String bp = new Bytes("BYTES PARAM".getBytes()).toString();
        byte[] b = bp.substring("0x".length()).getBytes();
        LOG.info("BYTES PARAM (" + new String(b) + ")");
        LOG.info("BYTES PARAM bp (" + bp + ")");
        LOG.info("BYTES PARAM bytearray (" + Arrays.toString("BYTES PARAM".getBytes()) + ")");
        BigInteger[] edgeLimit = new BigInteger[4];

        for(VarTest test : VarTest.values()) {
            if(test == VarTest.VAR_EDGE) {
                final String [][]edgeParams = {
                        {"v_int", "821"},
                        {"v_str", "POOL"},
                        {"v_bytes", new Bytes("PARAM BYTES".getBytes()).toString()},
                        {"v_addr", testWallets[1].getAddress().toString()},
                };
                test.params = edgeParams;
            }
            else if(test != VarTest.VAR_DELETE && test != VarTest.VAR_GET) {
                test.params = params;
            }
            for(int i = 0; i < 4; i++) {
                LOG.infoEntering("invoke (" + test + ") method : " + test.method + ", param " + params[i][0] + ", val : " + params[i][1]);
                BigInteger stepLimit = test == VarTest.VAR_EDGE ? edgeLimit[i].subtract(BigInteger.ONE) : BigInteger.valueOf(10000);
                BigInteger usedCoin = null;
                try {
                    usedCoin = sTx.call(caller, scoreAddr, test.method, test.getParams(i), null, stepLimit);
                    assertNotEquals(test, VarTest.VAR_EDGE);
                    sTx.addOperation(test.op, i, params[i][1]);
                    // TYPE_INT, TYPE_STR, TYPE_BYTES, TYPE_ADDR
                    assertEquals(sTx.estimatedCoin(), usedCoin);
                    assertEquals(sTx.estimatedCoin(), sTx.treasuryFee);
                }
                catch (TransactionFailureException ex) {
                    if(test != VarTest.VAR_EDGE) {
                        throw ex;
                    }
                    usedCoin = sTx.getSpentCoinByLastTx();

                    RpcObject callParam = new RpcObject.Builder()
                            .put("type", new RpcValue(BigInteger.valueOf(i)))
                            .build();
                    String dbVal = Utils.icxCall(iconService,
                            scoreAddr, "readFromVar", callParam).asString();
                    LOG.info("dvVal[" + i + "] : " + dbVal);
                    if(i == 0) {
                        BigInteger v = new BigInteger(dbVal.substring("0x".length()), 16);
                        LOG.info("v = " + v);
                    } else if(i == 2) {
                        LOG.info("byte[] : " + Arrays.toString(dbVal.getBytes()));
                    }

                    // TYPE_INT, TYPE_STR, TYPE_BYTES, TYPE_ADDR
                    assertEquals(stepLimit, usedCoin);
                    assertEquals(stepLimit, sTx.treasuryFee);
                }
                if(test == VarTest.VAR_REPLACE) {
                    edgeLimit[i] = sTx.estimatedCoin();
                }
                LOG.infoExiting();
            }
        }
        LOG.infoExiting();
    }
}
