package foundation.icon.test.cases;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.HelloWorld;
import org.junit.AfterClass;
import org.junit.BeforeClass;
import org.junit.Ignore;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.Parameterized;
import java.math.BigInteger;
import java.util.*;

import static org.junit.Assert.*;

@RunWith(Parameterized.class)
public class ChainScoreTest{
    public Address toAddr;
    private static Env.Chain chain;
    private static IconService iconService;

    public ChainScoreTest(Address input, String toScore){
        toAddr = input;
    }

    @Parameterized.Parameters(name = "{1}")
    public static Iterable<Object[]> initInput() {
        return Arrays.asList(new Object[][] {
                {Constants.CHAINSCORE_ADDRESS, "ToChainScore"},
                {Constants.GOV_ADDRESS, "ToGovernanceScore"},
        });
    }

    private static KeyWallet helloWorldOwner;
    private static KeyWallet[]testWallets;
    private static final int testWalletNum = 3;
    private static HelloWorld helloWorld;

    @BeforeClass
    public static void init() throws Exception {
        Env.Node node = Env.getInstance().nodes[0];
        chain = node.channels[0].chain;
        iconService = new IconService(new HttpProvider(node.channels[0].getAPIUrl(Env.testApiVer)));
        testWallets = new KeyWallet[testWalletNum];
        initChainScore();
    }

    static void initChainScore() throws Exception {
        String []cTypes = {"invoke", "query"};
        for(String cType : cTypes) {
            RpcObject.Builder builder = new RpcObject.Builder();
            builder.put("contextType", new RpcValue(cType));
            builder.put("limit", new RpcValue("100000"));
            TransactionResult result = Utils.sendTransactionWithCall(iconService, chain.networkId,
                    KeyWallet.create(), Constants.GOV_ADDRESS, "setMaxStepLimit", builder.build(), 0);
            if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
                throw new Exception();
            }
        }

        long value = 999999999;
        Bytes[]txHash = new Bytes[testWalletNum + 1];
        for (int i = 0; i < testWalletNum + 1; i++) {
            KeyWallet wallet = KeyWallet.create();
            try {
                txHash[i] = Utils.transfer(iconService, chain.networkId, chain.godWallet
                        , wallet.getAddress(), value);
            } catch (Exception ex) {
                System.out.println("Failed to transfer");
                throw ex;
            }
            if (i < testWalletNum) {
                testWallets[i] = wallet;
            } else {
                helloWorldOwner = wallet;
            }
        }

