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
import foundation.icon.test.common.*;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.score.HelloWorld;
import org.junit.AfterClass;
import org.junit.BeforeClass;
import org.junit.Ignore;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.Parameterized;

import java.math.BigInteger;
import java.util.*;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.Assert.*;

@RunWith(Parameterized.class)
public class ChainScoreTest{
    public Address toAddr;
    private static Env.Chain chain;
    private static IconService iconService;
    private static String testName;

    public ChainScoreTest(Address input, String name){
        toAddr = input;
        testName = name;
    }

    @Parameterized.Parameters(name = "{1}")
    public static Iterable<Object[]> initInput() {
        return Arrays.asList(new Object[][] {
                {Constants.CHAINSCORE_ADDRESS, "To_ChainScore"},
                {Constants.GOV_ADDRESS, "To_GovernanceScore"},
        });
    }

    private static KeyWallet helloWorldOwner;
    private static KeyWallet[]testWallets;
    private static final int testWalletNum = 3;
    private static HelloWorld helloWorld;

    @BeforeClass
    public static void init() {
        Env.Node node = Env.nodes[0];
        chain = node.channels[0].chain;
        iconService = new IconService(new HttpProvider(node.channels[0].getAPIUrl(Env.testApiVer)));
        try {
            initChainScore();
        }
        catch (Exception ex) {
            ex.printStackTrace();
            fail();
        }
    }

    static void initChainScore() throws Exception {
        Utils.transferAndCheck(iconService, chain, chain.godWallet, chain.governorWallet.getAddress(), Constants.DEFAULT_BALANCE);

        testWallets = new KeyWallet[testWalletNum];
        Address []testAddrs = new Address[testWalletNum];
        for(int i = 0; i < testWalletNum; i++) {
            KeyWallet wallet = KeyWallet.create();
            testWallets[i] = wallet;
            testAddrs[i] = wallet.getAddress();
        }
        Utils.transferAndCheck(iconService, chain, chain.godWallet, testAddrs, Constants.DEFAULT_BALANCE);

        helloWorldOwner = KeyWallet.create();
        Utils.transferAndCheck(iconService, chain, chain.godWallet, helloWorldOwner.getAddress(), Constants.DEFAULT_BALANCE);
        helloWorld = HelloWorld.install(iconService, chain, helloWorldOwner);

        String []cTypes = {"invoke", "query"};
        for(String cType : cTypes) {
            RpcObject.Builder builder = new RpcObject.Builder();
            builder.put("contextType", new RpcValue(cType));
            builder.put("limit", new RpcValue("100000"));
            TransactionResult result = Utils.sendTransactionWithCall(iconService, chain.networkId,
                    KeyWallet.create(), Constants.GOV_ADDRESS, "setMaxStepLimit", builder.build(), 0);
            Utils.assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        }
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

    public TransactionResult sendGovCallTx(String method, RpcObject params) throws Exception {
        TransactionResult result;
        try {
            result = Utils.sendTransactionWithCall(iconService, chain.networkId,
                    chain.governorWallet, toAddr, method, params, 0);
        }
        catch (Exception ex) {
            ex.printStackTrace();
            throw ex;
        }
        if(toAddr.equals(Constants.GOV_ADDRESS)) {
            Utils.assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        }
        else {
            Utils.assertEquals(Constants.STATUS_FAIL, result.getStatus());
        }
        return result;
    }

    @Test
    public void disableEnableScore() throws Exception{
        if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
            return;
        }

        LOG.infoEntering( "disableEnableScore");
        KeyWallet notOwner = testWallets[0];
        KeyWallet[]fromWallets = {
                notOwner,
                helloWorldOwner
        };
        KeyWallet caller = testWallets[1];
        TransactionResult result = helloWorld.invokeHello(caller);
        assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);

