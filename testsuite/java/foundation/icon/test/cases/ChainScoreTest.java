package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.score.HelloWorld;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.EnumSource;

import java.math.BigInteger;
import java.util.HashMap;
import java.util.Map;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.*;
import static org.junit.jupiter.api.Assumptions.assumeTrue;

/*
test methods
    disableEnableScore
    setRevision
    acceptScore
    rejectScore
    blockUnblockScore
    setStepPrice
    setStepCost
    setMaxStepLimit
    addRemoveMember
    addRemoveDeployer
 */
@Tag(Constants.TAG_PY_GOV)
public class ChainScoreTest{
    private static Env.Chain chain;
    private static IconService iconService;
    private static KeyWallet helloWorldOwner;
    private static KeyWallet[]testWallets;
    private static final int testWalletNum = 3;
    private static HelloWorld helloWorld;

    enum TargetScore {
        TO_CHAINSCORE(Constants.CHAINSCORE_ADDRESS),
        TO_GOVSCORE(Constants.GOV_ADDRESS);

        Address addr;
        TargetScore(Address addr) {
            this.addr = addr;
        }
    }

    @BeforeAll
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
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        }
    }

    @AfterAll
    public static void destroy() throws Exception {
        String []cTypes = {"invoke", "query"};
        BigInteger stepPrice = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getStepPrice", null).asInteger();
        LOG.info("stepPrice = " + stepPrice);
        for(String cType : cTypes) {
            RpcObject.Builder builder = new RpcObject.Builder();
            builder.put("contextType", new RpcValue(cType));
            builder.put("limit", new RpcValue("0"));
            TransactionResult result = Utils.sendTransactionWithCall(iconService, chain.networkId,
                    KeyWallet.create(), Constants.GOV_ADDRESS, "setMaxStepLimit", builder.build(), 0);
            if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
                LOG.info("result = " + result);
                throw new Exception();
            }
        }
    }

    public TransactionResult sendGovCallTx(Address toAddr, String method, RpcObject params) throws Exception {
        TransactionResult result;
        result = Utils.sendTransactionWithCall(iconService, chain.networkId,
                chain.governorWallet, toAddr, method, params, 0);
        if(toAddr.equals(Constants.GOV_ADDRESS)) {
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        }
        else {
            assertEquals(Constants.STATUS_FAIL, result.getStatus());
        }
        return result;
    }

    @Test
    public void disableEnableScore() throws Exception{
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
                assertEquals(from == helloWorldOwner ? Constants.STATUS_SUCCESS : Constants.STATUS_FAIL, result.getStatus());

                boolean disabled = Utils.icxCall(iconService,
                        Constants.CHAINSCORE_ADDRESS, "getScoreStatus", params).asObject().getItem("disabled").asBoolean();
                assertTrue(from == helloWorldOwner ? prevDisabled != disabled : prevDisabled == disabled);

                try {
                    LOG.infoEntering("method[hello], disabled[" + disabled + "]");
                    result = helloWorld.invokeHello(caller);
                    assertEquals(!disabled, Constants.STATUS_SUCCESS.equals(result.getStatus()));
                }
                catch (ResultTimeoutException ex) {
                    LOG.info("FAIL to get result by tx");
                    assertTrue(disabled);
                }
                LOG.infoExiting();
            }
        }
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "setRevision {0}")
    @EnumSource(TargetScore.class)
    public void setRevision(TargetScore score) throws Exception{
        LOG.infoEntering("setRevision");
        BigInteger oldRevision = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getRevision", null).asInteger();
        // TODO add test with greater revision
//        BigInteger newRevision = oldRevision.add(BigInteger.valueOf(1));
//        RpcObject params = new RpcObject.Builder()
//                .put("code", new RpcValue(newRevision))
//                .build();
//        LOG.infoEntering("method[setRevision] OLD[" + oldRevision + "], NEW[" + newRevision + "]");
//        sendGovCallTx(score.addr,  "setRevision", params);
//        LOG.infoExiting();
        BigInteger newRevision;
        RpcObject params;

        BigInteger revision = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getRevision", null).asInteger();
        if(score.addr.equals(Constants.GOV_ADDRESS)) {
//            assertEquals(newRevision, revision);
            for (int i = 1; i < 2; i++) {
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
                assertEquals(Constants.STATUS_FAIL, result.getStatus());
                newRevision = Utils.icxCall(iconService,
                        Constants.CHAINSCORE_ADDRESS, "getRevision", null).asInteger();
                assertEquals(revision, newRevision);
            }
        }
        else {
            assertEquals(oldRevision, revision);
        }
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "acceptScore {0}")
    @EnumSource(TargetScore.class)
    public void acceptScore(TargetScore score) throws Exception {
        if (!Utils.isAuditEnabled(iconService)) {
            return;
        }
        LOG.infoEntering("acceptScore");
        KeyWallet owner = KeyWallet.create();
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        Bytes txHash = Utils.deployScore(iconService, chain.networkId, owner, Constants.CHAINSCORE_ADDRESS,
                Constants.SCORE_HELLOWORLD_PATH, params);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, 5000);
        assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);

        Address scoreAddr = new Address(result.getScoreAddress());
        KeyWallet caller = KeyWallet.create();
        String []expectedStatus;
        if(score.addr == Constants.GOV_ADDRESS) {
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
            assertEquals(expect, object.getItem("status").asString());

            try {
                TransactionResult tr = Utils.sendTransactionWithCall(iconService, chain.networkId,
                        caller, scoreAddr, "hello", null, 0);
                assertEquals(expect.equals(Constants.SCORE_STATUS_ACTIVE), Constants.STATUS_SUCCESS.equals(tr.getStatus()));
            }
            catch(ResultTimeoutException ex) {
                LOG.info("FAIL to get result by tx");
                assertEquals(Constants.SCORE_STATUS_PENDING, expect);
            }
            if(expect == Constants.SCORE_STATUS_PENDING) {
                params = new RpcObject.Builder()
                        .put("txHash", new RpcValue(txHash))
                        .build();
                LOG.infoEntering( "accept score");
                TransactionResult acceptResult =
                        Utils.sendTransactionWithCall(iconService, chain.networkId,
                                chain.governorWallet, score.addr, "acceptScore", params, 0);
                if(score.addr == Constants.GOV_ADDRESS) {
                    assertEquals(Constants.STATUS_SUCCESS, acceptResult.getStatus());
                    expectedItem = "current";
                }
                else {
                    assertEquals(Constants.STATUS_FAIL, acceptResult.getStatus());
                }
                LOG.infoExiting();
            }
        }
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "rejectScore {0}")
    @EnumSource(TargetScore.class)
    public void rejectScore(TargetScore score) throws Exception {
        if (!Utils.isAuditEnabled(iconService)) {
            return;
        }
        LOG.infoEntering("rejectScore");
        KeyWallet owner = KeyWallet.create();
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        Bytes txHash = Utils.deployScore(iconService, chain.networkId, owner,
                Constants.CHAINSCORE_ADDRESS, Constants.SCORE_HELLOWORLD_PATH, params);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, 5000);
        assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);

        Address scoreAddr = new Address(result.getScoreAddress());
        KeyWallet caller = KeyWallet.create();
        String []expectedStatus;
        if(score.addr == Constants.GOV_ADDRESS) {
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
                TransactionResult tr = Utils.sendTransactionWithCall(iconService, chain.networkId, caller, scoreAddr, "hello", null, 0);
                assertEquals(Constants.STATUS_FAIL,tr.getStatus());
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
                                chain.governorWallet, score.addr, "rejectScore", params, 0);
                if(score.addr == Constants.GOV_ADDRESS) {
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

    @ParameterizedTest(name = "blockUnblockScore {0}")
    @EnumSource(TargetScore.class)
    public void blockUnblockScore(TargetScore score) throws Exception {
        LOG.infoEntering("blockUnblockScore");
        KeyWallet caller = testWallets[1];
        TransactionResult result = helloWorld.invokeHello(caller);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());

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
            sendGovCallTx(score.addr, method, params);
            LOG.infoExiting();
            rpcObject = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS, "getScoreStatus", params).asObject();
            boolean blocked = rpcObject.getItem("blocked").asBoolean();
            assertTrue(score.addr.equals(Constants.GOV_ADDRESS) ? prevBlocked != blocked : prevBlocked == blocked);
            prevBlocked = blocked;

            LOG.infoEntering("method[hello], disabled[" + blocked + "]");
            try {
                result = helloWorld.invokeHello(caller);
                assertEquals(!blocked, Constants.STATUS_SUCCESS.equals(result.getStatus()));
            }
            catch (ResultTimeoutException ex) {
                LOG.info("FAIL to get result by tx");
                assertTrue(blocked);
            }
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "setStepPrice {0}")
    @EnumSource(TargetScore.class)
    public void setStepPrice(TargetScore score) throws Exception {
        LOG.infoEntering("setStepPrice");
        BigInteger originPrice = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getStepPrice", null).asInteger();
        LOG.info("originPrice = " + originPrice);
        BigInteger newPrice = originPrice.add(BigInteger.valueOf(1));
        RpcObject params = new RpcObject.Builder()
                .put("price", new RpcValue(newPrice.toString()))
                .build();
        LOG.infoEntering("method[setStepPrice]");
        sendGovCallTx(score.addr, "setStepPrice", params);
        LOG.infoExiting();
        BigInteger resultPrice = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getStepPrice", null).asInteger();
        if(score.addr.equals(Constants.GOV_ADDRESS)) {
            assertEquals(newPrice, resultPrice);
            params = new RpcObject.Builder()
                    .put("price", new RpcValue(originPrice))
                    .build();
            sendGovCallTx(score.addr, "setStepPrice", params);
        }
        else {
            assertEquals(originPrice, resultPrice);
        }
        BigInteger curPrice = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getStepPrice", null).asInteger();
        LOG.info("revertedPrice = " + curPrice);
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "setStepCost {0}")
    @EnumSource(TargetScore.class)
    public void setStepCost(TargetScore score) throws Exception{
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
                    wallet, score.addr, "setStepCost", params, 0, false);
            LOG.infoExiting();
        }
        for(Bytes txHash : txHashList) {
            TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
            if(score.addr.equals(Constants.GOV_ADDRESS)) {
                assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            }
            else {
                assertEquals(Constants.STATUS_FAIL, result.getStatus());
            }
        }

        Map<String, BigInteger> cmpCosts;
        if(score.addr.equals(Constants.GOV_ADDRESS)) {
            cmpCosts = newStepCostsMap;
        }
        else {
            cmpCosts = originMap;
        }
        rpcObject = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getStepCosts", null).asObject();
        for (String type : GovScore.stepCostTypes) {
            assertEquals(cmpCosts.get(type), rpcObject.getItem(type).asInteger());
        }

        if(score.addr.equals(Constants.GOV_ADDRESS)) {
            // rollback
            txHashList = new Bytes[GovScore.stepCostTypes.length];
            for(int i = 0; i < GovScore.stepCostTypes.length; i++) {
                String type = GovScore.stepCostTypes[i];
                RpcObject params = new RpcObject.Builder()
                        .put("type", new RpcValue(type))
                        .put("cost", new RpcValue(originMap.get(type)))
                        .build();
                txHashList[i] = Utils.sendTransactionWithCall(iconService, chain.networkId,
                        wallet, score.addr, "setStepCost", params, 0, false);
            }

            for(Bytes txHash : txHashList) {
                TransactionResult result =
                        Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
                assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            }

            rpcObject = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS, "getStepCosts", null).asObject();
            for (String type : GovScore.stepCostTypes) {
                assertEquals(originMap.get(type), rpcObject.getItem(type).asInteger());
            }
        }
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "setMaxStepLimit {0}")
    @EnumSource(TargetScore.class)
    public void setMaxStepLimit(TargetScore score) throws Exception {
        LOG.infoEntering("setMaxStepLimit");
        for(String type : new String[]{"invoke", "query"}) {
            RpcObject qParams = new RpcObject.Builder()
                    .put("contextType", new RpcValue(type))
                    .build();
            BigInteger originLimit = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS, "getMaxStepLimit", qParams).asInteger();

            BigInteger newLimit = originLimit.add(BigInteger.valueOf(1));
            RpcObject params = new RpcObject.Builder()
                    .put("contextType", new RpcValue(type))
                    .put("limit", new RpcValue(newLimit))
                    .build();
            LOG.infoEntering("method[setMaxStepLimit], contextType[" + type + "]");
            sendGovCallTx(score.addr, "setMaxStepLimit", params);
            LOG.infoExiting();

            BigInteger resultLimit = Utils.icxCall(iconService,
                    Constants.CHAINSCORE_ADDRESS,"getMaxStepLimit", qParams).asInteger();
            if (score.addr.equals(Constants.GOV_ADDRESS)) {
                assertEquals(newLimit, resultLimit);
                params = new RpcObject.Builder()
                        .put("contextType", new RpcValue(type))
                        .put("limit", new RpcValue(originLimit))
                        .build();
                sendGovCallTx(score.addr, "setMaxStepLimit", params);
            }
            else {
                assertEquals(originLimit, resultLimit);
            }
        }
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "grantRevokeValidator {0}")
    @EnumSource(TargetScore.class)
    public void grantRevokeValidator(TargetScore score) throws Exception {
        LOG.infoEntering("grantRevokeValidator");
        assumeTrue(score.addr.equals(Constants.CHAINSCORE_ADDRESS) || Env.nodes.length >= 3);
        KeyWallet wallet = testWallets[0];
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(wallet.getAddress()))
                .build();
        RpcItem item = Utils.icxCall(iconService,
                Constants.CHAINSCORE_ADDRESS, "getValidators", null);
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
            LOG.infoEntering("method[" + method + "]");
            sendGovCallTx(score.addr, method, builder.build());
            LOG.infoExiting();

            item = Utils.icxCall(iconService, Constants.CHAINSCORE_ADDRESS,
                    "getValidators", null);
            boolean bFound = false;
            rpcArray = item.asArray();
            for(int i = 0; i < rpcArray.size(); i++) {
                if(rpcArray.get(i).asAddress().equals(wallet.getAddress())) {
                    bFound = true;
                    break;
                }
            }

            if(score.addr.equals(Constants.CHAINSCORE_ADDRESS)) {
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
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "addRemoveMember {0}")
    @EnumSource(TargetScore.class)
    public void addRemoveMember(TargetScore score) throws Exception{
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
            sendGovCallTx(score.addr, method, params);
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
            if(score.addr.equals(Constants.GOV_ADDRESS)) {
                if(bFound) {
                    assertEquals("addMember", method);
                }
                else {
                    assertEquals("removeMember", method);
                }
            } else {
                assertFalse(bFound);
            }
        }
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "addRemoveDeployer {0}")
    @EnumSource(TargetScore.class)
    public void addRemoveDeployer(TargetScore score) throws Exception {
        LOG.infoEntering( "addRemoveDeployer");
        KeyWallet wallet = testWallets[0];
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(wallet.getAddress()))
                .build();
        boolean isDeployer = Utils.icxCall(iconService, Constants.CHAINSCORE_ADDRESS,"isDeployer", params).asBoolean();
        assertFalse(isDeployer);

        for (String method : new String[]{"addDeployer", "removeDeployer"}) {
            LOG.infoEntering("method[" + method + "]");
            sendGovCallTx(score.addr, method, params);
            LOG.infoExiting();
            isDeployer = Utils.icxCall(iconService, Constants.CHAINSCORE_ADDRESS,"isDeployer", params).asBoolean();

            if(score.addr.equals(Constants.CHAINSCORE_ADDRESS)) {
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

    @ParameterizedTest(name = "setRoundLimitFactor {0}")
    @EnumSource(TargetScore.class)
    public void setRoundLimitFactor(TargetScore score) throws Exception {
        LOG.infoEntering("setRoundLimitFactor");

        int old = Utils.icxCall(iconService, Constants.CHAINSCORE_ADDRESS,
                "getRoundLimitFactor", null)
                .asInteger().intValue();

        int target = 3;
        RpcObject params = new RpcObject.Builder()
                .put("factor", new RpcValue(BigInteger.valueOf(target)))
                .build();

        sendGovCallTx(score.addr, "setRoundLimitFactor", params);

        if (score == TargetScore.TO_GOVSCORE) {
            int factor = Utils.icxCall(iconService, Constants.CHAINSCORE_ADDRESS,
                    "getRoundLimitFactor", null)
                    .asInteger().intValue();

            assertEquals(target, factor);

            params = new RpcObject.Builder()
                    .put("factor", new RpcValue(BigInteger.valueOf(old)))
                    .build();
            sendGovCallTx(score.addr, "setRoundLimitFactor", params);
        }

        LOG.infoExiting();
    }
}
