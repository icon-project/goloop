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

import example.APITest;
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
    private static KeyWallet calleeWallet;
    private static Score testScore;

    @BeforeAll
    static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);

        ownerWallet = KeyWallet.create();
        calleeWallet = KeyWallet.create();
        transferAndCheckResult(txHandler, ownerWallet.getAddress(), BigInteger.TEN.pow(20));
    }

    @Test
    void testSampleToken() throws Exception {
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

    private Address deployAPITest() throws Exception {
        LOG.infoEntering("deploy", "apiTest");
        testScore = txHandler.deploy(ownerWallet, APITest.class, null);
        LOG.info("scoreAddr = " + testScore.getAddress());
        LOG.infoExiting();
        return testScore.getAddress();
    }

    @Test
    void testAPITestForAddress() throws Exception {
        Address scoreAddr = deployAPITest();
        KeyWallet caller = KeyWallet.create();
        TransactionResult tr;

        // getAddress
        LOG.infoEntering("getAddress", "invoke");
        tr = testScore.invokeAndWaitResult(caller, "getAddress",
                new RpcObject.Builder().put("addr", new RpcValue(testScore.getAddress())).build(),
                0, 100000);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        tr = testScore.invokeAndWaitResult(caller, "getAddress",
                new RpcObject.Builder().put("addr", new RpcValue(caller.getAddress())).build(),
                0, 100000);
        assertEquals(Constants.STATUS_FAIL, tr.getStatus());
        LOG.infoExiting();

        LOG.infoEntering("getAddress", "query");
        RpcItem result = testScore.call("getAddressQuery", null);
        LOG.info("expected (" + scoreAddr + "), got (" + result.asAddress() + ")");
        assertEquals(scoreAddr, result.asAddress());
        LOG.infoExiting();

        // getCaller
        LOG.infoEntering("getCaller", "invoke");
        tr = testScore.invokeAndWaitResult(caller, "getCaller",
                new RpcObject.Builder().put("caller", new RpcValue(caller.getAddress())).build(),
                0, 100000);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        tr = testScore.invokeAndWaitResult(caller, "getCaller",
                new RpcObject.Builder().put("caller", new RpcValue(testScore.getAddress())).build(),
                0, 100000);
        assertEquals(Constants.STATUS_FAIL, tr.getStatus());
        LOG.infoExiting();

        LOG.infoEntering("getCaller", "query");
        result = testScore.call("getCallerQuery", null);
        LOG.info("expected (" + "null" + "), got (" + result + ")");
        assertNull(result);
        LOG.infoExiting();

        // getOrigin
        LOG.infoEntering("getOrigin", "invoke");
        tr = testScore.invokeAndWaitResult(caller, "getOrigin",
                new RpcObject.Builder().put("origin", new RpcValue(caller.getAddress())).build(),
                0, 100000);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        tr = testScore.invokeAndWaitResult(caller, "getOrigin",
                new RpcObject.Builder().put("origin", new RpcValue(testScore.getAddress())).build(),
                0, 100000);
        assertEquals(Constants.STATUS_FAIL, tr.getStatus());
        LOG.infoExiting();

        LOG.infoEntering("getOrigin", "query");
        result = testScore.call("getOriginQuery", null);
        LOG.info("expected (" + "null" + "), got (" + result + ")");
        assertNull(result);
        LOG.infoExiting();

        // getOwner
        LOG.infoEntering("getOwner", "invoke");
        tr = testScore.invokeAndWaitResult(caller, "getOwner",
                new RpcObject.Builder().put("owner", new RpcValue(ownerWallet.getAddress())).build(),
                0, 100000);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        tr = testScore.invokeAndWaitResult(caller, "getOwner",
                new RpcObject.Builder().put("owner", new RpcValue(caller.getAddress())).build(),
                0, 100000);
        assertEquals(Constants.STATUS_FAIL, tr.getStatus());
        LOG.infoExiting();

        LOG.infoEntering("getOwner", "query");
        result = testScore.call("getOwnerQuery", null);
        LOG.info("expected (" + ownerWallet.getAddress() + "), got (" + result.asAddress() + ")");
        assertEquals(ownerWallet.getAddress(), result.asAddress());
        LOG.infoExiting();
    }

    @Test
    void testAPITestForBlock() throws Exception {
        Address scoreAddr = deployAPITest();
        KeyWallet caller = KeyWallet.create();
        TransactionResult tr;
        RpcItem result = RpcValue.NULL;

        // getBlockHeight
        LOG.infoEntering("getBlockHeight", "invoke");
        tr = testScore.invokeAndWaitResult(caller, "getBlockHeight", null, 0, 100000);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + tr.getBlockHeight() + "), got (" + result.asInteger() + ")");
        assertEquals(0, tr.getBlockHeight().compareTo(result.asInteger()));
        LOG.infoExiting();

        LOG.infoEntering("getBlockHeight", "query");
        Block block = iconService.getLastBlock().execute();
        result = testScore.call("getBlockHeightQuery", null);
        LOG.info("expected (" + block.getHeight() + "), got (" + result.asInteger() + ")");
        assertTrue(block.getHeight().compareTo(result.asInteger()) <= 0);
        LOG.infoExiting();

        // getBlockTimestamp
        LOG.infoEntering("getBlockTimestamp", "invoke");
        tr = testScore.invokeAndWaitResult(caller, "getBlockTimestamp", null, 0, 100000);
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
        result = testScore.call("getBlockTimestampQuery", null);
        LOG.info("expected (" + block.getTimestamp() + "), got (" + result.asInteger() + ")");
        assertTrue(block.getTimestamp().compareTo(result.asInteger()) <= 0);
        LOG.infoExiting();
    }

    @Test
    void testAPITestForTransaction() throws Exception {
        Address scoreAddr = deployAPITest();
        KeyWallet caller = KeyWallet.create();
        TransactionResult tr;
        RpcItem result = RpcValue.NULL;

        // getTransactionHash
        LOG.infoEntering("getTransactionHash", "invoke");
        tr = testScore.invokeAndWaitResult(caller, "getTransactionHash", null, 0, 200000);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + tr.getTxHash() + "), got (" + result.asString() + ")");
        assertEquals(tr.getTxHash().toString(), result.asString());
        LOG.infoExiting();

        LOG.infoEntering("getTransactionHash", "query");
        result = testScore.call("getTransactionHashQuery", null);
        LOG.info("expected (" + "null" + "), got (" + result + ")");
        assertNull(result);
        LOG.infoExiting();

        // getTransactionIndex
        LOG.infoEntering("getTransactionIndex", "invoke");
        Bytes[] ids = new Bytes[5];
        for (int i = 0; i < ids.length; i++) {
            ids[i] = testScore.invoke(caller, "getTransactionIndex", null, 0, 200000);
        }
        for (Bytes id : ids) {
            tr = testScore.getResult(id);
            assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
            for (TransactionResult.EventLog e : tr.getEventLogs()) {
                RpcItem data = e.getData().get(0);
                LOG.info("expected (" + tr.getTxIndex() + "), got (" + data.asInteger() + ")");
                assertEquals(tr.getTxIndex(), data.asInteger());
            }
        }
        LOG.infoExiting();

        LOG.infoEntering("getTransactionIndex", "query");
        result = testScore.call("getTransactionIndexQuery", null);
        LOG.info("expected (" + "0" + "), got (" + result.asInteger() + ")");
        assertEquals(BigInteger.ZERO, result.asInteger());
        LOG.infoExiting();

        // getTransactionTimestamp
        LOG.infoEntering("getTransactionTimestamp", "invoke");
        BigInteger steps = BigInteger.valueOf(200000);
        // Add arbitrary milliseconds precision for testing
        BigInteger timestamp = BigInteger.valueOf((System.currentTimeMillis() * 1000L) - (new Random()).nextInt(100));
        Bytes txHash = testScore.invoke(caller, "getTransactionTimestamp", null, null, steps, timestamp, null);
        tr = testScore.getResult(txHash);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        ConfirmedTransaction ctx = iconService.getTransaction(tr.getTxHash()).execute();
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + ctx.getTimestamp() + "), got (" + result.asInteger() + ")");
        assertEquals(0, ctx.getTimestamp().compareTo(result.asInteger()));
        LOG.infoExiting();

        LOG.infoEntering("getTransactionTimestamp", "query");
        result = testScore.call("getTransactionTimestampQuery", null);
        LOG.info("expected (" + "0" + "), got (" + result.asInteger() + ")");
        assertEquals(BigInteger.ZERO, result.asInteger());
        LOG.infoExiting();

        // getTransactionNonce
        LOG.infoEntering("getTransactionNonce", "invoke");
        BigInteger nonce = BigInteger.valueOf(0x12345);
        txHash = testScore.invoke(caller, "getTransactionNonce", null, null, steps, null, nonce);
        tr = testScore.getResult(txHash);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + nonce + "), got (" + result.asInteger() + ")");
        assertEquals(nonce, result.asInteger());
        LOG.infoExiting();

        LOG.infoEntering("getTransactionNonce", "query");
        result = testScore.call("getTransactionNonceQuery", null);
        LOG.info("expected (" + "0" + "), got (" + result.asInteger() + ")");
        assertEquals(BigInteger.ZERO, result.asInteger());
        LOG.infoExiting();
    }

    @Test
    void testAPITestForCoin() throws Exception {
        Address scoreAddr = deployAPITest();
        KeyWallet caller = KeyWallet.create();
        TransactionResult tr;
        RpcItem result = RpcValue.NULL;

        // getValue
        LOG.infoEntering("getValue", "invoke");
        BigInteger coin = BigInteger.TEN.pow(18);
        BigInteger stepLimit = BigInteger.valueOf(100000);
        tr = testScore.invokeAndWaitResult(ownerWallet, "getValue", null, coin, stepLimit);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + coin + "), got (" + result.asInteger() + ")");
        assertEquals(coin, result.asInteger());
        Utils.ensureIcxBalance(txHandler, ownerWallet.getAddress(), 100, 99);
        Utils.ensureIcxBalance(txHandler, scoreAddr, 0, 1);
        LOG.infoExiting();

        LOG.infoEntering("getValue", "query");
        result = testScore.call("getValueQuery", null);
        LOG.info("expected (" + "0" + "), got (" + result.asInteger() + ")");
        assertEquals(BigInteger.ZERO, result.asInteger());
        LOG.infoExiting();

        // getBalance
        LOG.infoEntering("getBalance", "check owner balance");
        BigInteger ownerBalance = iconService.getBalance(ownerWallet.getAddress()).execute();
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(ownerWallet.getAddress()))
                .build();
        tr = testScore.invokeAndWaitResult(caller, "getBalance", params, null, stepLimit);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + ownerBalance + "), got (" + result.asInteger() + ")");
        assertEquals(ownerBalance, result.asInteger());
        LOG.infoExiting();

        LOG.infoEntering("getBalance", "check caller balance");
        tr = testScore.invokeAndWaitResult(caller, "getBalance", null, null, stepLimit);
        assertEquals(Constants.STATUS_SUCCESS, tr.getStatus());
        for (TransactionResult.EventLog e : tr.getEventLogs()) {
            result = e.getData().get(0);
        }
        LOG.info("expected (" + coin + "), got (" + result.asInteger() + ")");
        assertEquals(coin, result.asInteger());
        LOG.infoExiting();
    }
}
