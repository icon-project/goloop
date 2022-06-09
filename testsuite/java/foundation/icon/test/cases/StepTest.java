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
import foundation.icon.icx.SignedTransaction;
import foundation.icon.icx.Transaction;
import foundation.icon.icx.TransactionBuilder;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.common.ZipFile;
import foundation.icon.test.score.ChainScore;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.score.HelloWorld;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.math.BigInteger;
import java.nio.charset.StandardCharsets;
import java.util.Map;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertDoesNotThrow;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

@Tag(Constants.TAG_PY_GOV)
public class StepTest extends TestBase {
    private static TransactionHandler txHandler;
    private static ChainScore chainScore;
    private static GovScore govScore;
    private static GovScore.Fee fee;
    private static KeyWallet[] testWallets;

    private static final BigInteger STEP_PRICE = BigInteger.valueOf(10_000_000_000L);
    private static final int TYPE_INT = 0;
    private static final int TYPE_STR = 1;
    private static final int TYPE_BYTES = 2;
    private static final int TYPE_ADDR = 3;

    @BeforeAll
    static void setup() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        chainScore = new ChainScore(txHandler);
        govScore = new GovScore(txHandler);
        fee = govScore.getFee();

        testWallets = new KeyWallet[5];
        Address[] addresses = new Address[testWallets.length + 1];
        for (int i = 0; i < testWallets.length; i++) {
            KeyWallet wallet = KeyWallet.create();
            testWallets[i] = wallet;
            addresses[i] = wallet.getAddress();
        }
        addresses[testWallets.length] = chain.governorWallet.getAddress();
        transferAndCheckResult(txHandler, addresses, ICX.multiply(BigInteger.valueOf(30)));

