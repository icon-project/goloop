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
import foundation.icon.icx.data.ConfirmedTransaction ;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.*;
import foundation.icon.test.score.ChainScore;
import foundation.icon.test.score.EventGen;
import foundation.icon.test.score.GovScore;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.util.Arrays;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertArrayEquals;
import static org.junit.jupiter.api.Assertions.fail;

@Tag(Constants.TAG_PY_GOV)
@Tag(Constants.TAG_JAVA_GOV)
public class BTP2APITest extends TestBase {
    private static TransactionHandler txHandler;
    private static IconService iconService;
    private static Codec codec;
    private static ChainScore chainScore;
    private static GovScore govScore;
    private static KeyWallet owner;
    private static Env.Node node = Env.nodes[0];

    final int requiredRevision = 9;

    @BeforeAll
    static void init() {
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);

        var cname = chain.getProperty("codec", "rlp");
        if (cname.equals("rlp")) {
            codec = Codec.rlp;
        } else {
            codec = Codec.messagePack;
        }

        chainScore = new ChainScore(txHandler);
        govScore = new GovScore(txHandler);
    }

    private Boolean checkRevision(int requiredRevision) throws Exception {
        int revision = chainScore.getRevision();
        if (revision < requiredRevision) {
            LOG.info("Ignore this test at revision : " + revision);
            return false;
        }
        return true;
    }

    @Test
    public void setgetPublicKey() throws Exception {
        LOG.infoEntering("setgetPublicKey");
        if (this.checkRevision(requiredRevision) == false) {
            LOG.infoExiting();
            return;
        }
        KeyWallet caller = node.wallet;
        LOG.info("caller from node.wallet" + caller.getAddress());

        class Case {
            boolean add;
            boolean withDSA;
            String name;
            boolean success;
            String title;

            Case(boolean add, boolean withDSA, String name, boolean success, String title) {
                this.add = add;
                this.withDSA = withDSA;
                this.name = name;
                this.success = success;
                this.title = title;
            }
        }

        var cases = new Case[] {
                new Case(true, false, "eth", true, "Set with Network type `eth`"),
                new Case(true, false, "icon", true, "Set with Network type `icon`"),
                new Case(true, true, "ecdsa/secp256k1", true, "Set with DSA"),
                new Case(true, false, "InvalidName", false, "Set with Invalid name"),
                new Case(false, false, "InvalidName", false, "Delete with Invalid name"),
                new Case(false, true, "ecdsa/secp256k1", true, "Delete with DSA"),
                new Case(false, true, "ecdsa/secp256k1", true, "Delete empty with DSA"),
                new Case(false, false, "eth", true, "Delete with Network type `eth`"),
                new Case(false, false, "eth", true, "Delete empty with Network type `eth`"),
        };

        byte[] pubKeyDSA = caller.getPublicKey().toByteArray();
        byte[] pubKeyNT = caller.getAddress().getBody();
        byte[] pubKeyEmpty = new byte[0];

        for (Case c : cases) {
            LOG.infoEntering(c.title);
            byte[] pubKey = c.add ? (c.withDSA ? pubKeyDSA : pubKeyNT) : pubKeyEmpty;
            TransactionResult result;
            result = chainScore.setBTPPublicKey(caller, c.name, pubKey);
            if (c.success) {
                byte[] retPubKey = chainScore.getBTPPublicKey(caller.getAddress(), c.name);
                if (c.add) {
                    assertArrayEquals(pubKey, retPubKey);
                } else {
                    assertArrayEquals(null, retPubKey);
                }
            }
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }
}
