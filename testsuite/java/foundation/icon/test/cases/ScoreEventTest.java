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
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.EventGen;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

@Tag(Constants.TAG_PY_SCORE)
class ScoreEventTest {
    private static TransactionHandler txHandler;
    private static KeyWallet ownerWallet;
    private static EventGen testScore;

    @BeforeAll
    static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        initScore();
    }

    private static void initScore() throws Exception {
        ownerWallet = KeyWallet.create();
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        testScore = EventGen.install(txHandler, ownerWallet, Constants.CONTENT_TYPE_PYTHON, params);
    }

    @Test
    void generateNullByIndex() throws Exception {
        LOG.infoEntering("generateNullByIndex");
        final int NUM = 5;

        String[] expects = {"0x1", "0x1", "test", "hx0000000000000000000000000000000000000000", "0x01"};

        Bytes[] ids = new Bytes[NUM];
        for (int i = 0; i < NUM; i++) {
            RpcObject params = new RpcObject.Builder()
                    .put("_idx", new RpcValue(BigInteger.valueOf(i)))
                    .build();
            ids[i] = testScore.invoke(ownerWallet, "generateNullByIndex", params);
        }

        String[] blooms = new String[NUM];
        for (int i = 0; i < NUM; i++) {
            TransactionResult result = testScore.getResult(ids[i]);
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());

            blooms[i] = result.getLogsBloom();

            boolean checked = false;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                String sig = el.getIndexed().get(0).asString();
                if (!"EventEx(bool,int,str,Address,bytes)".equals(sig)) {
                    continue;
                }
                for (int j = 0; j < NUM; j++) {
                    RpcItem v;
                    if (j < 3) {
                        v = el.getIndexed().get(j + 1);
                    } else {
                        v = el.getData().get(j - 3);
                    }
                    if (i == j) {
                        assertTrue(v.isNull());
                    } else {
                        assertEquals(expects[j], v.asString());
                    }
                }
                checked = true;
            }
            assertTrue(checked);
        }

        assertEquals(blooms[3], blooms[4]);
        BigInteger base = new BigInteger(blooms[3].substring(2), 16);
        for (int i = 0; i < 3; i++) {
            BigInteger bloom = new BigInteger(blooms[i].substring(2), 16);
            assertEquals(bloom.and(base), bloom);
        }

        LOG.infoExiting();
    }
}
