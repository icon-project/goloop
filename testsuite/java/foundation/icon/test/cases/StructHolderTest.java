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
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.RpcItems;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.Score;
import foundation.icon.test.score.StructHolderScore;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;

@Tag(Constants.TAG_JAVA_SCORE)
public class StructHolderTest extends TestBase {
    private static TransactionHandler txHandler;
    private static KeyWallet ownerWallet;
    private static Score testScore;

    @BeforeAll
    static void setup() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        ownerWallet = KeyWallet.create();
        BigInteger amount = ICX.multiply(BigInteger.valueOf(100));
        txHandler.transfer(ownerWallet.getAddress(), amount);
        ensureIcxBalance(txHandler, ownerWallet.getAddress(), BigInteger.ZERO, amount);
    }

    @AfterAll
    static void shutdown() throws Exception {
        txHandler.refundAll(ownerWallet);
    }

    private StructHolderScore deployScore() throws Exception {
        if (testScore == null) {
            testScore = StructHolderScore.mustDeploy(txHandler, ownerWallet);
        }
        return (StructHolderScore) testScore;
    }

    @Test
    void testStruct() throws Exception {
        var s = deployScore();
        LOG.infoEntering("run");
        RpcObject complexStruct = new RpcObject.Builder()
                .put("string", new RpcValue("stringInComplexStruct"))
                .put("integer", new RpcValue(BigInteger.valueOf(100)))
                .put("address", new RpcValue(new Address("cx10776ee37f5b45bfaea8cff1d8232fbb6122ec32")))
                .put("bool", new RpcValue(true))
                .put("bytes", new RpcValue(new Bytes("0xCAFEBABE")))
                .put("simpleStruct", new RpcObject.Builder()
                        .put("string", new RpcValue("stringInSimpleStruct"))
                        .put("integer", new RpcValue(BigInteger.valueOf(200)))
                        .put("address", new RpcValue(new Address("cx20776ee37f5b45bfaea8cff1d8232fbb6122ec32")))
                        .put("bool", new RpcValue(false))
                        .put("bytes", new RpcValue(new Bytes("0xBABECAFE")))
                        .build()
                )
                .build();
        RpcObject params = new RpcObject.Builder()
                .put("complexStruct", complexStruct)
                .build();
        assertSuccess(s.setComplexStruct(ownerWallet, params));
        var res = s.call("getComplexStruct", null)
                .asObject();
        LOG.info("getComplexStruct() : " + res.toString());
        if (!RpcItems.equals(complexStruct, res)) {
            // show diff
            assertEquals(complexStruct.toString(), res.toString());
        }
        LOG.infoExiting();
    }

    @Test
    void invalidParams() throws Exception {
        var s = deployScore();
        LOG.infoEntering("run");
        RpcObject params = new RpcObject.Builder()
                .put("simpleStruct", new RpcObject.Builder()
                        .put("string", new RpcValue("Hello"))
                        .put("integer", new RpcValue(BigInteger.valueOf(100)))
                        .put("bool", new RpcValue(true))
                        .put("bytes", new RpcValue(new Bytes("0xCAFEBABE")))
                        .build())
                .build();
        assertFailure(s.setSimpleStruct(ownerWallet, params));
        LOG.infoExiting();
    }
}
