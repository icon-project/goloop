/*
 * Copyright 2019 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.ChainScore;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.score.HelloWorld;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.EnumSource;

import java.math.BigInteger;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static foundation.icon.test.cases.ScoreMethodTest.RpcAccessDenied;
import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.fail;
import static org.junit.jupiter.api.Assumptions.assumeTrue;

@Tag(Constants.TAG_PY_GOV)
@Tag(Constants.TAG_JAVA_GOV)
public class ChainScoreTest extends TestBase {
    private static final String SCORE_STATUS_PENDING = "pending";
    private static final String SCORE_STATUS_ACTIVE = "active";
    private static final String SCORE_STATUS_REJECTED = "rejected";

    private static TransactionHandler txHandler;
    private static ChainScore chainScore;
    private static GovScore govScore;
    private static GovScore.Fee fee;
    private static KeyWallet[] testWallets;
    private static final int testWalletNum = 3;
    private static KeyWallet governorWallet;

    enum TargetScore {
        TO_CHAINSCORE(Constants.CHAINSCORE_ADDRESS),
        TO_GOVSCORE(Constants.GOV_ADDRESS);

        Address addr;
        TargetScore(Address addr) {
            this.addr = addr;
        }
    }

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Chain chain = node.channels[0].chain;
        IconService iconService = new IconService(new HttpProvider(node.channels[0].getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        chainScore = new ChainScore(txHandler);
        govScore = new GovScore(txHandler);
        fee = govScore.getFee();
        governorWallet = chain.governorWallet;
        try {
            Bytes txHash = txHandler.transfer(chain.godWallet, governorWallet.getAddress(), ICX);
            assertSuccess(txHandler.getResult(txHash));

            testWallets = new KeyWallet[testWalletNum];
            for (int i = 0; i < testWalletNum; i++) {
                testWallets[i] = KeyWallet.create();
            }

            for (String type : new String[]{"invoke", "query"}) {
                assertSuccess(govScore.setMaxStepLimit(type, BigInteger.valueOf(10000000)));
            }
        } catch (Exception ex) {
            fail(ex.getMessage());
        }
    }

    @AfterAll
    public static void destroy() throws Exception {
        govScore.setFee(fee);
    }

    public void invokeAndCheckResult(Address target, String method, RpcObject params) throws Exception {
        if (target.equals(Constants.GOV_ADDRESS)) {
            assertSuccess(govScore.invokeAndWaitResult(governorWallet, method, params, null, Constants.DEFAULT_STEPS));
        } else {
            assertFailure(chainScore.invokeAndWaitResult(governorWallet, method, params, null, Constants.DEFAULT_STEPS));
        }
    }

    @Test
    public void disableEnableScore() throws Exception {
        LOG.infoEntering("disableEnableScore");
        // deploy new helloWorld score
        KeyWallet helloWorldOwner = KeyWallet.create();
        HelloWorld helloWorld = HelloWorld.install(txHandler, helloWorldOwner);
        KeyWallet caller = testWallets[0];
        KeyWallet[] fromWallets = { caller, helloWorldOwner};

        // check if invoking a method is successful first (i.e. enabled status)
        assertSuccess(helloWorld.invokeHello(caller));

        for (String method : new String[]{"disableScore", "enableScore"}) {
            for (KeyWallet from : fromWallets) {
                LOG.infoEntering("invoke", method + ", isOwner=" + (from == helloWorldOwner));
                RpcObject status = chainScore.getScoreStatus(helloWorld.getAddress());
                boolean prevDisabled = status.getItem("disabled").asBoolean();
                TransactionResult result;
                if (method.equals("disableScore")) {
                    result = chainScore.disableScore(from, helloWorld.getAddress());
                } else {
                    result = chainScore.enableScore(from, helloWorld.getAddress());
                }
                assertEquals(from == helloWorldOwner ? Constants.STATUS_SUCCESS : Constants.STATUS_FAILURE, result.getStatus());

                status = chainScore.getScoreStatus(helloWorld.getAddress());
                boolean disabled = status.getItem("disabled").asBoolean();
                assertEquals((from == helloWorldOwner), (prevDisabled != disabled));
                LOG.infoExiting();

                LOG.infoEntering("invokeHello", "disabled=" + disabled);
                if (disabled) {
                    try {
                        assertFailure(helloWorld.invokeHello(caller));
                    } catch (RpcError e) {
                        // expected failure
                    }
                } else {
                    assertSuccess(helloWorld.invokeHello(caller));
                }
                LOG.infoExiting();
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void setRevision() throws Exception {
        LOG.infoEntering("setRevision");
        int revision = chainScore.getRevision();
        for (int i = 1; i < 2; i++) {
            // It only allows to set a greater value than the current.
            // Setting the same revision would update the API info table.
            int wrongRevision = revision - i;
            LOG.infoEntering("setRevision to GOVERNANCE, OLD[" + revision + "], NEW[" + wrongRevision + "]");
            assertFailure(govScore.setRevision(wrongRevision));
            int newRevision = chainScore.getRevision();
            assertEquals(revision, newRevision);
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "acceptScore {0}")
    @EnumSource(TargetScore.class)
    public void acceptScore(TargetScore score) throws Exception {
        assumeTrue(chainScore.isAuditEnabled(), "audit is not enabled");
        LOG.infoEntering("acceptScore");
        // deploy new helloWorld score
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        Bytes txHash = txHandler.deployOnly(testWallets[0], HelloWorld.INSTALL_PATH, params);
        TransactionResult result = txHandler.getResult(txHash);
        assertSuccess(result);
        Address scoreAddr = new Address(result.getScoreAddress());
        HelloWorld helloWorld = new HelloWorld(txHandler, scoreAddr);
        KeyWallet caller = testWallets[1];
        String[] expectedStatus;
        if (score.addr == Constants.GOV_ADDRESS) {
            expectedStatus = new String[]{SCORE_STATUS_PENDING, SCORE_STATUS_ACTIVE};
        } else {
            expectedStatus = new String[]{SCORE_STATUS_PENDING, SCORE_STATUS_PENDING};
        }
        String expectedItem = "next";
        for (String expected : expectedStatus) {
            RpcObject status = chainScore.getScoreStatus(scoreAddr);
            RpcObject object = status.getItem(expectedItem).asObject();
            assertNotNull(object);
            assertEquals(expected, object.getItem("status").asString());
            LOG.infoEntering("invoke", "hello");
            try {
                result = helloWorld.invokeHello(caller);
                if (expected.equals(SCORE_STATUS_ACTIVE)) {
                    assertSuccess(result);
                } else {
                    assertFailure(result);
                }
            } catch (ResultTimeoutException ex) {
                assertEquals(SCORE_STATUS_PENDING, expected);
                LOG.info("Expected exception: " + ex.getMessage());
            }
            LOG.infoExiting();
            if (expected.equals(SCORE_STATUS_PENDING)) {
                LOG.infoEntering("invoke", "acceptScore");
                params = new RpcObject.Builder()
                        .put("txHash", new RpcValue(txHash))
                        .build();
                invokeAndCheckResult(score.addr, "acceptScore", params);
                if (score.addr == Constants.GOV_ADDRESS) {
                    expectedItem = "current";
                    // check the next item (it should be null)
                    status = chainScore.getScoreStatus(scoreAddr);
                    assertNull(status.getItem("next"));
                }
                LOG.infoExiting();
            }
        }
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "rejectScore {0}")
    @EnumSource(TargetScore.class)
    public void rejectScore(TargetScore score) throws Exception {
        assumeTrue(chainScore.isAuditEnabled(), "audit is not enabled");
        LOG.infoEntering("rejectScore");
        // deploy new helloWorld score
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        Bytes txHash = txHandler.deployOnly(testWallets[0], HelloWorld.INSTALL_PATH, params);
        TransactionResult result = txHandler.getResult(txHash);
        assertSuccess(result);
        Address scoreAddr = new Address(result.getScoreAddress());
        HelloWorld helloWorld = new HelloWorld(txHandler, scoreAddr);
        KeyWallet caller = testWallets[1];
        String[] expectedStatus;
        if (score.addr == Constants.GOV_ADDRESS) {
            expectedStatus = new String[]{SCORE_STATUS_PENDING, SCORE_STATUS_REJECTED};
        } else {
            expectedStatus = new String[]{SCORE_STATUS_PENDING, SCORE_STATUS_PENDING};
        }
        for (String expected : expectedStatus) {
            RpcObject status = chainScore.getScoreStatus(scoreAddr);
            assertNull(status.getItem("current"));
            RpcObject object = status.getItem("next").asObject();
            assertNotNull(object);
            assertEquals(expected, object.getItem("status").asString());
            LOG.infoEntering("invoke", "hello");
            try {
                assertFailure(helloWorld.invokeHello(caller));
            } catch (ResultTimeoutException ex) {
                LOG.info("Expected exception: " + ex.getMessage());
                //success
            }
            LOG.infoExiting();
            if (expected.equals(SCORE_STATUS_PENDING)) {
                LOG.infoEntering("invoke", "rejectScore");
                params = new RpcObject.Builder()
                        .put("txHash", new RpcValue(txHash))
                        .build();
                invokeAndCheckResult(score.addr, "rejectScore", params);
                LOG.infoExiting();
            }
        }
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "blockUnblockScore {0}")
    @EnumSource(TargetScore.class)
    public void blockUnblockScore(TargetScore score) throws Exception {
        LOG.infoEntering("blockUnblockScore");
        HelloWorld helloWorld = HelloWorld.install(txHandler, testWallets[0]);
        KeyWallet caller = testWallets[1];
        TransactionResult result = helloWorld.invokeHello(caller);
        assertSuccess(result);

        // check blocked is 0x0 (false)
        RpcObject status = chainScore.getScoreStatus(helloWorld.getAddress());
        boolean prevBlocked = status.getItem("blocked").asBoolean();
        assertFalse(prevBlocked);

        for (String method : new String[]{"blockScore", "unblockScore"}) {
            LOG.infoEntering("invoke", method);
            RpcObject params = new RpcObject.Builder()
                    .put("address", new RpcValue(helloWorld.getAddress()))
                    .build();
            invokeAndCheckResult(score.addr, method, params);
            LOG.infoExiting();
            status = chainScore.getScoreStatus(helloWorld.getAddress());
            boolean blocked = status.getItem("blocked").asBoolean();
            assertEquals(score.addr.equals(Constants.GOV_ADDRESS), (prevBlocked != blocked));
            prevBlocked = blocked;

            LOG.infoEntering("invokeHello", "blocked=" + blocked);
            try {
                result = helloWorld.invokeHello(caller);
                assertEquals(!blocked, Constants.STATUS_SUCCESS.equals(result.getStatus()));
            } catch (ResultTimeoutException ex) {
                assertTrue(blocked);
            } catch (RpcError err) {
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
        BigInteger originPrice = chainScore.getStepPrice();
        BigInteger newPrice = originPrice.add(BigInteger.valueOf(1));
        LOG.infoEntering("invoke", "setStepPrice, " + originPrice + " -> " + newPrice);
        RpcObject params = new RpcObject.Builder()
                .put("price", new RpcValue(newPrice))
                .build();
        invokeAndCheckResult(score.addr, "setStepPrice", params);
        LOG.infoExiting();

        BigInteger resultPrice = chainScore.getStepPrice();
        if (score.addr.equals(Constants.GOV_ADDRESS)) {
            assertEquals(newPrice, resultPrice);
            LOG.info("invoke setStepPrice again for revert");
            params = new RpcObject.Builder()
                    .put("price", new RpcValue(originPrice))
                    .build();
            invokeAndCheckResult(score.addr, "setStepPrice", params);
        } else {
            assertEquals(originPrice, resultPrice);
            LOG.info("no change");
        }
        BigInteger curPrice = chainScore.getStepPrice();
        LOG.info("revertedPrice = " + curPrice);
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "setStepCost {0}")
    @EnumSource(TargetScore.class)
    public void setStepCost(TargetScore score) throws Exception {
        LOG.infoEntering("setStepCost");
        KeyWallet wallet = testWallets[0];
        Score target = new Score(txHandler, score.addr);
        Bytes[] txHashList = new Bytes[GovScore.stepCostTypes.length];
        RpcObject rpcObject = chainScore.call("getStepCosts", null).asObject();
        Map<String, BigInteger> originMap = new HashMap<>();
        Map<String, BigInteger> newMap = new HashMap<>();
        long cnt = 1;
        for (int i = 0; i < GovScore.stepCostTypes.length; i++) {
            String type = GovScore.stepCostTypes[i];
            BigInteger oldCost = rpcObject.getItem(type).asInteger();
            originMap.put(type, oldCost);
            BigInteger newCost = oldCost.add(BigInteger.valueOf(cnt));
            newMap.put(type, newCost);
            cnt += 1;
            RpcObject params = new RpcObject.Builder()
                    .put("type", new RpcValue(type))
                    .put("cost", new RpcValue(newCost))
                    .build();
            LOG.info("invoke setStepCost: type=" + type + ", cost=" + newCost);
            txHashList[i] = target.invoke(wallet, "setStepCost", params);
        }
        for (Bytes txHash : txHashList) {
            if (score.addr.equals(Constants.GOV_ADDRESS)) {
                assertSuccess(target.getResult(txHash));
            } else {
                assertFailure(target.getResult(txHash));
            }
        }

        LOG.infoEntering("check", "stepCosts");
        Map<String, BigInteger> cmpCosts;
        if (score.addr.equals(Constants.GOV_ADDRESS)) {
            cmpCosts = newMap;
        } else {
            cmpCosts = originMap;
        }
        rpcObject = chainScore.call("getStepCosts", null).asObject();
        for (String type : GovScore.stepCostTypes) {
            assertEquals(cmpCosts.get(type), rpcObject.getItem(type).asInteger());
        }
        LOG.infoExiting();

        if (score.addr.equals(Constants.GOV_ADDRESS)) {
            LOG.infoEntering("rollback", "stepCosts");
            txHashList = new Bytes[GovScore.stepCostTypes.length];
            for (int i = 0; i < GovScore.stepCostTypes.length; i++) {
                String type = GovScore.stepCostTypes[i];
                RpcObject params = new RpcObject.Builder()
                        .put("type", new RpcValue(type))
                        .put("cost", new RpcValue(originMap.get(type)))
                        .build();
                txHashList[i] = target.invoke(wallet, "setStepCost", params);
            }
            for (Bytes txHash : txHashList) {
                assertSuccess(target.getResult(txHash));
            }
            rpcObject = chainScore.call("getStepCosts", null).asObject();
            for (String type : GovScore.stepCostTypes) {
                assertEquals(originMap.get(type), rpcObject.getItem(type).asInteger());
            }
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "setMaxStepLimit {0}")
    @EnumSource(TargetScore.class)
    public void setMaxStepLimit(TargetScore score) throws Exception {
        LOG.infoEntering("setMaxStepLimit");
        for (String type : new String[]{"invoke", "query"}) {
            RpcObject qParams = new RpcObject.Builder()
                    .put("contextType", new RpcValue(type))
                    .build();
            BigInteger originLimit = chainScore.call("getMaxStepLimit", qParams).asInteger();

            LOG.infoEntering("invoke", "setMaxStepLimit, contextType=" + type);
            BigInteger newLimit = originLimit.add(BigInteger.valueOf(1));
            RpcObject params = new RpcObject.Builder()
                    .put("contextType", new RpcValue(type))
                    .put("limit", new RpcValue(newLimit))
                    .build();
            invokeAndCheckResult(score.addr, "setMaxStepLimit", params);
            LOG.infoExiting();

            BigInteger resultLimit = chainScore.call("getMaxStepLimit", qParams).asInteger();
            if (score.addr.equals(Constants.GOV_ADDRESS)) {
                assertEquals(newLimit, resultLimit);
                LOG.infoEntering("rollback", "maxStepLimit");
                params = new RpcObject.Builder()
                        .put("contextType", new RpcValue(type))
                        .put("limit", new RpcValue(originLimit))
                        .build();
                invokeAndCheckResult(score.addr, "setMaxStepLimit", params);
                LOG.infoExiting();
            } else {
                assertEquals(originLimit, resultLimit);
            }
        }
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "grantRevokeValidator {0}")
    @EnumSource(TargetScore.class)
    public void grantRevokeValidator(TargetScore score) throws Exception {
        assumeTrue(score.addr.equals(Constants.CHAINSCORE_ADDRESS) || Env.nodes.length >= 3);
        LOG.infoEntering("grantRevokeValidator");
        KeyWallet wallet = testWallets[0];
        RpcItem item = chainScore.call("getValidators", null);
        RpcArray rpcArray = item.asArray();
        for (int i = 0; i < rpcArray.size(); i++) {
            if (rpcArray.get(i).asAddress().equals(wallet.getAddress())) {
                throw new Exception();
            }
        }
        for (String method : new String[]{"grantValidator", "revokeValidator"}) {
            LOG.infoEntering("invoke", method);
            RpcObject rpcObject = new RpcObject.Builder()
                    .put("address", new RpcValue(wallet.getAddress()))
                    .build();
            invokeAndCheckResult(score.addr, method, rpcObject);
            LOG.infoExiting();

            item = chainScore.call("getValidators", null);
            boolean bFound = false;
            rpcArray = item.asArray();
            for (int i = 0; i < rpcArray.size(); i++) {
                if (rpcArray.get(i).asAddress().equals(wallet.getAddress())) {
                    bFound = true;
                    break;
                }
            }
            if (score.addr.equals(Constants.GOV_ADDRESS)) {
                if (bFound) {
                    assertEquals("grantValidator", method);
                } else {
                    assertEquals("revokeValidator", method);
                }
            } else {
                assertFalse(bFound);
            }
        }
        LOG.infoExiting();
    }

    @ParameterizedTest(name = "addRemoveMember {0}")
    @EnumSource(TargetScore.class)
    public void addRemoveMember(TargetScore score) throws Exception {
        LOG.infoEntering("addRemoveMember");
        KeyWallet wallet = testWallets[0];
        RpcItem item = chainScore.call("getMembers", null);
        RpcArray rpcArray = item.asArray();
        for (int i = 0; i < rpcArray.size(); i++) {
            if (rpcArray.get(i).asAddress().equals(wallet.getAddress())) {
                throw new Exception();
            }
        }
        for (String method : new String[]{"addMember", "removeMember"}) {
            LOG.infoEntering("invoke", method);
            RpcObject params = new RpcObject.Builder()
                    .put("address", new RpcValue(wallet.getAddress()))
                    .build();
            invokeAndCheckResult(score.addr, method, params);
            LOG.infoExiting();
            item = chainScore.call("getMembers", null);
            boolean bFound = false;
            rpcArray = item.asArray();
            for (int i = 0; i < rpcArray.size(); i++) {
                if (rpcArray.get(i).asAddress().equals(wallet.getAddress())) {
                    bFound = true;
                    break;
                }
            }
            if (score.addr.equals(Constants.GOV_ADDRESS)) {
                if (bFound) {
                    assertEquals("addMember", method);
                } else {
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
        LOG.infoEntering("addRemoveDeployer");
        KeyWallet wallet = testWallets[0];
        boolean isDeployer = chainScore.isDeployer(wallet.getAddress());
        assertFalse(isDeployer);

        for (String method : new String[]{"addDeployer", "removeDeployer"}) {
            LOG.infoEntering("invoke", method);
            RpcObject params = new RpcObject.Builder()
                    .put("address", new RpcValue(wallet.getAddress()))
                    .build();
            invokeAndCheckResult(score.addr, method, params);
            LOG.infoExiting();

            isDeployer = chainScore.isDeployer(wallet.getAddress());
            if (score.addr.equals(Constants.CHAINSCORE_ADDRESS)) {
                assertFalse(isDeployer);
            } else {
                if (method.equals("addDeployer")) {
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
        int old = chainScore.call("getRoundLimitFactor", null).asInteger().intValue();
        final int target = 3;
        LOG.infoEntering("invoke", "setRoundLimitFactor -> " + target);
        RpcObject params = new RpcObject.Builder()
                .put("factor", new RpcValue(BigInteger.valueOf(target)))
                .build();
        invokeAndCheckResult(score.addr, "setRoundLimitFactor", params);
        LOG.infoExiting();

        if (score == TargetScore.TO_GOVSCORE) {
            int factor = chainScore.call("getRoundLimitFactor", null).asInteger().intValue();
            assertEquals(target, factor);

            LOG.infoEntering("rollback", "setRoundLimitFactor -> " + old);
            params = new RpcObject.Builder()
                    .put("factor", new RpcValue(BigInteger.valueOf(old)))
                    .build();
            invokeAndCheckResult(score.addr, "setRoundLimitFactor", params);
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    public void getDeployers() throws Exception {
        LOG.infoEntering("check revision");
        final int requiredRevision = 7;
        int revision = chainScore.getRevision();
        if (revision < requiredRevision) {
            LOG.infoEntering("assert", "MethodNotFound");
            try {
                chainScore.getDeployers();
                fail();
            } catch (RpcError e) {
                LOG.info("Expected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
            }
            LOG.infoExiting();
        }
        LOG.infoExiting();

        // NOTE: Uncomment the block below if you want to test when the revision < requiredRevision
        //revision = setRevisionIfRequired(revision, requiredRevision);

        assumeTrue(revision >= requiredRevision);
        LOG.infoEntering("invoke", "getDeployers");
        List<Address> deployers = chainScore.getDeployers();
        int prevSize = deployers.size();
        LOG.info(">>> prevSize=" + prevSize);
        LOG.infoExiting();

        LOG.infoEntering("invoke", "addDeployer");
        for (KeyWallet wallet : testWallets) {
            LOG.info("  - wallet=" + wallet.getAddress());
            assertFalse(chainScore.isDeployer(wallet.getAddress()));
            assertSuccess(govScore.addDeployer(wallet.getAddress()));
        }
        LOG.infoExiting();

        LOG.infoEntering("invoke", "getDeployers [2]");
        deployers = chainScore.getDeployers();
        LOG.info(">>> size=" + deployers.size());
        assertEquals(prevSize + testWallets.length, deployers.size());
        for (Address deployer : deployers) {
            LOG.info("  - deployer=" + deployer);
        }
        LOG.infoExiting();

        LOG.infoEntering("invoke", "removeDeployer");
        for (KeyWallet wallet : testWallets) {
            assertSuccess(govScore.removeDeployer(wallet.getAddress()));
        }
        LOG.infoExiting();
    }

    @Test
    public void setDeployerWhiteListEnabled() throws Exception {
        LOG.infoEntering("check revision");
        final int requiredRevision = 7;
        int revision = chainScore.getRevision();
        if (revision < requiredRevision) {
            LOG.infoEntering("assert", "MethodNotFound");
            assertFailure(govScore.setDeployerWhiteListEnabled(true));
            LOG.infoExiting();
        }
        LOG.infoExiting();

        // NOTE: Uncomment the block below if you want to test when the revision < requiredRevision
        //revision = setRevisionIfRequired(revision, requiredRevision);

        assumeTrue(revision >= requiredRevision);
        int serviceConfig = chainScore.getServiceConfig();
        boolean enabled = ChainScore.isDeployerWhiteListEnabled(serviceConfig);
        boolean expected = !enabled;

        LOG.infoEntering("invoke and check", enabled + " --> " + expected);
        assertSuccess(govScore.setDeployerWhiteListEnabled(expected));
        assertEquals(expected, chainScore.isDeployerWhiteListEnabled());
        LOG.infoExiting();

        if (expected) {
            LOG.info("TODO: invoke deploy test with the deployerWhiteList");
        }

        LOG.infoEntering("revert config", String.valueOf(enabled));
        assertSuccess(govScore.setDeployerWhiteListEnabled(enabled));
        assertEquals(enabled, chainScore.isDeployerWhiteListEnabled());
        LOG.infoExiting();
    }

    private int setRevisionIfRequired(int current, int required) throws Exception {
        if (current < required) {
            LOG.infoEntering("invoke", "setRevision " + current + " -> " + required);
            govScore.setRevision(required);
            current = chainScore.getRevision();
            assertEquals(required, current);
            LOG.infoExiting();
        }
        return current;
    }
}
