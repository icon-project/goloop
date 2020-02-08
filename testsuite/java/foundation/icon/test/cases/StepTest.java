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
import foundation.icon.test.common.Utils;
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
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;

@Tag(Constants.TAG_PY_GOV)
public class StepTest extends TestBase {
    private static TransactionHandler txHandler;
    private static ChainScore chainScore;
    private static GovScore govScore;
    private static GovScore.Fee fee;
    private static KeyWallet[] testWallets;
    private static final int testWalletNum = 4;

    private final static String STEP_DEFAULT = "default";
    private final static String STEP_CONTRACT_CALL = "contractCall";
    private final static String STEP_CONTRACT_CREATE = "contractCreate";
    private final static String STEP_CONTRACT_UPDATE = "contractUpdate";
    private final static String STEP_CONTRACT_SET = "contractSet";
    private final static String STEP_GET = "get";
    private final static String STEP_SET = "set";
    private final static String STEP_REPLACE = "replace";
    private final static String STEP_DELETE = "delete";
    private final static String STEP_INPUT = "input";
    private final static String STEP_EVENTLOG = "eventLog";
    private final static String STEP_APICALL = "apiCall";

    private final static int TYPE_INT = 0;
    private final static int TYPE_STR = 1;
    private final static int TYPE_BYTES = 2;
    private final static int TYPE_ADDR = 3;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        chainScore = new ChainScore(txHandler);
        govScore = new GovScore(txHandler);
        fee = govScore.getFee();

