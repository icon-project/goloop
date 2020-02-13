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
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.HelloWorld;
import foundation.icon.test.score.Score;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.fail;

@Tag(Constants.TAG_PY_SCORE)
public class SendAndWaitTest {
    private static Env.Channel channel;
    private static KeyWallet wallet;
    private static IconService iconService;
    private static Score testScore;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        channel = node.channels[0];
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        TransactionHandler txHandler = new TransactionHandler(iconService, channel.chain);

        wallet = KeyWallet.create();
        testScore = HelloWorld.install(txHandler, wallet);
    }

    private void resetTimeout(int timeout) {
        OkHttpClient httpClient;
        if (timeout > 0) {
            httpClient = new OkHttpClient.Builder()
                    .addInterceptor(chain -> {
                        Request request = chain.request().newBuilder()
                                .addHeader("Icon-Options", "timeout=" + timeout)
                                .build();
                        return chain.proceed(request);
                    }).build();
        } else {
            httpClient = new OkHttpClient.Builder().build();
        }
        iconService.setProvider(new HttpProvider(httpClient, channel.getAPIUrl(Env.testApiVer)));
    }

    private TransactionResult waitTransactionResult(Bytes txHash) throws IOException {
        while (true) {
            try {
                return testScore.waitResult(txHash);
            } catch (RpcError e) {
                LOG.info("[RpcError] code=" + e.getCode() + ", msg=" + e.getMessage() + ", data=" + e.getData());
                if (e.getCode() == -31006 || e.getCode() == -31007) {
                    continue;
                }
                throw e;
            }
        }
    }

    @Test
    public void testSendTxAndWait() throws Exception {
        int[] timeouts = {500, 1000, 2000, 3000, 10000, 0};
        for (int timeout : timeouts) {
            LOG.infoEntering("timeout " + timeout);
            resetTimeout(timeout);
            long t0 = System.currentTimeMillis();
            try {
                RpcObject params = new RpcObject.Builder()
                        .put("name", new RpcValue("Alice"))
                        .build();
                TransactionResult result = testScore.invokeAndWait(wallet, "helloWithName", params,
                        BigInteger.ZERO, BigInteger.valueOf(1000));
                assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            } catch (RpcError e) {
                LOG.info("[RpcError] code=" + e.getCode() + ", msg=" + e.getMessage() + ", data=" + e.getData());
                if (e.getCode() == -31006 || e.getCode() == -31007) {
                    LOG.infoEntering("call waitTransactionResult");
                    TransactionResult result = waitTransactionResult(e.getData());
                    assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
                    LOG.infoExiting();
                } else {
                    fail("Unexpected RpcError");
                }
            }
            long t1 = System.currentTimeMillis();
            LOG.info("elapsed=" + (t1 - t0));
            LOG.infoExiting();
        }
    }
}