        for (String method : new String[]{"disableScore", "enableScore"}) {
            for (KeyWallet from : fromWallets) {
                RpcObject params = new RpcObject.Builder()
                        .put("address", new RpcValue(helloWorld.getAddress()))
                        .build();
                boolean prevDisabled = Utils.icxCall(iconService,
                        Constants.CHAINSCORE_ADDRESS, "getScoreStatus", params).asObject().getItem("disabled").asBoolean();
                LOG.infoEntering("method[" + method + "], isOwner[" + (from == helloWorldOwner) + "]");
                result = Utils.sendTransactionWithCall(iconService,
                        chain.networkId, from, Constants.CHAINSCORE_ADDRESS, method, params);
                LOG.infoExiting();
                Utils.assertEquals(from == helloWorldOwner ? Constants.STATUS_SUCCESS : Constants.STATUS_FAIL, result.getStatus());

                boolean disabled = Utils.icxCall(iconService,
                        Constants.CHAINSCORE_ADDRESS, "getScoreStatus", params).asObject().getItem("disabled").asBoolean();
                assertTrue(from == helloWorldOwner ? prevDisabled != disabled : prevDisabled == disabled);

                try {
                    LOG.infoEntering("method[hello], disabled[" + disabled + "]");
                    result = helloWorld.invokeHello(caller);
                    Utils.assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
                }
                catch (ResultTimeoutException ex) {
                    LOG.info("FAIL to get result by tx");
                    if(from == helloWorldOwner) {
                        Utils.assertEquals("disableScore", method);
                    }
                }
                LOG.infoExiting();
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void setRevision() throws Exception{
        LOG.infoEntering("setRevision");
        BigInteger oldRevision = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getRevision", null).asInteger();
        BigInteger newRevision = oldRevision.add(BigInteger.valueOf(100));
        RpcObject params = new RpcObject.Builder()
                .put("code", new RpcValue(newRevision))
                .build();
        LOG.infoEntering("method[setRevision] OLD[" + oldRevision + "], NEW[" + newRevision + "]");
        sendGovCallTx( "setRevision", params);
        LOG.infoExiting();

        BigInteger revision = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getRevision", null).asInteger();
        if(toAddr.equals(Constants.GOV_ADDRESS)) {
            Utils.assertEquals(newRevision, revision);
            for (int i = 0; i < 2; i++) {
                // It allows to set a greater value than the current. test with same value & less value.
                BigInteger wrongRevision = revision.subtract(BigInteger.valueOf(i));
                params = new RpcObject.Builder()
                        .put("code", new RpcValue(wrongRevision))
                        .build();
                LOG.infoEntering("method[setRevision] TO GOVERNANCE, OLD[" + revision + "], NEW[" + wrongRevision + "]");
                TransactionResult result =
                        Utils.sendTransactionWithCall(iconService, chain.networkId,
                        chain.governorWallet, Constants.GOV_ADDRESS, "setRevision", params);
                LOG.infoExiting();
                Utils.assertEquals(Constants.STATUS_FAIL, result.getStatus());
                newRevision = Utils.icxCall(iconService,
                        Constants.CHAINSCORE_ADDRESS, "getRevision", null).asInteger();
                Utils.assertEquals(revision, newRevision);
            }
        }
        else {
            Utils.assertEquals(oldRevision, revision);
        }
        LOG.infoExiting();
    }

    @Test
    public void acceptScore() throws Exception {
        if (!Utils.isAudit(iconService)) {
            return;
        }
        LOG.infoEntering("acceptScore");
        KeyWallet owner = KeyWallet.create();
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        Bytes txHash = Utils.deployScore(iconService, chain.networkId, owner, Constants.CHAINSCORE_ADDRESS,
                Constants.SCORE_HELLOWORLD_PATH, params, 2000);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, 5000);
        assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);

        Address scoreAddr = new Address(result.getScoreAddress());
        KeyWallet caller = KeyWallet.create();
        String []expectedStatus;
        if(toAddr == Constants.GOV_ADDRESS) {
            expectedStatus = new String[]{Constants.SCORE_STATUS_PENDING, Constants.SCORE_STATUS_ACTIVE};
        }
        else {
            expectedStatus = new String[]{Constants.SCORE_STATUS_PENDING, Constants.SCORE_STATUS_PENDING};
        }
        String expectedItem = "next";
        for(String expect : expectedStatus) {
            params = new RpcObject.Builder()
                    .put("address", new RpcValue(scoreAddr))
                    .build();
            RpcObject rpcObject = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS, "getScoreStatus", params).asObject();
            RpcObject object = rpcObject.getItem(expectedItem).asObject();
            assertNotNull(object);
            Utils.assertEquals(expect, object.getItem("status").asString());

            try {
                Utils.sendTransactionWithCall(iconService, chain.networkId,
                        caller, scoreAddr, "hello", null, 0);
                Utils.assertEquals(Constants.SCORE_STATUS_ACTIVE, expect);
            }
            catch(ResultTimeoutException ex) {
                LOG.info("FAIL to get result by tx");
                Utils.assertEquals(Constants.SCORE_STATUS_PENDING, expect);
            }
            if(expect == Constants.SCORE_STATUS_PENDING) {
                params = new RpcObject.Builder()
                        .put("txHash", new RpcValue(txHash))
                        .build();
                LOG.infoEntering( "accept score");
                TransactionResult acceptResult =
                        Utils.sendTransactionWithCall(iconService, chain.networkId,
                                chain.governorWallet, toAddr, "acceptScore", params, 0);
                if(toAddr == Constants.GOV_ADDRESS) {
                    Utils.assertEquals(Constants.STATUS_SUCCESS, acceptResult.getStatus());
                    expectedItem = "current";
                }
                else {
                    Utils.assertEquals(Constants.STATUS_FAIL, acceptResult.getStatus());
                }
                LOG.infoExiting();
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void rejectScore() throws Exception {
        if (!Utils.isAudit(iconService)) {
            return;
        }
        LOG.infoEntering("rejectScore");
        KeyWallet owner = KeyWallet.create();
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        Bytes txHash = Utils.deployScore(iconService, chain.networkId, owner,
                Constants.CHAINSCORE_ADDRESS, Constants.SCORE_HELLOWORLD_PATH, params, 2000);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, 5000);
        assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);

        Address scoreAddr = new Address(result.getScoreAddress());
        KeyWallet caller = KeyWallet.create();
        String []expectedStatus;
        if(toAddr == Constants.GOV_ADDRESS) {
            expectedStatus = new String[]{Constants.SCORE_STATUS_PENDING, Constants.SCORE_STATUS_REJECT};
        }
        else {
            expectedStatus = new String[]{Constants.SCORE_STATUS_PENDING, Constants.SCORE_STATUS_PENDING};
        }

        for(String expect : expectedStatus) {
            params = new RpcObject.Builder()
                    .put("address", new RpcValue(scoreAddr))
                    .build();
            RpcObject rpcObject =
                    Utils.icxCall(iconService, Constants.CHAINSCORE_ADDRESS, "getScoreStatus", params).asObject();
            RpcObject object = rpcObject.getItem("next").asObject();
            assertNotNull(object);

            String status = object.getItem("status").asString();
            assertEquals(status, expect);

            try {
                Utils.sendTransactionWithCall(iconService, chain.networkId, caller, scoreAddr, "hello", null, 0);
                // cannot reach here
                fail();
            }
            catch(ResultTimeoutException ex) {
                LOG.info("FAIL to get result by tx");
                //success
            }
            if(expect == Constants.SCORE_STATUS_PENDING) {
                LOG.infoEntering("reject score");
                params = new RpcObject.Builder()
                        .put("txHash", new RpcValue(txHash))
                        .build();
                TransactionResult acceptResult =
                        Utils.sendTransactionWithCall(iconService, chain.networkId,
                                chain.governorWallet, toAddr, "rejectScore", params, 0);
                if(toAddr == Constants.GOV_ADDRESS) {
                    assertEquals(acceptResult.getStatus(), Constants.STATUS_SUCCESS);
                }
                else {
                    assertEquals(acceptResult.getStatus(), Constants.STATUS_FAIL);
                }
                LOG.infoExiting();
            }
        }
        LOG.infoExiting();
    }