        testWallets = new KeyWallet[testWalletNum];
        Address[] addresses = new Address[testWalletNum];
        for (int i = 0; i < testWalletNum; i++) {
            KeyWallet wallet = KeyWallet.create();
            testWallets[i] = wallet;
            addresses[i] = wallet.getAddress();
        }
        transferAndCheckResult(txHandler, chain.governorWallet.getAddress(), Constants.DEFAULT_BALANCE);
        transferAndCheckResult(txHandler, addresses, Constants.DEFAULT_BALANCE);
        initSteps();
    }

    @AfterAll
    public static void destroy() throws Exception {
        govScore.setFee(fee);
    }

    public static void initSteps() throws Exception {
        LOG.infoEntering("initSteps");
        govScore.setMaxStepLimit("invoke", new BigInteger("10000000000"));
        govScore.setMaxStepLimit("query", new BigInteger("10000000000"));
        govScore.setStepPrice(BigInteger.ONE);
        Map<String, BigInteger> stepCosts = new HashMap<>();
        stepCosts.put(STEP_DEFAULT, BigInteger.valueOf(1000));
        stepCosts.put(STEP_CONTRACT_CALL, BigInteger.valueOf(100));
        stepCosts.put(STEP_CONTRACT_CREATE, BigInteger.valueOf(100000));
        stepCosts.put(STEP_CONTRACT_UPDATE, BigInteger.valueOf(160000));
        stepCosts.put(STEP_CONTRACT_SET, BigInteger.valueOf(30));
        stepCosts.put(STEP_GET, BigInteger.valueOf(0));
        stepCosts.put(STEP_SET, BigInteger.valueOf(320));
        stepCosts.put(STEP_REPLACE, BigInteger.valueOf(80));
        stepCosts.put(STEP_DELETE, BigInteger.valueOf(-240));
        stepCosts.put(STEP_INPUT, BigInteger.valueOf(10));
        stepCosts.put(STEP_EVENTLOG, BigInteger.valueOf(5));
        govScore.setStepCosts(stepCosts);
        LOG.infoExiting();
    }

    static class StepTransaction {
        static Map<String, BigInteger> steps;
        static BigInteger stepPrice;
        BigInteger expectedStep;
        BigInteger usedCoin;
        BigInteger treasuryFee;
        Address scoreAddr;

        StepTransaction() throws Exception {
            if (steps == null) {
                steps = govScore.getStepCosts();
                stepPrice = chainScore.call("getStepPrice", null).asInteger();
            }
        }

        void addOperation(String op, int type, String val) {
            BigInteger stepUsed = steps.get(op);
            long valSize = 0;
            switch (type) {
                case TYPE_INT:
                    int v = Integer.parseInt(val);
                    long bitLen = BigInteger.valueOf(v).bitLength();
                    valSize = bitLen / 8 + 1;
                    // int to byte
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
            expectedStep = expectedStep.add(BigInteger.valueOf(valSize).multiply(stepUsed));
        }

        BigInteger estimatedCoin() {
            return expectedStep.multiply(stepPrice);
        }

        BigInteger calcTransactionStep(Transaction tx) {
            // default + input * data
            BigInteger stepUsed = steps.get(STEP_DEFAULT);
            if (tx.getDataType().equals("message")) {
                // tx.getData() returns message with no quotes
                long dataSize = tx.getData().asString().getBytes(StandardCharsets.UTF_8).length + 2;
                stepUsed = stepUsed.add(BigInteger.valueOf(dataSize).multiply(steps.get(STEP_INPUT)));
            } else {
                int dataLen = 2; // curly brace
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
                stepUsed = stepUsed.add(BigInteger.valueOf(dataLen).multiply(steps.get(STEP_INPUT)));
            }
            return stepUsed;
        }

        BigInteger calcDeployStep(Transaction tx, byte[] content, boolean update) throws IOException {
            // get the default transaction steps first
            BigInteger stepUsed = calcTransactionStep(tx);
            if (!chainScore.isAuditEnabled()) {
                // if Audit is disabled, the sender must pay steps for executing on_install() or on_update()
                // NOTE: the following calculation can only be applied to hello_world score
                RpcObject params = tx.getData().asObject().getItem("params").asObject();
                String name = params.getItem("name").asString();
                if (update) {
                    stepUsed = stepUsed.add(steps.get(STEP_REPLACE).multiply(BigInteger.valueOf(name.length())));
                } else {
                    stepUsed = stepUsed.add(steps.get(STEP_SET).multiply(BigInteger.valueOf(name.length())));
                }
            }
            // contractCreate or contractUpdate
            // contractSet * codeLen
            BigInteger codeLen = BigInteger.valueOf(content.length);
            if (update) {
                stepUsed = steps.get(STEP_CONTRACT_UPDATE).add(stepUsed);
            } else {
                stepUsed = steps.get(STEP_CONTRACT_CREATE).add(stepUsed);
            }
            stepUsed = stepUsed.add(steps.get(STEP_CONTRACT_SET).multiply(codeLen));
            return stepUsed;
        }

        BigInteger calcCallStep(Transaction tx) {
            BigInteger stepUsed = calcTransactionStep(tx);
            // contractCall
            stepUsed = steps.get(STEP_CONTRACT_CALL).add(stepUsed);
            return stepUsed;
        }

        // return used coin
        BigInteger transfer(KeyWallet from, Address to, BigInteger value, String msg) throws Exception {
            BigInteger prevTreasury = txHandler.getBalance(Constants.TREASURY_ADDRESS);
            BigInteger prevBal = txHandler.getBalance(from.getAddress());
            Transaction transaction = TransactionBuilder.newBuilder()
                    .nid(txHandler.getNetworkId())
                    .from(from.getAddress())
                    .to(to)
                    .value(value)
                    .stepLimit(new BigInteger("100000"))
                    .message(msg)
                    .build();
            this.expectedStep = calcTransactionStep(transaction);
            Bytes txHash = txHandler.invoke(from, transaction);
            assertSuccess(txHandler.getResult(txHash, Constants.DEFAULT_WAITING_TIME));

            BigInteger bal = txHandler.getBalance(from.getAddress());
            BigInteger treasury = txHandler.getBalance(Constants.TREASURY_ADDRESS);
            treasuryFee = treasury.subtract(prevTreasury);
            return prevBal.subtract(bal.add(value));
        }

        BigInteger deploy(KeyWallet from, Address to, String contentPath, RpcObject params) throws Exception {
            BigInteger prevTreasury = txHandler.getBalance(Constants.TREASURY_ADDRESS);
            BigInteger prevBal = txHandler.getBalance(from.getAddress());
            byte[] content = Utils.zipContent(contentPath);
            if (to == null) {
                to = Constants.CHAINSCORE_ADDRESS;
            }
            Transaction transaction = TransactionBuilder.newBuilder()
                    .nid(txHandler.getNetworkId())
                    .from(from.getAddress())
                    .to(to)
                    .stepLimit(new BigInteger("10000000"))
                    .deploy(Constants.CONTENT_TYPE_PYTHON, content)
                    .params(params)
                    .build();
            this.expectedStep = calcDeployStep(transaction, content, to != Constants.CHAINSCORE_ADDRESS);
            Bytes txHash = txHandler.invoke(from, transaction);
            TransactionResult result = txHandler.getResult(txHash, Constants.DEFAULT_WAITING_TIME);
            assertSuccess(result);

            try {
                txHandler.acceptScoreIfAuditEnabled(txHash);
            } catch (TransactionFailureException ex) {
                LOG.infoExiting();
                throw ex;
            }

            this.scoreAddr = new Address(result.getScoreAddress());
            BigInteger bal = txHandler.getBalance(from.getAddress());
            BigInteger treasury = txHandler.getBalance(Constants.TREASURY_ADDRESS);
            treasuryFee = treasury.subtract(prevTreasury);
            return prevBal.subtract(bal);
        }

        BigInteger getSpentCoinByLastTx() {
            return usedCoin;
        }

        BigInteger call(KeyWallet from, Address to, String method, RpcObject params, BigInteger stepLimit) throws Exception {
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
            TransactionResult result = txHandler.getResult(txHash, Constants.DEFAULT_WAITING_TIME);

            BigInteger bal = txHandler.getBalance(from.getAddress());
            BigInteger treasury = txHandler.getBalance(Constants.TREASURY_ADDRESS);
            treasuryFee = treasury.subtract(prevTreasury);
            usedCoin = prevBal.subtract(bal);
            if (Constants.STATUS_SUCCESS.compareTo(result.getStatus()) != 0) {
                LOG.info("Expected " + result.getFailure());
                throw new TransactionFailureException(result.getFailure());
            }
            return usedCoin;
        }
    }

    @Test
    public void transferStep() throws Exception {
        LOG.infoEntering("transferStep");
        StepTransaction sTx = new StepTransaction();
        LOG.infoEntering("transfer");
        BigInteger usedCoin = sTx.transfer(testWallets[0], testWallets[1].getAddress(), BigInteger.valueOf(1), "HELLO");
        LOG.infoExiting();
        LOG.infoExiting();
        assertEquals(sTx.estimatedCoin(), usedCoin);
        assertEquals(sTx.estimatedCoin(), sTx.treasuryFee);
    }

    @Test
    public void deployStep() throws Exception {
        LOG.infoEntering("deployStep");
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        StepTransaction sTx = new StepTransaction();
        LOG.infoEntering("deploy", "helloWorld");
        BigInteger usedCoin = sTx.deploy(testWallets[0], null, HelloWorld.INSTALL_PATH, params);
        LOG.infoExiting();
        assertEquals(sTx.estimatedCoin(), usedCoin);
        if (!chainScore.isAuditEnabled()) {
            assertEquals(sTx.estimatedCoin(), sTx.treasuryFee);
        }

        Address scoreAddr = sTx.scoreAddr;
        params = new RpcObject.Builder()
                .put("name", new RpcValue("Updated HelloWorld"))
                .build();
        sTx = new StepTransaction();
        LOG.infoEntering("update", "helloWorld");
        usedCoin = sTx.deploy(testWallets[0], scoreAddr, HelloWorld.UPDATE_PATH, params);
        LOG.infoExiting();
        assertEquals(sTx.estimatedCoin(), usedCoin);
        if (!chainScore.isAuditEnabled()) {
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

        String[][] params;
        String method;
        String op;

        VarTest(String method, String op) {
            this.method = method;
            this.op = op;
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
        KeyWallet scoreOwner = testWallets[2];
        KeyWallet caller = testWallets[3];
        LOG.infoEntering("deploy", "db_step");
        Score dbScore = txHandler.deploy(scoreOwner, Constants.SCORE_DB_STEP_PATH, null);
        LOG.infoExiting();

        Address scoreAddr = dbScore.getAddress();
        StepTransaction sTx = new StepTransaction();
        String[][] params = {
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

        for (VarTest test : VarTest.values()) {
            if (test == VarTest.VAR_EDGE) {
                test.params = new String[][]{
                        {"v_int", "821"},
                        {"v_str", "POOL"},
                        {"v_bytes", new Bytes("PARAM BYTES".getBytes()).toString()},
                        {"v_addr", testWallets[1].getAddress().toString()},
                };
            }
            else if (test != VarTest.VAR_DELETE && test != VarTest.VAR_GET) {
                test.params = params;
            }
            for (int i = 0; i < 4; i++) {
                String val = params[i][1];
                if (test == VarTest.VAR_EDGE) {
                    val = test.params[i][1];
                }
                LOG.infoEntering("invoke", "(" + test + ") method=" + test.method + ", param=" + params[i][0] + ", val=" + val);
                BigInteger stepLimit = (test == VarTest.VAR_EDGE) ? edgeLimit[i].subtract(BigInteger.ONE) : BigInteger.valueOf(10000);
                BigInteger usedCoin;
                try {
                    usedCoin = sTx.call(caller, scoreAddr, test.method, test.getParams(i), stepLimit);
                    assertNotEquals(test, VarTest.VAR_EDGE);
                    sTx.addOperation(test.op, i, params[i][1]);
                    // TYPE_INT, TYPE_STR, TYPE_BYTES, TYPE_ADDR
                    assertEquals(sTx.estimatedCoin(), usedCoin);
                    assertEquals(sTx.estimatedCoin(), sTx.treasuryFee);
                    if (test == VarTest.VAR_REPLACE) {
                        edgeLimit[i] = sTx.estimatedCoin();
                    }
                }
                catch (TransactionFailureException ex) {
                    if (test != VarTest.VAR_EDGE) {
                        throw ex;
                    }
                    usedCoin = sTx.getSpentCoinByLastTx();

                    RpcObject callParam = new RpcObject.Builder()
                            .put("type", new RpcValue(BigInteger.valueOf(i)))
                            .build();
                    String dbVal = dbScore.call("readFromVar", callParam).asString();
                    LOG.info("dbVal[" + i + "] : " + dbVal);
                    if (i == 0) {
                        BigInteger v = new BigInteger(dbVal.substring("0x".length()), 16);
                        LOG.info("v = " + v);
                    } else if (i == 2) {
                        LOG.info("byte[] : " + Arrays.toString(dbVal.getBytes()));
                    }
                    // TYPE_INT, TYPE_STR, TYPE_BYTES, TYPE_ADDR
                    assertEquals(stepLimit, usedCoin);
                    assertEquals(stepLimit, sTx.treasuryFee);
                }
                LOG.infoExiting();
            }
        }
        LOG.infoExiting();
    }
}