        LOG.infoEntering("initSteps");
        initStepCosts(govScore);
        LOG.infoExiting();
    }

    static void initStepCosts(GovScore govScore) throws Exception {
        govScore.setMaxStepLimit("invoke", BigInteger.valueOf(2_500_000_000L));
        govScore.setMaxStepLimit("query", BigInteger.valueOf(50_000_000L));
        govScore.setStepPrice(STEP_PRICE);
        govScore.setStepCosts(StepType.getStepCosts());
    }

    @AfterAll
    static void destroy() throws Exception {
        govScore.setFee(fee);
    }

    private enum StepType {
        DEFAULT("default", 100000),
        CONTRACT_CALL("contractCall", 25000),
        CONTRACT_CREATE("contractCreate", 1000000000),
        CONTRACT_UPDATE("contractUpdate", 1600000000),
        CONTRACT_SET("contractSet", 30000),
        GET("get", 0),
        SET("set", 320),
        REPLACE("replace", 80),
        DELETE("delete", -240),
        INPUT("input", 200),
        EVENTLOG("eventLog", 100),
        APICALL("apiCall", 10000);

        private final String name;
        private final BigInteger steps;

        StepType(String name, long steps) {
            this.name = name;
            this.steps = BigInteger.valueOf(steps);
        }
        public String getName() { return name; }
        public BigInteger getSteps() { return steps; }

        public static Map<String, BigInteger> getStepCosts() {
            return Map.ofEntries(
                    Map.entry(DEFAULT.getName(), DEFAULT.getSteps()),
                    Map.entry(CONTRACT_CALL.getName(), CONTRACT_CALL.getSteps()),
                    Map.entry(CONTRACT_CREATE.getName(), CONTRACT_CREATE.getSteps()),
                    Map.entry(CONTRACT_UPDATE.getName(), CONTRACT_UPDATE.getSteps()),
                    Map.entry(CONTRACT_SET.getName(), CONTRACT_SET.getSteps()),
                    Map.entry(GET.getName(), GET.getSteps()),
                    Map.entry(SET.getName(), SET.getSteps()),
                    Map.entry(REPLACE.getName(), REPLACE.getSteps()),
                    Map.entry(DELETE.getName(), DELETE.getSteps()),
                    Map.entry(INPUT.getName(), INPUT.getSteps()),
                    Map.entry(EVENTLOG.getName(), EVENTLOG.getSteps()),
                    Map.entry(APICALL.getName(), APICALL.getSteps())
            );
        }
    }

    private static class StepTransaction {
        private BigInteger expectedStep;
        private BigInteger usedFee;
        private BigInteger treasuryFee;
        private Address scoreAddr;

        BigInteger expectedStep() {
            return expectedStep;
        }

        BigInteger expectedFee() {
            return expectedStep.multiply(STEP_PRICE);
        }

        BigInteger getUsedFee() {
            return usedFee;
        }

        BigInteger getTreasuryFee() {
            return treasuryFee;
        }

        public Address getScoreAddress() {
            return scoreAddr;
        }

        void addOperation(StepType stepType, int type, String val) {
            long valSize;
            switch (type) {
                case TYPE_INT:
                    var v = new BigInteger(val.substring(2), 16);
                    valSize = (v.bitLength()+1) / 8 + 1;
                    break;
                case TYPE_BYTES:
                    if (!val.startsWith("0x")) {
                        throw new IllegalArgumentException();
                    }
                    valSize = (val.length() - 2) / 2;
                    break;
                case TYPE_ADDR:
                    valSize = 21; // address should be 21 bytes
                    break;
                case TYPE_STR:
                default:
                    valSize = val.length();
                    break;
            }
            LOG.info("addOperation val : " + val + ", valSize : " + valSize);
            expectedStep = expectedStep.add(stepType.getSteps().multiply(BigInteger.valueOf(valSize)));
        }

        void addOperation(StepType stepType, int cnt) {
            expectedStep = expectedStep.add(stepType.getSteps().multiply(BigInteger.valueOf(cnt)));
        }

        BigInteger calcTransactionStep(Transaction tx) {
            // default + input * dataLen
            BigInteger stepUsed = StepType.DEFAULT.getSteps();
            long dataLen = 0;
            if (tx.getData() != null) {
                if ("message".equals(tx.getDataType())) {
                    // tx.getData() returns message with no quotes
                    dataLen = tx.getData().asString().getBytes(StandardCharsets.UTF_8).length + 2;
                } else {
                    dataLen = 2; // curly brace
                    RpcObject rpcObject = tx.getData().asObject();
                    for (String key : rpcObject.keySet()) {
                        // Quotes for key(2) + colon(1) + comma(1)
                        dataLen += 4;
                        dataLen += key.length();
                        if ("params".equals(key)) {
                            RpcObject paramObj = rpcObject.getItem(key).asObject();
                            dataLen += 2; // curly brace
                            for (String param : paramObj.keySet()) {
                                dataLen += paramObj.getItem(param).asString().getBytes(StandardCharsets.UTF_8).length;
                                dataLen += param.getBytes(StandardCharsets.UTF_8).length;
                                // Quotes for key(2) + Quotes for value(2) + colon(1) + comma(1)
                                dataLen += 6;
                            }
                            dataLen -= 1; // subtract last comma
                        } else {
                            dataLen += rpcObject.getItem(key).asString().length();
                            dataLen += 2; // add Quotes for value
                        }
                    }
                    dataLen -= 1; // subtract last comma
                }
            }
            return stepUsed.add(StepType.INPUT.getSteps().multiply(BigInteger.valueOf(dataLen)));
        }

        BigInteger calcDeployStep(Transaction tx, byte[] content, boolean update) {
            // get the default transaction steps first
            BigInteger stepUsed = calcTransactionStep(tx);
            // contractCreate or contractUpdate
            if (update) {
                stepUsed = StepType.CONTRACT_UPDATE.getSteps().add(stepUsed);
            } else {
                stepUsed = StepType.CONTRACT_CREATE.getSteps().add(stepUsed);
            }
            // contractSet * codeLen
            BigInteger codeLen = BigInteger.valueOf(content.length);
            return stepUsed.add(StepType.CONTRACT_SET.getSteps().multiply(codeLen));
        }

        BigInteger calcCallStep(Transaction tx) {
            BigInteger stepUsed = calcTransactionStep(tx);
            return StepType.CONTRACT_CALL.getSteps().add(stepUsed);
        }

        BigInteger calcAcceptStep(Transaction tx, RpcObject params, boolean update) {
            BigInteger callSteps = calcCallStep(tx);
            BigInteger initSteps = calcInitStep(params, update);
            return callSteps.add(initSteps);
        }

        BigInteger calcInitStep(RpcObject params, boolean update) {
            // NOTE: the following calculation can only be applied to hello_world score
            String name = params.getItem("name").asString();
            BigInteger nameLength = BigInteger.valueOf(name.length());
            if (update) {
                return StepType.REPLACE.getSteps().multiply(nameLength);
            } else {
                return StepType.SET.getSteps().multiply(nameLength);
            }
        }

        BigInteger transfer(KeyWallet from, Address to, BigInteger value, String msg) throws Exception {
            BigInteger prevTreasury = txHandler.getBalance(Constants.TREASURY_ADDRESS);
            BigInteger prevBal = txHandler.getBalance(from.getAddress());
            TransactionBuilder.Builder builder = TransactionBuilder.newBuilder()
                    .nid(txHandler.getNetworkId())
                    .from(from.getAddress())
                    .to(to)
                    .value(value)
                    .stepLimit(Constants.DEFAULT_STEPS);
            if (msg != null) {
                builder.message(msg);
            }
            Transaction transaction = builder.build();
            this.expectedStep = calcTransactionStep(transaction);
            Bytes txHash = txHandler.invoke(from, transaction);
            assertSuccess(txHandler.getResult(txHash));
            return getUsedFee(from, value, prevTreasury, prevBal);
        }

        BigInteger deploy(KeyWallet from, Address to, String contentPath, RpcObject params) throws Exception {
            BigInteger prevTreasury = txHandler.getBalance(Constants.TREASURY_ADDRESS);
            BigInteger prevBal = txHandler.getBalance(from.getAddress());
            byte[] content = ZipFile.zipContent(contentPath);
            if (to == null) {
                to = Constants.CHAINSCORE_ADDRESS;
            }
            Transaction transaction = TransactionBuilder.newBuilder()
                    .nid(txHandler.getNetworkId())
                    .from(from.getAddress())
                    .to(to)
                    .stepLimit(new BigInteger("70000000", 16))
                    .deploy(Constants.CONTENT_TYPE_PYTHON, content)
                    .params(params)
                    .build();
            this.expectedStep = calcDeployStep(transaction, content, to != Constants.CHAINSCORE_ADDRESS);
            Bytes txHash = txHandler.invoke(from, transaction);
            TransactionResult result = txHandler.getResult(txHash);
            assertSuccess(result);

            if (govScore.isAuditEnabledOnly()) {
                var governor = txHandler.getChain().governorWallet;
                RpcObject acceptParams = new RpcObject.Builder()
                        .put("txHash", new RpcValue(txHash))
                        .build();
                Transaction acceptTX = TransactionBuilder.newBuilder()
                        .nid(txHandler.getNetworkId())
                        .from(governor.getAddress())
                        .to(Constants.GOV_ADDRESS)
                        .stepLimit(Constants.DEFAULT_STEPS)
                        .call("acceptScore")
                        .params(acceptParams)
                        .build();
                var acceptSteps = calcAcceptStep(acceptTX, params, to != Constants.CHAINSCORE_ADDRESS);
                Bytes acceptHash = txHandler.invoke(governor, acceptTX);
                TransactionResult acceptResult = txHandler.getResult(acceptHash);

                assertSuccess(acceptResult);
                assertEquals(acceptSteps, acceptResult.getStepUsed());
            } else {
                // if Audit is disabled, the sender must pay steps for executing on_install() or on_update()
                var initSteps = calcInitStep(params, to != Constants.CHAINSCORE_ADDRESS);
                expectedStep = expectedStep.add(initSteps);
            }
            this.scoreAddr = new Address(result.getScoreAddress());
            return getUsedFee(from, BigInteger.ZERO, prevTreasury, prevBal);
        }

        BigInteger call(KeyWallet from, Address to, String method, RpcObject params, BigInteger stepLimit) throws Exception {
            return this.call(from, to, method, params, stepLimit, Constants.STATUS_SUCCESS);
        }

        BigInteger call(KeyWallet from, Address to, String method, RpcObject params, BigInteger stepLimit, BigInteger status) throws Exception {
            BigInteger prevTreasury = txHandler.getBalance(Constants.TREASURY_ADDRESS);
            BigInteger prevBal = txHandler.getBalance(from.getAddress());
            TransactionBuilder.Builder builder = TransactionBuilder.newBuilder()
                    .nid(txHandler.getNetworkId())
                    .from(from.getAddress())
                    .to(to)
                    .stepLimit(stepLimit);
            Transaction transaction;
            if (params != null) {
                transaction = builder.call(method).params(params).build();
            } else {
                transaction = builder.call(method).build();
            }
            this.expectedStep = calcCallStep(transaction);

            Bytes txHash = txHandler.invoke(from, transaction);
            TransactionResult result = txHandler.getResult(txHash);
            usedFee = getUsedFee(from, BigInteger.ZERO, prevTreasury, prevBal);
            if (!status.equals(result.getStatus())) {
                throw new TransactionFailureException(result.getFailure());
            }
            return usedFee;
        }

        private BigInteger getUsedFee(KeyWallet from, BigInteger value, BigInteger prevTreasury, BigInteger prevBal)
                throws IOException {
            BigInteger treasury = txHandler.getBalance(Constants.TREASURY_ADDRESS);
            BigInteger bal = txHandler.getBalance(from.getAddress());
            treasuryFee = treasury.subtract(prevTreasury);
            return prevBal.subtract(bal.add(value));
        }
    }

    @Test
    public void testTransfer() throws Exception {
        LOG.infoEntering("testTransfer");
        StepTransaction stx = new StepTransaction();
        LOG.infoEntering("transfer", "simple");
        BigInteger usedFee = stx.transfer(testWallets[0], testWallets[1].getAddress(), ICX, null);
        assertEquals(usedFee, StepType.DEFAULT.getSteps().multiply(STEP_PRICE));
        assertEquals(stx.expectedFee(), usedFee);
        assertEquals(stx.expectedFee(), stx.getTreasuryFee());
        LOG.infoExiting();

        LOG.infoEntering("transfer", "with message");
        usedFee = stx.transfer(testWallets[0], testWallets[2].getAddress(), ICX.multiply(BigInteger.TWO), "Hello");
        assertTrue(usedFee.compareTo(StepType.DEFAULT.getSteps().multiply(STEP_PRICE)) > 0);
        assertEquals(stx.expectedFee(), usedFee);
        assertEquals(stx.expectedFee(), stx.getTreasuryFee());
        LOG.infoExiting();
        LOG.infoExiting();
    }

    @Test
    public void transferFromScore() throws Exception {
        LOG.infoEntering("transferFromScore");
        LOG.infoEntering("deploy", "Scores");
        Score fromScore = HelloWorld.install(txHandler, testWallets[1]);
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld2"))
                .build();
        Score toScore = new HelloWorld(txHandler.deploy(testWallets[2], Score.getFilePath("hello_world2"), params));
        LOG.infoExiting();
        LOG.infoEntering("deposit", "initial funds");
        transferAndCheckResult(txHandler, fromScore.getAddress(), ICX.multiply(BigInteger.TEN));
        LOG.infoExiting();

        LOG.infoEntering("transfer", "to Score");
        StepTransaction stx = new StepTransaction();
        // get the base fee
        params = new RpcObject.Builder()
                .put("to", new RpcValue(toScore.getAddress()))
                .put("amount", new RpcValue(BigInteger.ZERO))
                .build();
        BigInteger baseFee = stx.call(testWallets[1], fromScore.getAddress(), "transferICX", params, Constants.DEFAULT_STEPS);
        BigInteger expectedFee = baseFee.add(STEP_PRICE.multiply(StepType.CONTRACT_CALL.getSteps()));
        // transfer icx and compare with the base fee
        params = new RpcObject.Builder()
                .put("to", new RpcValue(toScore.getAddress()))
                .put("amount", new RpcValue(BigInteger.ONE))
                .build();
        BigInteger usedFee = stx.call(testWallets[1], fromScore.getAddress(), "transferICX", params, Constants.DEFAULT_STEPS);
        assertEquals(expectedFee, usedFee);
        assertEquals(BigInteger.ONE, txHandler.getBalance(toScore.getAddress()));
        LOG.infoExiting();

        LOG.infoEntering("transfer", "to EOA");
        KeyWallet callee = KeyWallet.create();
        params = new RpcObject.Builder()
                .put("to", new RpcValue(callee.getAddress()))
                .put("amount", new RpcValue(BigInteger.ONE))
                .build();
        usedFee = stx.call(testWallets[1], fromScore.getAddress(), "transferICX", params, Constants.DEFAULT_STEPS);
        assertEquals(expectedFee, usedFee);
        assertEquals(BigInteger.ONE, txHandler.getBalance(callee.getAddress()));
        LOG.infoExiting();
        LOG.infoExiting();
    }

    @Test
    public void testDeploy() throws Exception {
        LOG.infoEntering("testDeploy");
        LOG.infoEntering("deploy", "helloWorld");
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        StepTransaction stx = new StepTransaction();
        BigInteger usedFee = stx.deploy(testWallets[0], null, HelloWorld.INSTALL_PATH, params);
        LOG.infoExiting();
        assertEquals(stx.expectedFee(), usedFee);
        if (!chainScore.isAuditEnabled()) {
            assertEquals(stx.expectedFee(), stx.getTreasuryFee());
        }

        LOG.infoEntering("update", "helloWorld");
        Address scoreAddr = stx.getScoreAddress();
        params = new RpcObject.Builder()
                .put("name", new RpcValue("Updated HelloWorld"))
                .build();
        stx = new StepTransaction();
        usedFee = stx.deploy(testWallets[0], scoreAddr, HelloWorld.UPDATE_PATH, params);
        LOG.infoExiting();
        assertEquals(stx.expectedFee(), usedFee);
        if (!chainScore.isAuditEnabled()) {
            assertEquals(stx.expectedFee(), stx.getTreasuryFee());
        }
        LOG.infoExiting();
    }

    private enum VarTest {
        VAR_SET("setToVar", StepType.SET),
        VAR_GET("getFromVar", StepType.GET),
        VAR_REPLACE("setToVar", StepType.REPLACE),
        VAR_EXACT("setToVar", StepType.REPLACE),
        VAR_EDGE("setToVar", StepType.REPLACE),
        VAR_DELETE("delFromVar", StepType.DELETE);

        String[][] params;
        String method;
        StepType stepType;

        VarTest(String method, StepType type) {
            this.method = method;
            this.stepType = type;
        }

        public RpcObject getParams(int type) {
            RpcObject.Builder paramObj = new RpcObject.Builder();
            // set, get, replace, edge, delete
            if (params == null) {
                paramObj = paramObj.put("type", new RpcValue(BigInteger.valueOf(type)));
            } else {
                for (int i = 0; i < type + 1; i++) {
                    paramObj = paramObj.put(params[i][0], new RpcValue(params[i][1]));
                }
            }
            return paramObj.build();
        }
    }

    @Test
    public void testVarDB() throws Exception {
        LOG.infoEntering("testVarDB");
        LOG.infoEntering("deploy", "db_step");
        Score dbScore = txHandler.deploy(testWallets[2], Score.getFilePath("db_step"), null);
        LOG.infoExiting();

        KeyWallet caller = testWallets[3];
        Address scoreAddr = dbScore.getAddress();
        StepTransaction stx = new StepTransaction();
        String[][] initialParams = {
                {"v_int", "0x180"},
                {"v_str", "tortoise"},
                {"v_bytes", new Bytes("tortoise".getBytes()).toString()},
                {"v_addr", testWallets[0].getAddress().toString()},
        };
        String[][] updatedParams = {
                {"v_int", "0x335"},
                {"v_str", "esiotrot"},
                {"v_bytes", new Bytes("esiotrot".getBytes()).toString()},
                {"v_addr", testWallets[1].getAddress().toString()},
        };
        BigInteger[] edgeLimit = new BigInteger[4];

        for (VarTest test : VarTest.values()) {
            if (test == VarTest.VAR_SET || test == VarTest.VAR_REPLACE || test == VarTest.VAR_EDGE) {
                test.params = initialParams;
            } else if (test == VarTest.VAR_EXACT) {
                test.params = updatedParams;
            }
            for (int i = 0; i < initialParams.length; i++) {
                String param = initialParams[i][0];
                String val = initialParams[i][1];
                if (test == VarTest.VAR_EXACT) {
                    val = test.params[i][1];
                }
                BigInteger stepLimit = (test == VarTest.VAR_EXACT) ? edgeLimit[i]
                        : (test == VarTest.VAR_EDGE) ? edgeLimit[i].subtract(BigInteger.ONE)
                        : Constants.DEFAULT_STEPS;
                LOG.infoEntering("invoke", "(" + test + ") method=" + test.method + ", param=" + param +
                        ", val=" + val + ", limit=" + stepLimit);
                try {
                    BigInteger usedFee = stx.call(caller, scoreAddr, test.method, test.getParams(i), stepLimit);
                    assertNotEquals(test, VarTest.VAR_EDGE);
                    stx.addOperation(test.stepType, i, val);
                    assertEquals(stx.expectedFee(), usedFee);
                    assertEquals(stx.expectedFee(), stx.getTreasuryFee());
                    if (test == VarTest.VAR_REPLACE) {
                        edgeLimit[i] = stx.expectedStep();
                    }
                } catch (TransactionFailureException ex) {
                    if (test != VarTest.VAR_EDGE) {
                        throw ex;
                    }
                    RpcObject callParam = new RpcObject.Builder()
                            .put("type", new RpcValue(BigInteger.valueOf(i)))
                            .build();
                    String dbVal = dbScore.call("readFromVar", callParam).asString();
                    LOG.info("dbVal[" + i + "] : " + dbVal);
                    assertEquals(updatedParams[i][1], dbVal);
                    assertEquals(STEP_PRICE.multiply(stepLimit), stx.getUsedFee());
                    assertEquals(STEP_PRICE.multiply(stepLimit), stx.getTreasuryFee());
                }
                LOG.infoExiting();
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void testChainScoreCall() throws Exception {
        LOG.infoEntering("testChainScoreCall");

        LOG.infoEntering("deploy", "hello_world");
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        Score score = txHandler.deploy(testWallets[1], Score.getFilePath("hello_world"), params);
        LOG.infoExiting();

        var currentRevision = chainScore.getRevision();

        LOG.infoEntering("invoke", "HelloWorld.checkRevision() -> ChainSCORE.getRevision()");
        params = new RpcObject.Builder()
                .put("code", new RpcValue(BigInteger.valueOf(currentRevision)))
                .build();
        StepTransaction stx = new StepTransaction();
        var usedFee = stx.call(testWallets[1], score.getAddress(),
                "checkRevision", params, Constants.DEFAULT_STEPS);
        stx.addOperation(StepType.CONTRACT_CALL, 1);
        assertEquals(usedFee, stx.expectedFee());
        var expectedStep = stx.expectedStep();
        LOG.infoExiting();

        LOG.infoEntering("invoke", "HelloWorld.setRevision() -> Governance -> ChainSCORE.setRevision()");
        stx = new StepTransaction();
        usedFee = stx.call(testWallets[1], score.getAddress(),
                "setRevision", params, expectedStep);
        stx.addOperation(StepType.CONTRACT_CALL, 1);
        assertEquals(usedFee, stx.expectedFee());
        LOG.infoExiting();

        LOG.infoExiting();
    }

    @Test
    public void testEstimateStep() throws Exception {
        LOG.infoEntering("testEstimateStep");
        IconService iconService = txHandler.getIconService();
        KeyWallet from = testWallets[3];
        KeyWallet to = KeyWallet.create();

        TransactionBuilder.Builder builder = TransactionBuilder.newBuilder()
                .nid(txHandler.getNetworkId())
                .from(from.getAddress())
                .to(to.getAddress())
                .value(ICX);
        Transaction transaction = builder.build();
        BigInteger estimatedStep = iconService.estimateStep(transaction).execute();
        assertEquals(StepType.DEFAULT.getSteps(), estimatedStep);
        assertThrows(IllegalArgumentException.class, () -> {
            new SignedTransaction(transaction, from);
        });
        assertDoesNotThrow(() -> {
            new SignedTransaction(transaction, from, estimatedStep);
        });

        SignedTransaction signedTransaction = new SignedTransaction(transaction, from, estimatedStep);
        Bytes txHash = iconService.sendTransaction(signedTransaction).execute();
        TransactionResult result = txHandler.getResult(txHash);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        assertEquals(estimatedStep, result.getStepUsed());

        Transaction transaction2 = builder
                .stepLimit(estimatedStep)
                .build();
        // this should override the existing stepLimit
        BigInteger customStep = estimatedStep.add(estimatedStep);
        signedTransaction = new SignedTransaction(transaction2, from, customStep);
        RpcObject properties = signedTransaction.getProperties();
        assertEquals(customStep, properties.getItem("stepLimit").asInteger());
        txHash = iconService.sendTransaction(signedTransaction).execute();
        result = txHandler.getResult(txHash);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        // the actual stepUsed should still be the estimatedStep
        assertEquals(estimatedStep, result.getStepUsed());
        LOG.infoExiting();
    }

    @Test
    public void testVariousCallTest() throws Exception {
        LOG.infoEntering("testVariousCallTest");

        LOG.infoEntering("deploy", "Scores");
        Score fromScore = HelloWorld.install(txHandler, testWallets[4]);
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld2"))
                .build();
        Score toScore = new HelloWorld(txHandler.deploy(testWallets[4], Score.getFilePath("hello_world2"), params));
        LOG.infoExiting();

        LOG.infoEntering("deposit", "initial funds");
        transferAndCheckResult(txHandler, fromScore.getAddress(), ICX.multiply(BigInteger.TEN));
        LOG.infoExiting();

        BigInteger[] expects = {
                Constants.STATUS_SUCCESS,
                Constants.STATUS_FAILURE,
                Constants.STATUS_FAILURE,
                Constants.STATUS_FAILURE,
                Constants.STATUS_SUCCESS,
                Constants.STATUS_FAILURE,
                Constants.STATUS_SUCCESS,
                Constants.STATUS_FAILURE,
                Constants.STATUS_FAILURE,
                Constants.STATUS_FAILURE,
                Constants.STATUS_SUCCESS,
        };
        BigInteger[] extraSteps = {
                STEP_PRICE.multiply(StepType.CONTRACT_CALL.getSteps()),
                STEP_PRICE.multiply(StepType.CONTRACT_CALL.getSteps()),
                STEP_PRICE.multiply(StepType.CONTRACT_CALL.getSteps()),
                STEP_PRICE.multiply(StepType.CONTRACT_CALL.getSteps()),
                STEP_PRICE.multiply(StepType.CONTRACT_CALL.getSteps()),
                STEP_PRICE.multiply(StepType.CONTRACT_CALL.getSteps()),
                STEP_PRICE.multiply(StepType.CONTRACT_CALL.getSteps()),
                STEP_PRICE.multiply(StepType.CONTRACT_CALL.getSteps()),
                STEP_PRICE.multiply(StepType.CONTRACT_CALL.getSteps()),
                STEP_PRICE.multiply(StepType.CONTRACT_CALL.getSteps()),
                STEP_PRICE.multiply(StepType.CONTRACT_CALL.getSteps()),
        };

        for(int i = 0; i < expects.length; i++) {
            LOG.infoEntering("Call", "case " + i);
            StepTransaction stx = new StepTransaction();
            params = new RpcObject.Builder()
                    .put("to", new RpcValue(toScore.getAddress()))
                    .put("method", new RpcValue(BigInteger.valueOf(i)))
                    .build();
            var used = stx.call(testWallets[4], fromScore.getAddress(),
                    "callMethodOf", params, Constants.DEFAULT_STEPS, expects[i]);
            var expected = stx.expectedFee().add(extraSteps[i]);
            assertEquals(expected, used);
            LOG.infoExiting();
        }

        LOG.infoExiting();
    }
}