    // test block / unblock score
    @Test
    public void blockUnblockScore() throws Exception {
        LOG.infoEntering("blockUnblockScore");
        KeyWallet caller = testWallets[1];
        TransactionResult result = helloWorld.invokeHello(caller);
        Utils.assertEquals(Constants.STATUS_SUCCESS, result.getStatus());

        //check blocked is 0x0 (false)
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(helloWorld.getAddress()))
                .build();
        RpcObject rpcObject = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getScoreStatus", params).asObject();
        boolean prevBlocked = rpcObject.getItem("blocked").asBoolean();
        assertFalse(prevBlocked);

        for (String method : new String[]{"blockScore", "unblockScore"}) {
            LOG.infoEntering("method[" + method + "]");
            sendGovCallTx(method, params);
            LOG.infoExiting();
            rpcObject = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS, "getScoreStatus", params).asObject();
            boolean blocked = rpcObject.getItem("blocked").asBoolean();
            assertTrue(toAddr.equals(Constants.GOV_ADDRESS) ? prevBlocked != blocked : prevBlocked == blocked);
            prevBlocked = blocked;

            LOG.infoEntering("method[hello], disabled[" + blocked + "]");
            try {
                result = helloWorld.invokeHello(caller);
                Utils.assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            }
            catch (ResultTimeoutException ex) {
                LOG.info("FAIL to get result by tx");
                if(toAddr.equals(Constants.GOV_ADDRESS)) {
                    Utils.assertEquals("blockScore", method);
                }
            }
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    public void setStepPrice() throws Exception {
        LOG.infoEntering("setStepPrice");
        BigInteger originPrice = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getStepPrice", null).asInteger();
        BigInteger newPrice = originPrice.add(BigInteger.valueOf(1));
        RpcObject params = new RpcObject.Builder()
                .put("price", new RpcValue(newPrice.toString()))
                .build();
        LOG.infoEntering("method[setStepPrice]");
        sendGovCallTx("setStepPrice", params);
        LOG.infoExiting();
        BigInteger resultPrice = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getStepPrice", null).asInteger();
        if(toAddr.equals(Constants.GOV_ADDRESS)) {
            Utils.assertEquals(newPrice, resultPrice);
            params = new RpcObject.Builder()
                    .put("price", new RpcValue(originPrice))
                    .build();
            sendGovCallTx("setStepPrice", params);
        }
        else {
            Utils.assertEquals(originPrice, resultPrice);
        }
        LOG.infoExiting();
    }