        helloWorld = HelloWorld.install(iconService, chain, helloWorldOwner);
    }

    @AfterClass
    public static void destroy() throws Exception {
        String []cTypes = {"invoke", "query"};
        for(String cType : cTypes) {
            RpcObject.Builder builder = new RpcObject.Builder();
            builder.put("contextType", new RpcValue(cType));
            builder.put("limit", new RpcValue("0"));
            TransactionResult result = Utils.sendTransactionWithCall(iconService, chain.networkId,
                    KeyWallet.create(), Constants.GOV_ADDRESS, "setMaxStepLimit", builder.build(), 0);
            if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
                throw new Exception();
            }
        }
    }

    public TransactionResult sendGovCallTx(KeyWallet fromWallet, String method, RpcObject params) throws Exception {
        TransactionResult result = Utils.sendTransactionWithCall(iconService, chain.networkId,
                fromWallet, this.toAddr, method, params, 0);
        if ((!this.toAddr.equals(Constants.GOV_ADDRESS) && Constants.STATUS_SUCCESS.equals(result.getStatus())) ||
                (this.toAddr.equals(Constants.GOV_ADDRESS) && !Constants.STATUS_SUCCESS.equals(result.getStatus()))) {
            throw new Exception();
        }
        return result;
    }

    @Test
    public void disableEnableScore() throws Exception{
        if(this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
            return;
        }

        KeyWallet leaserWallet = testWallets[0];
        String []methods = {"disableScore", "enableScore"};
        KeyWallet[]fromWallets = {
                leaserWallet,
                helloWorldOwner
        };

        TransactionResult result = helloWorld.invokeHello(leaserWallet);
        assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);

        for (String method : methods) {
            for (KeyWallet wallet : fromWallets) {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("address", new RpcValue(helloWorld.getAddress()));
                result = Utils.sendTransactionWithCall(iconService,
                        chain.networkId, wallet, Constants.CHAINSCORE_ADDRESS, method, builder.build(), 0);
                if ((Constants.STATUS_SUCCESS.equals(result.getStatus()) && wallet != helloWorldOwner) ||
                        (!Constants.STATUS_SUCCESS.equals(result.getStatus()) && wallet == helloWorldOwner)) {
                    throw new Exception();
                }

                try {
                    result = helloWorld.invokeHello(leaserWallet);
                    if(result.getStatus().compareTo(Constants.STATUS_SUCCESS) != 0) {
                        throw new Exception();
                    }
                }
                catch (ResultTimeoutException ex) {
                    if((wallet == helloWorldOwner && method.compareTo("disableScore") == 0)
                            || (wallet != helloWorldOwner && method.compareTo("enableScore") == 0)){
                        continue;
                    } else {
                        throw ex;
                    }
                }
            }
        }
    }

    @Test
    public void setRevision() throws Exception{
        KeyWallet wallet = KeyWallet.create();

        RpcItem item = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getRevision", null);

        BigInteger revision = item.asInteger();
        revision = revision.add(BigInteger.valueOf(100));
        RpcObject.Builder builder = new RpcObject.Builder();
        builder.put("code", new RpcValue(revision));
        sendGovCallTx(wallet, "setRevision", builder.build());

        BigInteger newRevision = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getRevision", null).asInteger();
        if (!revision.equals(newRevision)) {
            if(!this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
                throw new Exception("Failed to set Revision");
            }
        }

        if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
            for (int i = 0; i < 2; i++) {
                // It allows to set a greater value than the current. test with same value & less value.
                BigInteger wrongRevision = revision.subtract(BigInteger.valueOf(i));
                builder = new RpcObject.Builder();
                builder.put("code", new RpcValue(wrongRevision));
                TransactionResult result = Utils.sendTransactionWithCall(iconService, chain.networkId,
                        wallet, Constants.GOV_ADDRESS, "setRevision", builder.build(), 0);
                if (result.getStatus().compareTo(Constants.STATUS_SUCCESS) == 0) {
                    throw new Exception();
                }
            }
        }
    }

    @Ignore
    @Test
    public void acceptScore() throws Exception{
        if (!Utils.isAudit(iconService)) {
            return;
        }
        Bytes txHash = Utils.deployScore(iconService, chain.networkId, helloWorldOwner, Constants.CHAINSCORE_ADDRESS,
                Constants.SCORE_ROOT + "helloWorld.zip", null, -1);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, 5000);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new Exception();
        }
        Address addr = new Address(result.getScoreAddress());
        KeyWallet tmpWallet = KeyWallet.create();
        boolean []list = {false, true};
        for(boolean accepted : list) {
            try {
                Utils.sendTransactionWithCall(iconService, chain.networkId,
                        tmpWallet, addr, "hello", null, 0);
                if(!accepted) {
                    throw new Exception();
                }
            }
            catch(ResultTimeoutException ex) {
                if(accepted) {
                    throw new Exception();
                }
            }
            if(!accepted) {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("txHash", new RpcValue(txHash.toHexString(true)));
                Utils.sendTransactionWithCall(iconService, chain.networkId,
                        tmpWallet, this.toAddr, "acceptScore", builder.build(), 0);
            }
        }
    }

    @Ignore
    @Test
    public void rejectScore() throws Exception {
        if (!Utils.isAudit(iconService)) {
            return;
        }
        Bytes txHash = Utils.deployScore(iconService, chain.networkId, helloWorldOwner, Constants.CHAINSCORE_ADDRESS,
                Constants.SCORE_ROOT + "helloWorld.zip", null, -1);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, 5000);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new Exception();
        }
        Address addr = new Address(result.getScoreAddress());
        KeyWallet tmpWallet = KeyWallet.create();
        boolean []list = {false, true};
        for(boolean rejected : list) {
            try {
                Utils.sendTransactionWithCall(iconService, chain.networkId,
                        tmpWallet, addr, "hello", null, 0);
                throw new Exception();
            }
            catch(ResultTimeoutException ex) {
            }
            if(!rejected) {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("txHash", new RpcValue(txHash.toHexString(true)));
                Utils.sendTransactionWithCall(iconService, chain.networkId,
                        tmpWallet, this.toAddr, "rejectScore", builder.build(), 0);
            } else {
                // TODO get scoreStatus
            }
        }
    }

    // test block / unblock score
    @Test
    public void blockUnblockScore() throws Exception {
        KeyWallet wallet = KeyWallet.create();

        TransactionResult result = helloWorld.invokeHello(wallet);
        if(result.getStatus().compareTo(Constants.STATUS_SUCCESS) != 0) {
            throw new Exception();
        }

        //check blocked is 0x0 (false)
        RpcObject.Builder builder = new RpcObject.Builder();
        builder.put("address", new RpcValue(helloWorld.getAddress()));
        RpcObject rpcObject = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getScoreStatus", builder.build()).asObject();
        RpcItem blocked = rpcObject.getItem("blocked");
        if (blocked == null || blocked.asBoolean()) {
            System.out.println("blocked = " + blocked);
            throw new Exception();
        }

        String []methods = {"blockScore", "unblockScore"};
        for (String method : methods) {
            sendGovCallTx(wallet, method, builder.build());

            rpcObject = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS, "getScoreStatus", builder.build()).asObject();
            blocked = rpcObject.getItem("blocked");
            if (blocked == null) {
                throw new Exception();
            }
            if(blocked.asBoolean()) {
                if(!this.toAddr.equals(Constants.GOV_ADDRESS) || !method.equals("blockScore")) {
                    throw new Exception();
                }
            }

            try {
                result = helloWorld.invokeHello(wallet);
                if(result.getStatus().compareTo(Constants.STATUS_SUCCESS) != 0) {
                    throw new Exception();
                }
            }
            catch (ResultTimeoutException ex) {
                if(!this.toAddr.equals(Constants.GOV_ADDRESS) || method.compareTo("blockScore") != 0) {
                    throw ex;
                }
            }
        }
    }

    @Test
    public void setStepPrice() throws Exception {
        KeyWallet wallet = testWallets[0];
        BigInteger originPrice = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getStepPrice", null).asInteger();
        BigInteger newPrice = originPrice.add(BigInteger.valueOf(10));
        BigInteger []stepPrices = new BigInteger[] {newPrice, originPrice};
        for(BigInteger price : stepPrices) {
            RpcObject.Builder builder = new RpcObject.Builder();
            builder.put("price", new RpcValue(price.toString()));
            sendGovCallTx(wallet, "setStepPrice", builder.build());
            BigInteger cmp;
            if (this.toAddr.equals(Constants.GOV_ADDRESS)) {
                cmp = price;
            }
            else {
                cmp = originPrice;
            }

            BigInteger queryPrice = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS, "getStepPrice", null).asInteger();
            if(queryPrice.compareTo(cmp) != 0) {
                throw new Exception();
            }
        }
    }

    @Test
    public void setStepCost() throws Exception{
        KeyWallet wallet = testWallets[0];
        RpcItem stepCosts = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getStepCosts", null);
        Bytes []txHashList = new Bytes[GovScore.stepCostTypes.length];

        Map<String, BigInteger> originMap = new HashMap<>();
        Map<String, BigInteger> newStepCostsMap = new HashMap<>();
        RpcObject rpcObject = stepCosts.asObject();
        long cnt = 1;
        for(int i = 0; i < GovScore.stepCostTypes.length; i++) {
            String type = GovScore.stepCostTypes[i];
            BigInteger oCost = rpcObject.getItem(type).asInteger();
            originMap.put(type, oCost);

            BigInteger newCost = oCost.add(BigInteger.valueOf(cnt));
            newStepCostsMap.put(type, newCost);
            cnt += 1;
            try {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("type", new RpcValue(type));
                builder.put("cost", new RpcValue(newCost.toString()));
                txHashList[i] = Utils.sendTransactionWithCall(iconService, chain.networkId,
                        wallet, this.toAddr, "setStepCost", builder.build(), 0, false);
            }
            catch(Exception ex) {
                throw ex;
            }
        }

        for(Bytes txHash : txHashList) {
            TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
            if ((!this.toAddr.equals(Constants.GOV_ADDRESS) && Constants.STATUS_SUCCESS.equals(result.getStatus())) ||
                    (this.toAddr.equals(Constants.GOV_ADDRESS) && !Constants.STATUS_SUCCESS.equals(result.getStatus()))) {
                throw new Exception();
            }
        }

        Map<String, BigInteger> cmpCosts;

        if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
            cmpCosts = newStepCostsMap;
        }
        else {
            cmpCosts = originMap;
        }

        rpcObject = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getStepCosts", null).asObject();
        for (String type : GovScore.stepCostTypes) {
            if (cmpCosts.get(type).compareTo(rpcObject.getItem(type).asInteger()) != 0) {
                throw new Exception();
            }
        }

        if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
            // rollback
            txHashList = new Bytes[GovScore.stepCostTypes.length];
            for(int i = 0; i < GovScore.stepCostTypes.length; i++) {
                String type = GovScore.stepCostTypes[i];
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("type", new RpcValue(type));
                builder.put("cost", new RpcValue(originMap.get(type)));
                txHashList[i] = Utils.sendTransactionWithCall(iconService, chain.networkId,
                        wallet, this.toAddr, "setStepCost", builder.build(), 0, false);
            }

            for(Bytes txHash : txHashList) {
                TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
                if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
                    throw new Exception();
                }
            }

            rpcObject = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS, "getStepCosts", null).asObject();
            for (String type : GovScore.stepCostTypes) {
                if (originMap.get(type).compareTo(rpcObject.getItem(type).asInteger()) != 0) {
                    throw new Exception();
                }
            }
        }
    }

    @Test
    public void setMaxStepLimit() throws Exception {
        KeyWallet wallet = KeyWallet.create();
        String []contextTypes = {"invoke", "query"};
        for(String type : contextTypes) {
            RpcObject params = new RpcObject.Builder()
                    .put("contextType", new RpcValue(type))
                    .build();
            BigInteger originLimit = Utils.icxCall(iconService, Constants.CHAINSCORE_ADDRESS,
                    "getMaxStepLimit", params).asInteger();

            BigInteger newLimit = originLimit.add(BigInteger.valueOf(10));
            BigInteger []limits = new BigInteger[] {
                    newLimit, originLimit
            };
            for(BigInteger limit : limits) {
                RpcObject.Builder builder = new RpcObject.Builder();
                builder.put("contextType", new RpcValue(type));
                builder.put("limit", new RpcValue(limit));
                sendGovCallTx(wallet, "setMaxStepLimit", builder.build());

                BigInteger cmp;
                if (this.toAddr.equals(Constants.GOV_ADDRESS)) {
                    cmp = limit;
                }
                else {
                    cmp = originLimit;
                }
                BigInteger queryLimit = Utils.icxCall(iconService,
                        Constants.CHAINSCORE_ADDRESS,"getMaxStepLimit", params).asInteger();
                if(queryLimit.compareTo(cmp) != 0) {
                    throw new Exception();
                }
            }
        }
    }

    // TBD : setValidator API
