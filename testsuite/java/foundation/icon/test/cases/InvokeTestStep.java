/*
 * Copyright 2020 ICON Foundation
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
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.score.HelloWorld;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

@Tag(Constants.TAG_PY_GOV)
public class InvokeTestStep extends TestBase {
    private static TransactionHandler txHandler;
    private static GovScore govScore;
    private static GovScore.Fee fee;
    private static KeyWallet owner;
    private static Score helloScore;

    private static final long defaultStep = 100;
    private static final long contractCallStep = 10;
    private static final long stepPrice = 1;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);

        owner = KeyWallet.create();
        Address[] addresses = {owner.getAddress(), chain.governorWallet.getAddress()};
        transferAndCheckResult(txHandler, addresses, ICX);

        govScore = new GovScore(txHandler);
        helloScore = HelloWorld.install(txHandler, owner);

        fee = govScore.getFee();
        govScore.setMaxStepLimit("invoke", BigInteger.valueOf(10000000));
        govScore.setMaxStepLimit("query", BigInteger.valueOf(10000000));
        govScore.setStepCost("default", BigInteger.valueOf(defaultStep));
        govScore.setStepCost("contractCall", BigInteger.valueOf(contractCallStep));
        govScore.setStepPrice(BigInteger.valueOf(stepPrice));
    }

    @AfterAll
    public static void destroy() throws Exception {
        govScore.setFee(fee);
    }

    @Test
    public void notEnoughStepLimit() throws Exception{
        LOG.infoEntering("notEnoughStepLimit");
        KeyWallet testWallet = KeyWallet.create();
        final long needStep = defaultStep + contractCallStep;
        final long needValue = needStep * stepPrice;
        final long preValidationFailureStep = defaultStep - 1;
        // expected {preValidation failure, transaction execution failure, transaction execution success}
        for (long step : new long[]{preValidationFailureStep, needStep - 1, needStep}) {
            try {
                BigInteger sub = BigInteger.valueOf(needValue).subtract(txHandler.getBalance(testWallet.getAddress()));
                if (sub.compareTo(BigInteger.ZERO) > 0) {
                    transferAndCheckResult(txHandler, testWallet.getAddress(), sub);
                }
                LOG.infoEntering("invoke", "step=" + step);
                TransactionResult result = helloScore.invokeAndWaitResult(
                        testWallet, "hello", null, BigInteger.ZERO, BigInteger.valueOf(step));
                if (step < needStep) {
                    assertFailure(result);
                } else {
                    assertSuccess(result);
                }
            } catch (RpcError e) {
                LOG.info("RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
                assertEquals(preValidationFailureStep, step);
            } finally {
                LOG.infoExiting();
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void notEnoughBalance() throws Exception {
        LOG.infoEntering("notEnoughBalance");
        KeyWallet testWallet = KeyWallet.create();
        final long needStep = defaultStep + contractCallStep;
        final long needValue = needStep * stepPrice;
        for (long value : new long[]{needValue, needValue - 1}) {
            transferAndCheckResult(txHandler, testWallet.getAddress(), BigInteger.valueOf(value));
            LOG.infoEntering("invoke", "value=" + value);
            try {
                TransactionResult result = helloScore.invokeAndWaitResult(
                        testWallet, "hello", null, BigInteger.ZERO, BigInteger.valueOf(needStep));
                assertSuccess(result);
                assertEquals(value, needValue);
                assertEquals(BigInteger.ZERO, txHandler.getBalance(testWallet.getAddress()));
            } catch (ResultTimeoutException e) {
                assertTrue(value < needValue);
                assertEquals(BigInteger.valueOf(value), txHandler.getBalance(testWallet.getAddress()));
                LOG.info("Expected exception: msg=" + e.getMessage());
            }
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    public void callWithValue() throws Exception {
        LOG.infoEntering("callWithValue");
        final long needStep = defaultStep + contractCallStep;
        final long needValue = needStep * stepPrice;
        final long testVal = 10 * stepPrice;
        KeyWallet testWallet = KeyWallet.create();
        transferAndCheckResult(txHandler, testWallet.getAddress(), BigInteger.valueOf(testVal + needValue));
        assertSuccess(helloScore.invokeAndWaitResult(
                testWallet, "transfer", null, BigInteger.valueOf(testVal), BigInteger.valueOf(needStep)));

        RpcObject params = new RpcObject.Builder()
                .put("_owner", new RpcValue(testWallet.getAddress()))
                .build();
        BigInteger expectedBal = helloScore.call("balanceOf", params).asInteger();
        assertEquals(BigInteger.valueOf(testVal), expectedBal);
        assertEquals(BigInteger.ZERO, txHandler.getBalance(testWallet.getAddress()));
        assertEquals(expectedBal, txHandler.getBalance(helloScore.getAddress()));
        LOG.infoExiting();
    }

    @Test
    public void infiniteInterCall() throws Exception {
        LOG.infoEntering("infiniteInterCall");
        LOG.infoEntering("deploy", "another helloWorld");
        Score anotherScore = HelloWorld.install(txHandler, owner);
        LOG.infoExiting();

        KeyWallet caller = KeyWallet.create();
        transferAndCheckResult(txHandler, caller.getAddress(), ICX);
        final long testStep = defaultStep + contractCallStep * 5;
        BigInteger[] limits = {ICX, BigInteger.valueOf(testStep)};
        for (BigInteger limit : limits) {
            LOG.infoEntering("invoke", "with stepLimit=" + limit);
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(anotherScore.getAddress().toString()))
                    .put("call_cnt", new RpcValue(BigInteger.ZERO))
                    .build();
            TransactionResult result = helloScore.invokeAndWaitResult(
                    caller, "infinite_intercall", params, BigInteger.ZERO, limit);
            // Maximum recursion depth exceeded and OutOfStep are expected
            assertFailure(result);
            LOG.infoExiting();
        }
        LOG.info("sender's balance=" + txHandler.getBalance(caller.getAddress()));
        LOG.infoExiting();
    }

    @Test
    public void notEnoughBalToCall() throws Exception {
        LOG.infoEntering("notEnoughBalToCall");
        KeyWallet caller = KeyWallet.create();
        transferAndCheckResult(txHandler, caller.getAddress(), ICX);
        BigInteger prevBal = txHandler.getBalance(caller.getAddress());

        final long needStep = defaultStep + contractCallStep;
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("Alice"))
                .build();
        assertSuccess(helloScore.invokeAndWaitResult(
                caller, "helloWithName", params, BigInteger.ZERO, BigInteger.valueOf(needStep)));

        BigInteger curBal = txHandler.getBalance(caller.getAddress());
        BigInteger cost = prevBal.subtract(curBal);
        LOG.info("cost=" + cost);

        KeyWallet testWallet = KeyWallet.create();
        BigInteger testValue = cost.subtract(BigInteger.ONE);
        transferAndCheckResult(txHandler, testWallet.getAddress(), testValue);
        BigInteger tBal = txHandler.getBalance(testWallet.getAddress());
        assertEquals(tBal, testValue);

        assertFailure(helloScore.invokeAndWaitResult(
                testWallet, "helloWithName", params, BigInteger.ZERO, testValue));
        tBal = txHandler.getBalance(testWallet.getAddress());
        assertEquals(tBal.compareTo(BigInteger.ZERO), 0);
        LOG.infoExiting();
    }
}
