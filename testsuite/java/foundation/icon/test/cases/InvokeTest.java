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
import foundation.icon.icx.crypto.IconKeys;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.score.HelloWorld;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.fail;

@Tag(Constants.TAG_PY_SCORE)
public class InvokeTest extends TestBase {
    private static TransactionHandler txHandler;
    private static KeyWallet callerWallet;
    private static Score helloScore;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        callerWallet = KeyWallet.create();
        helloScore = HelloWorld.install(txHandler, KeyWallet.create());
    }

    @Test
    public void invalidScoreAddr() throws Exception {
        LOG.infoEntering("invalidScoreAddr");
        Address invalidAddr = new Address(Address.AddressPrefix.CONTRACT,
                                          IconKeys.getAddressHash(KeyWallet.create().getPublicKey().toByteArray()));
        Score invalidScore = new Score(txHandler, invalidAddr);
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("Alice"))
                .build();
        assertFailure(invalidScore.invokeAndWaitResult(callerWallet, "helloWithName", params,
                null, Constants.DEFAULT_STEPS));
        LOG.infoExiting();
    }

    @Test
    public void invalidMethodName() throws Exception {
        LOG.infoEntering("invalidMethodName");
        final String[] methods = new String[]{"helloWithName", "helloWithName2", "hi"};
        Bytes[] hashes = new Bytes[3];
        int cnt = 0;
        for (String method : methods) {
            RpcObject params = new RpcObject.Builder()
                    .put("name", new RpcValue("Alice"))
                    .build();
            hashes[cnt++] = helloScore.invoke(callerWallet, method, params);
        }
        for (int i = 0; i < cnt; i++) {
            LOG.infoEntering("check", "method=" + methods[i]);
            if (i == 0) {
                assertSuccess(txHandler.getResult(hashes[i]));
            } else {
                assertFailure(txHandler.getResult(hashes[i]));
            }
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    public void invalidParamName() throws Exception {
        LOG.infoEntering("invalidParamName");
        for (String key : new String[]{"name", "wrong"}) {
            LOG.infoEntering("invoke", "key=" + key);
            RpcObject params = new RpcObject.Builder()
                    .put(key, new RpcValue("Alice"))
                    .build();
            TransactionResult result = helloScore.invokeAndWaitResult(callerWallet, "helloWithName", params,
                    null, Constants.DEFAULT_STEPS);
            if (key.equals("name")) {
                assertSuccess(result);
            } else {
                assertFailure(result);
            }
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    public void unexpectedParams() throws Exception {
        LOG.infoEntering("unexpectedParams");
        String[][] params = new String[][]{{}, {"age"}, {"name"}, {"name", "age"}, {"name", "etc"}, {"name", "age", "etc"}};
        Bytes[] hashes = new Bytes[params.length];
        String[] paramStrs = new String[params.length];
        for (int i = 0; i < params.length; i++) {
            RpcObject.Builder builder = new RpcObject.Builder();
            for (String param : params[i]) {
                builder.put(param, new RpcValue("Alice"));
            }
            RpcObject objParam = builder.build();
            hashes[i] = helloScore.invoke(callerWallet, "helloWithName", objParam);
            paramStrs[i] = String.join(", ", params[i]);
        }
        for (int i = 0; i < hashes.length; i++) {
            LOG.infoEntering("check", "params={" + paramStrs[i] + "}");
            if (i == 2 || i == 3) {
                assertSuccess(txHandler.getResult(hashes[i]));
            } else {
                assertFailure(txHandler.getResult(hashes[i]));
            }
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    public void timeoutCallInfiniteLoop() throws Exception {
        LOG.infoEntering("invoke", "infiniteLoop");
        assertFailure(helloScore.invokeAndWaitResult(callerWallet, "infiniteLoop", null,
                null, Constants.DEFAULT_STEPS));
        LOG.infoExiting();
    }

    @Test
    public void testMaxBufferSize() throws Exception {
        LOG.infoEntering("testMaxBufferSize");
        LOG.infoEntering("invoke", "success case");
        RpcObject params = new RpcObject.Builder()
                .put("size", new RpcValue(BigInteger.valueOf(600)))
                .build();
        assertSuccess(helloScore.invokeAndWaitResult(callerWallet, "testMaxBufferSize", params,
                null, Constants.DEFAULT_STEPS));
        LOG.infoExiting();

        LOG.infoEntering("invoke", "failure case");
        params = new RpcObject.Builder()
                .put("size", new RpcValue(BigInteger.valueOf(1000)))
                .build();
        assertFailure(helloScore.invokeAndWaitResult(callerWallet, "testMaxBufferSize", params,
                null, Constants.DEFAULT_STEPS));
        LOG.infoExiting();

        LOG.infoEntering("cleanup");
        params = new RpcObject.Builder()
                .put("size", new RpcValue(BigInteger.ONE))
                .build();
        assertFailure(helloScore.invokeAndWaitResult(callerWallet, "testMaxBufferSize", params,
                null, Constants.DEFAULT_STEPS));
        LOG.infoExiting();
        LOG.infoExiting();
    }

    @Test
    public void invalidSignature() throws Exception {
        LOG.infoEntering("invalidSignature");
        LOG.infoEntering("setup", "test wallets");
        KeyWallet[] testWallets = new KeyWallet[10];
        for (int i = 0; i < testWallets.length; i++) {
            testWallets[i] = KeyWallet.create();
        }
        LOG.infoExiting();

        for (int i = 0; i < testWallets.length; i++) {
            LOG.infoEntering("invoke", "helloWorld transfer");
            KeyWallet wallet = testWallets[i];
            Transaction t = TransactionBuilder.newBuilder()
                    .nid(txHandler.getNetworkId())
                    .from(wallet.getAddress())
                    .to(helloScore.getAddress())
                    .nonce(BigInteger.TEN)
                    .stepLimit(BigInteger.TEN)
                    .call("transfer")
                    .build();
            try {
                Bytes hash = txHandler.invoke(testWallets[0], t);
                assertEquals(0, i);
                assertSuccess(txHandler.getResult(hash));
            } catch (RpcError e) {
                assertNotEquals(0, i);
                LOG.info("Expected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
            } finally {
                LOG.infoExiting();
            }
        }
        LOG.infoExiting();
    }

    /*
     * If Governance SCORE has not been deployed, anyone can initially install Governance SCORE.
     */
    @Test
    public void deployGovScore() throws Exception {
        LOG.infoEntering("deployGovScore");
        LOG.infoEntering("install", "new governance");
        KeyWallet govOwner = KeyWallet.create();
        try {
            RpcObject params = new RpcObject.Builder()
                    .put("name", new RpcValue("HelloWorld"))
                    .put("value", new RpcValue(BigInteger.ONE))
                    .build();
            txHandler.deploy(govOwner, GovScore.INSTALL_PATH, Constants.GOV_ADDRESS, params, Constants.DEFAULT_STEPS);
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }

        // check install result
        Score govScore = new Score(txHandler, Constants.GOV_ADDRESS);
        boolean updated = govScore.call("updated", null).asBoolean();
        assertFalse(updated);

        // check failure when update with invalid wallet
        LOG.infoEntering("update", "with invalid owner");
        try {
            txHandler.deploy(KeyWallet.create(), GovScore.UPDATE_PATH, Constants.GOV_ADDRESS, null, Constants.DEFAULT_STEPS);
            fail();
        } catch (TransactionFailureException e) {
            LOG.info("Expected exception: code=" + e.getCode() + " msg=" + e.getMessage());
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }
        updated = govScore.call("updated", null).asBoolean();
        assertFalse(updated);

        // check success when update with owner
        LOG.infoEntering("update", "with owner");
        try {
            txHandler.deploy(govOwner, GovScore.UPDATE_PATH, Constants.GOV_ADDRESS, null, Constants.DEFAULT_STEPS);
            // check updated result
            updated = govScore.call("updated", null).asBoolean();
            assertTrue(updated);
        } catch (Exception e) {
            fail(e);
        } finally {
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }
}
