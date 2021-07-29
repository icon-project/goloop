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
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.ChainScore;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.fail;

@Tag(Constants.TAG_PY_SCORE)
public class ResultTest extends TestBase {
    private static final String SCORE_RESULT_GEN_PATH = Score.getFilePath("result_gen");
    private static KeyWallet ownerWallet;
    private static Score score1, score2;
    private static Score chainSCORE;

    @BeforeAll
    public static void setUp() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        TransactionHandler txHandler = new TransactionHandler(iconService, chain);

        ownerWallet = KeyWallet.create();
        score1 = txHandler.deploy(ownerWallet, SCORE_RESULT_GEN_PATH, null);
        score2 = txHandler.deploy(ownerWallet, SCORE_RESULT_GEN_PATH, null);
        chainSCORE = new ChainScore(txHandler);
    }

    final static int CODE_REVERTED = 32;
    final static int CODE_LIMIT_REV5 = 99;
    final static int CODE_LIMIT_REV6 = 999;

    @Test
    void checkFailureCodeForRevert() throws Exception {
        LOG.infoEntering("checkFailureCodeForRevert");
        BigInteger[] cases = {
                BigInteger.ZERO,
                BigInteger.valueOf(CODE_LIMIT_REV5 - CODE_REVERTED + 1),
                BigInteger.valueOf(CODE_LIMIT_REV6 - CODE_REVERTED + 1),
        };
        BigInteger[] expect;
        RpcItem rev = chainSCORE.call("getRevision", null);
        LOG.info("Revision: " + rev.asString());
        if (rev.asInteger().intValue() < 6) {
            expect = new BigInteger[]{
                    BigInteger.valueOf(CODE_REVERTED),
                    BigInteger.valueOf(CODE_LIMIT_REV5),
                    BigInteger.valueOf(CODE_LIMIT_REV5),
            };
        } else {
            expect = new BigInteger[]{
                    BigInteger.valueOf(CODE_REVERTED),
                    BigInteger.valueOf(CODE_LIMIT_REV5 + 1),
                    BigInteger.valueOf(CODE_LIMIT_REV6),
            };
        }

        Bytes[] txs = new Bytes[cases.length];
        Bytes[] icTxs = new Bytes[cases.length];
        Bytes[] iccTxs = new Bytes[cases.length];
        for (int i = 0; i < cases.length; i++) {
            LOG.info("send tx normal case" + i);
            RpcObject params = new RpcObject.Builder()
                    .put("index", new RpcValue(cases[i]))
                    .build();
            txs[i] = score1.invoke(ownerWallet, "callRevertWithIndex", params);

            LOG.info("query case" + i);
            try {
                RpcItem qr = score1.call("queryRevertWithIndex", params);
                fail();
            } catch (RpcError e) {
                assertEquals(-30000 - expect[i].intValue(), e.getCode());
            }

            LOG.info("send tx inter-call case" + i);
            params = new RpcObject.Builder()
                    .put("addr", new RpcValue(score2.getAddress()))
                    .put("index", new RpcValue(cases[i]))
                    .build();
            icTxs[i] = score1.invoke(ownerWallet, "interCallRevertWithIndex", params);

            LOG.info("send tx inter-call catch case" + i);
            iccTxs[i] = score1.invoke(ownerWallet, "interCallCatchRevertWithIndex", params);
        }

        for (int i = 0; i < cases.length; i++) {
            LOG.info("check result for normal case" + i);
            TransactionResult result = score1.getResult(txs[i]);
            assertEquals(result.getStatus(), Constants.STATUS_FAILURE);
            assertEquals(expect[i], result.getFailure().getCode());

            LOG.info("check result for inter-call case" + i);
            result = score1.getResult(icTxs[i]);
            assertEquals(result.getStatus(), Constants.STATUS_FAILURE);
            assertEquals(expect[i], result.getFailure().getCode());

            LOG.info("check result for inter-call catch case" + i);
            result = score1.getResult(iccTxs[i]);
            assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);
            boolean checked = false;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                if (el.getIndexed().get(0).asString().equals("RevertCatch(int)")) {
                    assertEquals(expect[i], el.getData().get(0).asInteger());
                    checked = true;
                }
            }
            assertTrue(checked);
        }

        LOG.infoExiting();
    }

    @Test
    void checkExternalReturnValue() throws Exception {
        LOG.infoEntering("checkExternalReturnValue");
        String[] values = {
                "HelloWorld", "한글", ""
        };

        Bytes[] txs = new Bytes[values.length];
        for (int i = 0; i < values.length; i++) {
            RpcObject params = new RpcObject.Builder()
                    .put("addr", new RpcValue(score2.getAddress()))
                    .put("value", new RpcValue(values[i]))
                    .build();
            txs[i] = score1.invoke(ownerWallet, "interCallReturnStr", params);
        }

        for (int i = 0; i < values.length; i++) {
            TransactionResult result = score1.getResult(txs[i]);
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            int checked = 0;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                if (el.getIndexed().get(0).asString().equals("ReturnedStr(str)")) {
                    assertEquals(values[i], el.getData().get(0).asString());
                    checked += 1;
                }
            }
            assertEquals(1, checked);
        }
        LOG.infoExiting();
    }

    @Test
    void checkBytesReturns() throws Exception {
        LOG.infoEntering("checkBytesReturns");
        String index = "1234";
        String expected = "Bytes: " + index;
        RpcObject params = new RpcObject.Builder()
                .put("addr", new RpcValue(score2.getAddress()))
                .put("index", new RpcValue(index))
                .build();
        assertSuccess(score1.invokeAndWaitResult(ownerWallet, "set_bytes_value", params));

        RpcObject result = score1.call("get_bytes_value", params).asObject();
        for (String key : result.keySet()) {
            if ("index".equals(key)) {
                assertEquals(index, result.getItem(key).asString());
            } else if ("address".equals(key)) {
                assertEquals(score2.getAddress(), result.getItem(key).asAddress());
            } else if ("bytes".equals(key)) {
                assertEquals(new Bytes(expected.getBytes()).toString(),
                        result.getItem(key).asString());
            } else {
                fail("Unexpected key: " + key + ", value=" + result.getItem(key));
            }
        }
        LOG.infoExiting();
    }

    @Test
    void checkInvalidKeyReturns() throws Exception {
        LOG.infoEntering("checkInvalidKeyReturns");
        try {
            score1.call("get_invalid_key", null);
        } catch (RpcError e) {
            assertEquals(-30001, e.getCode());
            LOG.info("Expected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
        }
        LOG.infoExiting();
    }
}
