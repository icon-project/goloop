/*
 * Copyright 2022 ICON Foundation
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
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Base64;
import foundation.icon.icx.data.Block;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.*;
import foundation.icon.test.score.ChainScore;
import foundation.icon.test.score.EventGen;
import foundation.icon.test.score.GovScore;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;

@Tag(Constants.TAG_JAVA_SCORE)
public class NodeWalletTest extends TestBase {
    private static TransactionHandler txHandler;
    private static IconService iconService;
    private static ChainScore chainScore;
    private static Env.Node node = Env.nodes[0];

    @BeforeAll
    static void init() {
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);

        chainScore = new ChainScore(txHandler);
    }

    @Test
    public void nodeWallet() throws Exception {
        LOG.infoEntering("nodeWallet");
        boolean fail = true;
        KeyWallet wallet = node.wallet;
        LOG.info("Get node.wallet for : " + wallet.getAddress());

        RpcItem item = chainScore.call("getValidators", null);
        RpcArray rpcArray = item.asArray();
        for (int i = 0; i < rpcArray.size(); i++) {
            if (rpcArray.get(i).asAddress().equals(wallet.getAddress())) {
                fail = false;
                LOG.info("node.wallet.getAddress() == validator(" + i + ")");
                break;
            }
        }
        assertEquals(fail, false);
        LOG.infoExiting();
    }
}