//    public void setValidators() {}
    @Ignore
    @Test
    public void grantRevokeValidator() throws Exception {
        KeyWallet wallet = testWallets[0];
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(wallet.getAddress()))
                .build();
        RpcItem item = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getValidators", params);
        RpcArray rpcArray = item.asArray();
        for(int i = 0; i < rpcArray.size(); i++) {
            if(rpcArray.get(i).asAddress().equals(wallet)) {
                throw new Exception();
            }
        }
        String []methods = {"grantValidator", "revokeValidator"};
        for (String method : methods) {
            RpcObject.Builder builder = new RpcObject.Builder();
            builder.put("address", new RpcValue(wallet.getAddress().toString()));
            sendGovCallTx(wallet, method, builder.build());

            item = Utils.icxCall(iconService, Constants.CHAINSCORE_ADDRESS,
                    "getValidators", params);
            boolean bFound = false;
            rpcArray = item.asArray();
            for(int i = 0; i < rpcArray.size(); i++) {
                if(rpcArray.get(i).asAddress().equals(wallet.getAddress())) {
                    bFound = true;
                    break;
                }
            }

            if(this.toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
                if(bFound == true) {
                    throw new Exception();
                }
            } else {
                if(method.compareTo("grantValidator") == 0) {
                    if(bFound == false) {
                        throw new Exception();
                    }
                }
            }
        }
    }

    @Test
    public void addRemoveMember() throws Exception{
        KeyWallet wallet = testWallets[0];
        RpcItem item = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS,"getMembers", null);

        RpcArray rpcArray = item.asArray();
        for(int i = 0; i < rpcArray.size(); i++) {
            if(rpcArray.get(i).asAddress().equals(wallet)) {
                throw new Exception();
            }
        }
        String []methods = {"addMember", "removeMember"};
        for (String method : methods) {
            RpcObject.Builder builder = new RpcObject.Builder();
            builder.put("address", new RpcValue(wallet.getAddress().toString()));
            sendGovCallTx(wallet, method, builder.build());
            item = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS, "getMembers", null);

            boolean bFound = false;
            rpcArray = item.asArray();
            for(int i = 0; i < rpcArray.size(); i++) {
                if(rpcArray.get(i).asAddress().equals(wallet.getAddress())) {
                    bFound = true;
                    break;
                }
            }

            if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
                if(method.compareTo("addMember") == 0 && !bFound) {
                    throw new Exception();
                }
                else if(method.compareTo("removeMember") == 0 && bFound) {
                    throw new Exception();
                }
            } else {
                if(bFound) {
                    throw new Exception();
                }
            }
        }
    }

    @Test
    public void addRemoveDeployer() throws Exception {
        KeyWallet wallet = testWallets[0];
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(wallet.getAddress()))
                .build();
        RpcItem item = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS,"isDeployer", params);
        if (item.asBoolean()) {
            throw new Exception();
        }
        String []methods = {"addDeployer", "removeDeployer"};
        for (String method : methods) {
            RpcObject.Builder builder = new RpcObject.Builder();
            builder.put("address", new RpcValue(wallet.getAddress().toString()));
            sendGovCallTx(wallet, method, builder.build());
            item = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS,"isDeployer", params);
           if(item.asBoolean() && (!this.toAddr.equals(Constants.GOV_ADDRESS)
                   && method.compareTo("addDeployer") != 0)) {
               throw new Exception();
           }
        }
    }

    public void addLicense() {}

    public void removeLicense( ) {}

}
