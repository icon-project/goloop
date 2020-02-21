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

import foundation.icon.ee.util.Crypto;
import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Block;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.ConfirmedTransaction;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.SampleTokenScore;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import testcases.APITest;

import java.math.BigInteger;
import java.util.Random;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

@Tag(Constants.TAG_JAVA_SCORE)
class JavaScoreTest extends TestBase {
    private static IconService iconService;
    private static TransactionHandler txHandler;
    private static KeyWallet ownerWallet;
    private static Score testScore;

    @BeforeAll
    static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);

        ownerWallet = KeyWallet.create();
        transferAndCheckResult(txHandler, ownerWallet.getAddress(), BigInteger.TEN.pow(20));
    }

    @Test
    public void testSampleToken() throws Exception {
        KeyWallet calleeWallet = KeyWallet.create();

        // 1. deploy
        BigInteger decimals = BigInteger.valueOf(18);
        BigInteger initialSupply = BigInteger.valueOf(1000);
        SampleTokenScore tokenScore = SampleTokenScore.mustDeploy(txHandler, ownerWallet,
                decimals, initialSupply, Constants.CONTENT_TYPE_JAVA);

        // 2. balanceOf
        LOG.infoEntering("balanceOf", "owner (initial)");
        BigInteger oneToken = BigInteger.TEN.pow(decimals.intValue());
        BigInteger totalSupply = oneToken.multiply(initialSupply);
        BigInteger bal = callBalanceOf(tokenScore, ownerWallet.getAddress()).asInteger();
        LOG.info("expected (" + totalSupply + "), got (" + bal + ")");
        assertEquals(totalSupply, bal);
        LOG.infoExiting();

        // 3. transfer #1
        LOG.infoEntering("transfer", "#1");
        assertSuccess(invokeTransfer(tokenScore, ownerWallet, calleeWallet.getAddress(), oneToken, true));
        LOG.infoExiting();

        // 3.1 transfer #2
        LOG.infoEntering("transfer", "#2");
        assertSuccess(invokeTransfer(tokenScore, ownerWallet, calleeWallet.getAddress(), oneToken, false));
        LOG.infoExiting();

        // 4. check balance of callee
        LOG.infoEntering("balanceOf", "callee");
        BigInteger expected = oneToken.add(oneToken);
        bal = callBalanceOf(tokenScore, calleeWallet.getAddress()).asInteger();
        LOG.info("expected (" + expected + "), got (" + bal + ")");
        assertEquals(expected, bal);
        LOG.infoExiting();

        // 5. check balance of owner
        LOG.infoEntering("balanceOf", "owner");
        expected = totalSupply.subtract(expected);
        bal = callBalanceOf(tokenScore, ownerWallet.getAddress()).asInteger();
        LOG.info("expected (" + expected + "), got (" + bal + ")");
        assertEquals(expected, bal);
        LOG.infoExiting();
    }

    private RpcItem callBalanceOf(Score score, Address addr) throws Exception {
        RpcObject params = new RpcObject.Builder()
                .put("_owner", new RpcValue(addr.toString()))
                .build();
        return score.call("balanceOf", params);
    }

    private TransactionResult invokeTransfer(Score score, Wallet from, Address to, BigInteger value,
                                             boolean includeData) throws Exception {
        RpcObject.Builder builder = new RpcObject.Builder()
                .put("_to", new RpcValue(to))
                .put("_value", new RpcValue(value));
        if (includeData) {
            builder.put("_data", new RpcValue("Hello".getBytes()));
        }
        return score.invokeAndWaitResult(from, "transfer", builder.build());
    }

    private Score deployAPITest() throws Exception {
        if (testScore == null) {
            LOG.infoEntering("deploy", "apiTest");
            testScore = txHandler.deploy(ownerWallet, APITest.class, null);
            LOG.info("scoreAddr = " + testScore.getAddress());
            LOG.infoExiting();
        }
        return testScore;
    }

    static class TestCase {
        private final String method;
        private final RpcObject params;
        private final BigInteger expectedStatus;

        TestCase(String method, RpcObject params, BigInteger expectedStatus) {
            this.method = method;
            this.params = params;
            this.expectedStatus = expectedStatus;
        }
    }

    @Test
    public void testAPIForAddress() throws Exception {
        Score apiScore = deployAPITest();
        KeyWallet caller = KeyWallet.create();
        TransactionResult tr;

        LOG.infoEntering("invoke");
        TestCase[] testCases = {
                new TestCase("getAddress", new RpcObject.Builder()
                        .put("addr", new RpcValue(apiScore.getAddress())).build(),
                        Constants.STATUS_SUCCESS),
                new TestCase("getAddress", new RpcObject.Builder()
                        .put("addr", new RpcValue(caller.getAddress())).build(),
                        Constants.STATUS_FAILURE),
                new TestCase("getCaller", new RpcObject.Builder()
                        .put("caller", new RpcValue(caller.getAddress())).build(),
                        Constants.STATUS_SUCCESS),
                new TestCase("getCaller", new RpcObject.Builder()
                        .put("caller", new RpcValue(ownerWallet.getAddress())).build(),
                        Constants.STATUS_FAILURE),
                new TestCase("getOrigin", new RpcObject.Builder()
                        .put("origin", new RpcValue(caller.getAddress())).build(),
                        Constants.STATUS_SUCCESS),
                new TestCase("getOrigin", new RpcObject.Builder()
                        .put("origin", new RpcValue(ownerWallet.getAddress())).build(),
                        Constants.STATUS_FAILURE),
                new TestCase("getOwner", new RpcObject.Builder()
                        .put("owner", new RpcValue(ownerWallet.getAddress())).build(),
                        Constants.STATUS_SUCCESS),
                new TestCase("getOwner", new RpcObject.Builder()
                        .put("owner", new RpcValue(caller.getAddress())).build(),
                        Constants.STATUS_FAILURE),
        };

        Bytes[] ids = new Bytes[testCases.length];
        int cnt = 0;
        for (TestCase tc : testCases) {
            LOG.info(tc.method);
            ids[cnt++] = apiScore.invoke(caller, tc.method, tc.params);
        }
        for (int i = 0; i < cnt; i++) {
            tr = txHandler.getResult(ids[i]);
            assertStatus(testCases[i].expectedStatus, tr);
            if (tr.getFailure() != null) {
                LOG.info("Expected " + tr.getFailure());
            }
        }
        LOG.infoExiting();

        LOG.infoEntering("getAddress", "query");
        RpcItem result = apiScore.call("getAddressQuery", null);
        LOG.info("expected (" + apiScore.getAddress() + "), got (" + result.asAddress() + ")");
        assertEquals(apiScore.getAddress(), result.asAddress());
        LOG.infoExiting();

        LOG.infoEntering("getCaller", "query");
        result = apiScore.call("getCallerQuery", null);
        LOG.info("expected (" + "null" + "), got (" + result + ")");
        assertNull(result);
        LOG.infoExiting();

        LOG.infoEntering("getOrigin", "query");
        result = apiScore.call("getOriginQuery", null);
        LOG.info("expected (" + "null" + "), got (" + result + ")");
        assertNull(result);
        LOG.infoExiting();

        LOG.infoEntering("getOwner", "query");
        result = apiScore.call("getOwnerQuery", null);
        LOG.info("expected (" + ownerWallet.getAddress() + "), got (" + result.asAddress() + ")");
        assertEquals(ownerWallet.getAddress(), result.asAddress());
        LOG.infoExiting();
    }

    @Test
    public void testAPIForBlock() throws Exception {
        Score apiScore = deployAPITest();
        KeyWallet caller = KeyWallet.create();
        TransactionResult tr;
        RpcItem result = RpcValue.NULL;

        // getBlockHeight
        LOG.infoEntering("getBlockHeight", "invoke");
        tr = apiScore.invokeAndWaitResult(caller, "getBlockHeight", null);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + tr.getBlockHeight() + "), got (" + result.asInteger() + ")");
        assertEquals(0, tr.getBlockHeight().compareTo(result.asInteger()));
        LOG.infoExiting();

        LOG.infoEntering("getBlockHeight", "query");
        Block block = iconService.getLastBlock().execute();
        result = apiScore.call("getBlockHeightQuery", null);
        LOG.info("expected (" + block.getHeight() + "), got (" + result.asInteger() + ")");
        assertTrue(block.getHeight().compareTo(result.asInteger()) <= 0);
        LOG.infoExiting();

        // getBlockTimestamp
        LOG.infoEntering("getBlockTimestamp", "invoke");
        tr = apiScore.invokeAndWaitResult(caller, "getBlockTimestamp", null);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        block = iconService.getBlock(tr.getBlockHeight()).execute();
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + block.getTimestamp() + "), got (" + result.asInteger() + ")");
        assertEquals(0, block.getTimestamp().compareTo(result.asInteger()));
        LOG.infoExiting();

        LOG.infoEntering("getBlockTimestamp", "query");
        block = iconService.getLastBlock().execute();
        result = apiScore.call("getBlockTimestampQuery", null);
        LOG.info("expected (" + block.getTimestamp() + "), got (" + result.asInteger() + ")");
        assertTrue(block.getTimestamp().compareTo(result.asInteger()) <= 0);
        LOG.infoExiting();
    }

    @Test
    public void testAPIForTransaction() throws Exception {
        Score apiScore = deployAPITest();
        KeyWallet caller = KeyWallet.create();
        TransactionResult tr;
        RpcItem result = RpcValue.NULL;

        // getTransactionHash
        LOG.infoEntering("getTransactionHash", "invoke");
        tr = apiScore.invokeAndWaitResult(caller, "getTransactionHash", null);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + tr.getTxHash() + "), got (" + result.asString() + ")");
        assertEquals(tr.getTxHash().toString(), result.asString());
        LOG.infoExiting();

        LOG.infoEntering("getTransactionHash", "query");
        result = apiScore.call("getTransactionHashQuery", null);
        LOG.info("expected (" + "null" + "), got (" + result + ")");
        assertNull(result);
        LOG.infoExiting();

        // getTransactionIndex
        LOG.infoEntering("getTransactionIndex", "invoke");
        Bytes[] ids = new Bytes[5];
        for (int i = 0; i < ids.length; i++) {
            ids[i] = apiScore.invoke(caller, "getTransactionIndex", null);
        }
        for (Bytes id : ids) {
            tr = apiScore.getResult(id);
            assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
            for (TransactionResult.EventLog e : tr.getEventLogs()) {
                RpcItem data = e.getData().get(0);
                LOG.info("expected (" + tr.getTxIndex() + "), got (" + data.asInteger() + ")");
                assertEquals(tr.getTxIndex(), data.asInteger());
            }
        }
        LOG.infoExiting();

        LOG.infoEntering("getTransactionIndex", "query");
        result = apiScore.call("getTransactionIndexQuery", null);
        LOG.info("expected (" + "0" + "), got (" + result.asInteger() + ")");
        assertEquals(BigInteger.ZERO, result.asInteger());
        LOG.infoExiting();

        // getTransactionTimestamp
        LOG.infoEntering("getTransactionTimestamp", "invoke");
        BigInteger steps = BigInteger.valueOf(200000);
        // Add arbitrary milliseconds precision for testing
        BigInteger timestamp = BigInteger.valueOf((System.currentTimeMillis() * 1000L) - (new Random()).nextInt(100));
        Bytes tid = apiScore.invoke(caller, "getTransactionTimestamp", null, null, steps, timestamp, null);
        tr = apiScore.getResult(tid);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        ConfirmedTransaction ctx = iconService.getTransaction(tr.getTxHash()).execute();
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + ctx.getTimestamp() + "), got (" + result.asInteger() + ")");
        assertEquals(0, ctx.getTimestamp().compareTo(result.asInteger()));
        LOG.infoExiting();

        LOG.infoEntering("getTransactionTimestamp", "query");
        result = apiScore.call("getTransactionTimestampQuery", null);
        LOG.info("expected (" + "0" + "), got (" + result.asInteger() + ")");
        assertEquals(BigInteger.ZERO, result.asInteger());
        LOG.infoExiting();

        // getTransactionNonce
        LOG.infoEntering("getTransactionNonce", "invoke");
        BigInteger nonce = BigInteger.valueOf(0x12345);
        tid = apiScore.invoke(caller, "getTransactionNonce", null, null, steps, null, nonce);
        tr = apiScore.getResult(tid);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + nonce + "), got (" + result.asInteger() + ")");
        assertEquals(nonce, result.asInteger());
        LOG.infoExiting();

        LOG.infoEntering("getTransactionNonce", "query");
        result = apiScore.call("getTransactionNonceQuery", null);
        LOG.info("expected (" + "0" + "), got (" + result.asInteger() + ")");
        assertEquals(BigInteger.ZERO, result.asInteger());
        LOG.infoExiting();
    }

    @Test
    public void testAPIForCoin() throws Exception {
        Score apiScore = deployAPITest();
        KeyWallet caller = KeyWallet.create();
        TransactionResult tr;
        RpcItem result = RpcValue.NULL;

        // getValue
        LOG.infoEntering("getValue", "invoke");
        BigInteger coin = BigInteger.TEN.pow(18);
        BigInteger stepLimit = BigInteger.valueOf(100000);
        tr = apiScore.invokeAndWaitResult(ownerWallet, "getValue", null, coin, stepLimit);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + coin + "), got (" + result.asInteger() + ")");
        assertEquals(coin, result.asInteger());
        Utils.ensureIcxBalance(txHandler, ownerWallet.getAddress(), 100, 99);
        Utils.ensureIcxBalance(txHandler, apiScore.getAddress(), 0, 1);
        LOG.infoExiting();

        LOG.infoEntering("getValue", "query");
        result = apiScore.call("getValueQuery", null);
        LOG.info("expected (" + "0" + "), got (" + result.asInteger() + ")");
        assertEquals(BigInteger.ZERO, result.asInteger());
        LOG.infoExiting();

        // getBalance
        LOG.infoEntering("getBalance", "check owner balance");
        BigInteger ownerBalance = txHandler.getBalance(ownerWallet.getAddress());
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(ownerWallet.getAddress()))
                .build();
        tr = apiScore.invokeAndWaitResult(caller, "getBalance", params, null, stepLimit);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + ownerBalance + "), got (" + result.asInteger() + ")");
        assertEquals(ownerBalance, result.asInteger());
        LOG.infoExiting();

        LOG.infoEntering("getBalance", "query");
        result = apiScore.call("getBalanceQuery", params);
        LOG.info("expected (" + ownerBalance + "), got (" + result.asInteger() + ")");
        assertEquals(ownerBalance, result.asInteger());
        LOG.infoExiting();

        LOG.infoEntering("getBalance", "check score balance");
        tr = apiScore.invokeAndWaitResult(caller, "getBalance", null, null, stepLimit);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + coin + "), got (" + result.asInteger() + ")");
        assertEquals(coin, result.asInteger());
        LOG.infoExiting();

        LOG.infoEntering("getBalance", "query");
        result = apiScore.call("getBalanceQuery", null);
        LOG.info("expected (" + coin + "), got (" + result.asInteger() + ")");
        assertEquals(coin, result.asInteger());
        LOG.infoExiting();
    }

    @Test
    public void testAPIForSHA3_256() throws Exception {
        Score apiScore = deployAPITest();
        KeyWallet caller = KeyWallet.create();
        TransactionResult tr;
        RpcItem result = RpcValue.NULL;

        // computeHash
        LOG.infoEntering("computeHash", "invoke");
        byte[] data = "Hello world".getBytes();
        RpcObject params = new RpcObject.Builder()
                .put("data", new RpcValue(data))
                .build();
        tr = apiScore.invokeAndWaitResult(caller, "computeHash", params);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        Bytes expected = new Bytes(Crypto.sha3_256(data));
        LOG.info("expected (" + expected + "), got (" + result.asString() + ")");
        assertEquals(expected.toString(), result.asString());
        LOG.infoExiting();

        LOG.infoEntering("computeHash", "query");
        result = apiScore.call("computeHashQuery", params);
        LOG.info("result=" + result);
        LOG.info("expected (" + expected + "), got (" + result.asString() + ")");
        assertEquals(expected.toString(), result.asString());
        LOG.infoExiting();
    }
}