    @Test
    public void setStepCost() throws Exception{
        LOG.infoEntering("setStepCost");
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
            RpcObject params = new RpcObject.Builder()
                    .put("type", new RpcValue(type))
                    .put("cost", new RpcValue(newCost))
                    .build();
            LOG.infoEntering("method[setStepCost], type[" + type + "], cost[" + newCost + "]");
            txHashList[i] = Utils.sendTransactionWithCall(iconService, chain.networkId,
                    wallet, this.toAddr, "setStepCost", params, 0, false);
            LOG.infoExiting();
        }
        for(Bytes txHash : txHashList) {
            TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
            if(toAddr.equals(Constants.GOV_ADDRESS)) {
                Utils.assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            }
            else {
                Utils.assertEquals(Constants.STATUS_FAIL, result.getStatus());
            }
        }

        Map<String, BigInteger> cmpCosts;
        if(toAddr.equals(Constants.GOV_ADDRESS)) {
            cmpCosts = newStepCostsMap;
        }
        else {
            cmpCosts = originMap;
        }
        rpcObject = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getStepCosts", null).asObject();
        for (String type : GovScore.stepCostTypes) {
            Utils.assertEquals(cmpCosts.get(type), rpcObject.getItem(type).asInteger());
        }

        if(this.toAddr.equals(Constants.GOV_ADDRESS)) {
            // rollback
            txHashList = new Bytes[GovScore.stepCostTypes.length];
            for(int i = 0; i < GovScore.stepCostTypes.length; i++) {
                String type = GovScore.stepCostTypes[i];
                RpcObject params = new RpcObject.Builder()
                        .put("type", new RpcValue(type))
                        .put("cost", new RpcValue(originMap.get(type)))
                        .build();
                txHashList[i] = Utils.sendTransactionWithCall(iconService, chain.networkId,
                        wallet, this.toAddr, "setStepCost", params, 0, false);
            }

            for(Bytes txHash : txHashList) {
                TransactionResult result =
                        Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
                Utils.assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            }

            rpcObject = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS, "getStepCosts", null).asObject();
            for (String type : GovScore.stepCostTypes) {
                Utils.assertEquals(originMap.get(type), rpcObject.getItem(type).asInteger());
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void setMaxStepLimit() throws Exception {
        LOG.infoEntering("setMaxStepLimit");
        for(String type : new String[]{"invoke", "query"}) {
            RpcObject params = new RpcObject.Builder()
                    .put("contextType", new RpcValue(type))
                    .build();
            BigInteger originLimit = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS, "getMaxStepLimit", params).asInteger();

            BigInteger newLimit = originLimit.add(BigInteger.valueOf(1));
            params = new RpcObject.Builder()
                    .put("contextType", new RpcValue(type))
                    .put("limit", new RpcValue(newLimit))
                    .build();
            LOG.infoEntering("method[setMaxStepLimit], contextType[" + type + "]");
            sendGovCallTx("setMaxStepLimit", params);
            LOG.infoExiting();

            BigInteger resultLimit = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS,"getMaxStepLimit", params).asInteger();
            if (this.toAddr.equals(Constants.GOV_ADDRESS)) {
                Utils.assertEquals(newLimit, resultLimit);
                params = new RpcObject.Builder()
                        .put("contextType", new RpcValue(type))
                        .put("limit", new RpcValue(originLimit))
                        .build();
                sendGovCallTx("setMaxStepLimit", params);
            }
            else {
                Utils.assertEquals(originLimit, resultLimit);
            }
        }
        LOG.infoExiting();
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
            sendGovCallTx(method, builder.build());

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
        LOG.infoEntering("addRemoveMember");
        KeyWallet wallet = testWallets[0];
        RpcItem item = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS,"getMembers", null);

        RpcArray rpcArray = item.asArray();
        for(int i = 0; i < rpcArray.size(); i++) {
            if(rpcArray.get(i).asAddress().equals(wallet)) {
                throw new Exception();
            }
        }
        for (String method : new String[]{"addMember", "removeMember"}) {
            RpcObject params = new RpcObject.Builder()
                    .put("address", new RpcValue(wallet.getAddress().toString()))
                    .build();
            LOG.infoEntering("method[" + method + "]");
            sendGovCallTx(method, params);
            LOG.infoExiting();
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
            if(toAddr.equals(Constants.GOV_ADDRESS)) {
                if(bFound) {
                    Utils.assertEquals("addMember", method);
                }
                else {
                    Utils.assertEquals("removeMember", method);
                }
            } else {
                assertFalse(bFound);
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void addRemoveDeployer() throws Exception {
        LOG.infoEntering( "addRemoveDeployer");
        KeyWallet wallet = testWallets[0];
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(wallet.getAddress()))
                .build();
        boolean isDeployer = Utils.icxCall(iconService, Constants.CHAINSCORE_ADDRESS,"isDeployer", params).asBoolean();
        assertFalse(isDeployer);

        for (String method : new String[]{"addDeployer", "removeDeployer"}) {
            LOG.infoEntering("method[" + method + "]");
            sendGovCallTx(method, params);
            LOG.infoExiting();
            isDeployer = Utils.icxCall(iconService, Constants.CHAINSCORE_ADDRESS,"isDeployer", params).asBoolean();

            if(toAddr.equals(Constants.CHAINSCORE_ADDRESS)) {
                assertFalse(isDeployer);
            } else {
                if(method.equals("addDeployer")) {
                    assertTrue(isDeployer);
                } else {
                    assertFalse(isDeployer);
                }
            }
        }
        LOG.infoExiting();
    }

//    public void addLicense() {}
//
//    public void removeLicense( ) {}

}
