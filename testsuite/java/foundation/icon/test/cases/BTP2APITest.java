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

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.BTPNetworkInfo;
import foundation.icon.icx.data.BTPNetworkTypeInfo;
import foundation.icon.icx.data.Base64;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.test.common.BTPBlockHeader;
import foundation.icon.test.common.Codec;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.ChainScore;
import foundation.icon.test.score.GovScore;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.MethodOrderer;
import org.junit.jupiter.api.Order;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.TestMethodOrder;

import java.io.IOException;
import java.math.BigInteger;
import java.security.SecureRandom;
import java.util.List;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertArrayEquals;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assumptions.assumeTrue;

@Tag(Constants.TAG_PY_GOV)
@Tag(Constants.TAG_JAVA_GOV)
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
public class BTP2APITest extends TestBase {
    private static final String DSA_SECP256K1 = "ecdsa/secp256k1";
    private static final String NT_ETH = "eth";
    private static final String NT_ICON = "icon";
    private static final String[] NT_NAMES = {NT_ETH, NT_ICON};

    private static TransactionHandler txHandler;
    private static IconService iconService;
    private static ChainScore chainScore;
    private static GovScore govScore;
    private static Env.Node node = Env.nodes[0];
    private static SecureRandom secureRandom;

    private byte[] getRandomBytes(int size) {
        byte[] bytes = new byte[size];
        secureRandom.nextBytes(bytes);
        bytes[0] = 0; // make positive
        return bytes;
    }

    @BeforeAll
    static void init() throws Exception {
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        secureRandom = new SecureRandom();
        chainScore = new ChainScore(txHandler);
        govScore = new GovScore(txHandler);

        assumeTrue(checkRevision());
    }

    private static Boolean checkRevision() throws Exception {
        int revision = chainScore.getRevision();
        if (revision < 9) {
            LOG.info("Ignore this test at revision : " + revision);
            return false;
        }
        return true;
    }

    private byte[] ntPubKeyFromWallet(KeyWallet wallet) {
        var address = wallet.getAddress();
        var body = address.getBody();
        var prefix = address.getPrefix();
        byte[] raw = new byte[body.length + 1];
        raw[0] = (byte) prefix.ordinal();
        System.arraycopy(body, 0, raw, 1, body.length);
        return raw;
    }

    @Test
    @Order(100)
    public void managePublicKey() throws Exception {
        LOG.infoEntering("Public key management");
        KeyWallet caller1 = KeyWallet.create();
        KeyWallet caller2 = KeyWallet.create();

        class Case {
            final KeyWallet caller;
            final String name;
            final boolean success;
            final String title;
            final byte[] pubKey;

            public Case(KeyWallet caller, String name, byte[] pubKey, boolean success, String title) {
                this.caller = caller;
                this.name = name;
                this.pubKey = pubKey;
                this.success = success;
                this.title = title;
            }
        }

        byte[] pubKeyEmpty = new byte[0];

        var cases = new Case[]{
                new Case(caller1, NT_ICON, caller1.getPublicKey().toByteArray(), false, "Invalid name"),
                new Case(caller1, DSA_SECP256K1, "a023bd9e".getBytes(), false, "Invalid public key"),
                new Case(caller1, DSA_SECP256K1, caller1.getPublicKey().toByteArray(), true, "Set public key"),
//                new Case(caller2, DSA_SECP256K1, caller1.getPublicKey().toByteArray(), false, "Set public key with already exist"),
                new Case(caller1, DSA_SECP256K1, caller2.getPublicKey().toByteArray(), true, "Modify public key"),
                new Case(caller1, NT_ICON, pubKeyEmpty, false, "Delete with Invalid name"),
                new Case(caller1, DSA_SECP256K1, pubKeyEmpty, true, "Delete public key"),
                new Case(caller1, DSA_SECP256K1, pubKeyEmpty, true, "Delete empty public key"),
        };

        for (Case c : cases) {
            LOG.infoEntering(c.title);
            TransactionResult result;
            result = chainScore.setBTPPublicKey(caller1, c.name, c.pubKey);
            if (c.success) {
                assertSuccess(result);
                byte[] retPubKey = chainScore.getBTPPublicKey(caller1.getAddress(), c.name);
                if (c.pubKey.equals(pubKeyEmpty)) {
                    assertArrayEquals(null, retPubKey);
                } else {
                    assertArrayEquals(c.pubKey, retPubKey);
                }
            } else {
                assertFailure(result);
            }
            LOG.infoExiting();
        }
        LOG.infoExiting();
    }

    @Test
    @Order(101)
    public void manageBTPNetwork() throws Exception {
        LOG.infoEntering("BTPNetwork management");
        KeyWallet wallet = node.wallet;
        TransactionResult result;

        LOG.infoEntering("Try to open BTP network without publicKey");
        result = govScore.openBTPNetwork(NT_ETH, "ethereum", wallet.getAddress());
        assertFailure(result);
        LOG.infoExiting();

        LOG.infoEntering("Set public keys");
        setNodePublicKeys();
        LOG.infoExiting();

        class Case {
            final boolean open;
            final String ntName;
            final String dsa;
            final String name;
            final Address owner;
            BigInteger id;

            public Case(boolean open, String ntName, String dsa, String name, KeyWallet owner) {
                this.open = open;
                this.ntName = ntName;
                this.dsa = dsa;
                this.name = name;
                this.owner = owner.getAddress();
            }

            public void setId(BigInteger id) {
                this.id = id;
            }

        }

        var cases = new Case[]{
                new Case(true, NT_ETH, DSA_SECP256K1, "ethereum", wallet),
                new Case(false, NT_ETH, DSA_SECP256K1, "ethereum", wallet),
                new Case(true, NT_ETH, DSA_SECP256K1, "bsc", wallet),
                new Case(true, NT_ICON, DSA_SECP256K1, "ICON", wallet),
                new Case(true, NT_ETH, DSA_SECP256K1, "ethereum", wallet),
                new Case(false, NT_ETH, DSA_SECP256K1, "bsc", wallet),
                new Case(false, NT_ICON, DSA_SECP256K1, "ICON", wallet),
                new Case(false, NT_ETH, DSA_SECP256K1, "ethereum", wallet),
        };

        for (int i = 0; i < cases.length; i++) {
            Case c = cases[i];
            LOG.infoEntering((c.open ? "open" : "close") + " network type=" + c.ntName + " name=" + c.name);
            if (c.open) {
                BigInteger nid = openBTPNetwork(c.ntName, c.name, c.owner);
                c.setId(nid);
            } else {
                for (int j = i - 1; j >= 0; j--) {
                    Case t = cases[j];
                    if (t.open && t.name.equals(c.name)) {
                        closeBTPNetwork(t.id);
                        c.setId(t.id);
                        break;
                    }
                }
            }
            LOG.info("network ID=" + c.id);
            LOG.infoExiting();
        }
    }

    @Test
    @Order(102)
    public void modifyPublicKey() throws Exception {
        LOG.infoEntering("Modify public key while BTP network is working");
        // check count of validator >= 4
        if (4 > Env.nodes.length) {
            LOG.infoExiting("Not enough validator < 4");
            return;
        }

        KeyWallet wallet = node.wallet;
        TransactionResult result;

        LOG.infoEntering("Open BTP Networks 'ethereum' and 'icon' for test");
        BigInteger nidEth = openBTPNetwork(NT_ETH, "ethereum", wallet.getAddress());
        BigInteger nidIcon = openBTPNetwork(NT_ICON, "icon", wallet.getAddress());
        var ntidEth = iconService.btpGetNetworkInfo(nidEth).execute().getNetworkTypeID();
        var ntidIcon = iconService.btpGetNetworkInfo(nidIcon).execute().getNetworkTypeID();
        BigInteger[] ntids = {ntidEth, ntidIcon};
        BigInteger[] nids = {nidEth, nidIcon};
        LOG.infoExiting();

        byte[] pubKeyEmpty = new byte[0];

        LOG.infoEntering("Modify public key : all network type and network changed");
        KeyWallet nWallet = KeyWallet.create();
        result = chainScore.setBTPPublicKey(wallet, DSA_SECP256K1, nWallet.getPublicKey().toByteArray());
        assertSuccess(result);
        BigInteger height = result.getBlockHeight();
        for (BigInteger ntid: ntids) {
            checkNetworkType(height, ntid);
        }
        for (BigInteger nid: nids) {
            checkNetwork(height, nid, true);
            checkHeader(height.add(BigInteger.ONE), nid);
        }
        LOG.infoExiting();

        LOG.infoEntering("Set public key with same one");
        result = chainScore.setBTPPublicKey(wallet, DSA_SECP256K1, nWallet.getPublicKey().toByteArray());
        assertSuccess(result);
        height = result.getBlockHeight();
        for (BigInteger ntid: ntids) {
            checkNetworkTypeNotChanged(height, ntid);
        }
        for (BigInteger nid: nids) {
            checkNetworkNotChanged(height, nid);
        }
        LOG.infoExiting();

        LOG.infoEntering("Delete public key");
        result = chainScore.setBTPPublicKey(wallet, DSA_SECP256K1, pubKeyEmpty);
        assertSuccess(result);
        height = result.getBlockHeight();
        for (BigInteger ntid: ntids) {
            checkNetworkType(height, ntid);
        }
        for (BigInteger nid: nids) {
            checkNetwork(height, nid, true);
        }
        LOG.infoExiting();

        LOG.infoEntering("Modify public key with new one");
        nWallet = KeyWallet.create();
        result = chainScore.setBTPPublicKey(wallet, DSA_SECP256K1, nWallet.getPublicKey().toByteArray());
        assertSuccess(result);
        height = result.getBlockHeight();
        for (BigInteger ntid: ntids) {
            checkNetworkType(height, ntid);
        }
        for (BigInteger nid: nids) {
            checkNetwork(height, nid, true);
            checkHeader(height.add(BigInteger.ONE), nid);
        }
        LOG.infoExiting();

        LOG.infoEntering("Change public key for network type that has no open network");
        LOG.info("Close network 'ethereum'");
        closeBTPNetwork(nidEth);
        LOG.info("Modify public key");
        nWallet = KeyWallet.create();
        result = chainScore.setBTPPublicKey(wallet, DSA_SECP256K1, nWallet.getPublicKey().toByteArray());
        assertSuccess(result);
        height = result.getBlockHeight();
        LOG.info("Network type 'eth' not changed");
        checkNetworkTypeNotChanged(height, ntidEth);
        checkNetworkNotChanged(height, nidEth);
        LOG.info("Network type 'icon' changed");
        checkNetworkTypeNotChanged(height, ntidIcon);
        checkNetworkNotChanged(height, nidIcon);
        LOG.infoExiting();

        LOG.infoEntering("Reset public keys");
        resetNodePublicKeys();
        LOG.infoExiting();

        LOG.infoExiting();
    }
    @Test
    @Order(103)
    public void sendBTPMessage() throws Exception {
        LOG.infoEntering("Send BTP message");

        KeyWallet wallet = node.wallet;
        TransactionResult result;

        LOG.infoEntering("Open BTP Networks for test");
        BigInteger nid = openBTPNetwork(NT_ICON, "icon", wallet.getAddress());
        var ntid = iconService.btpGetNetworkInfo(nid).execute().getNetworkTypeID();
        LOG.infoExiting();

        var firstSN = 0;
        var msgCount = 1;

        LOG.infoEntering("Send first BTP message");
        byte[] msg = getRandomBytes(10);
        byte[][] firstMsgs = {msg};
        result = chainScore.sendBTPMessage(wallet, nid, msg);
        var height = result.getBlockHeight();
        checkNetworkTypeNotChanged(height, ntid);
        checkNetwork(height, nid, msgCount);
        height = height.add(BigInteger.ONE);
        checkHeader(height, nid, firstSN, msgCount);
        firstSN = firstSN + msgCount;
        checkMessage(height, nid, firstMsgs);
        LOG.infoExiting();

        LOG.infoEntering("Send BTP message again");
        msg = getRandomBytes(20);
        byte[][] secondMsgs = {msg};
        result = chainScore.sendBTPMessage(wallet, nid, msg);
        height = result.getBlockHeight();
        checkNetworkTypeNotChanged(height, ntid);
        checkNetwork(height, nid, msgCount);
        height = height.add(BigInteger.ONE);
        checkHeader(height, nid, firstSN, msgCount);
        checkMessage(height, nid, secondMsgs);
        LOG.infoExiting();

        LOG.infoExiting();
    }

    private void setNodePublicKeys() throws IOException, ResultTimeoutException {
        for (int i = 0; i < Env.nodes.length; i++) {
            KeyWallet w = Env.nodes[i].wallet;
            Bytes pubKey = w.getPublicKey();
            LOG.info(w.getAddress() + " : " + pubKey);
            TransactionResult result;
            result = chainScore.setBTPPublicKey(w, DSA_SECP256K1, pubKey.toByteArray());
            assertSuccess(result);
        }
    }

    private void resetNodePublicKeys() throws IOException, ResultTimeoutException {
        byte[] pubKeyEmpty = new byte[0];
        for (int i = 0; i < Env.nodes.length; i++) {
            KeyWallet w = Env.nodes[i].wallet;
            TransactionResult result;
            // clear public key of network type
            for (String name: NT_NAMES) {
                result = chainScore.setBTPPublicKey(w, name, pubKeyEmpty);
                assertSuccess(result);
            }
            // set public key with dsa
            result = chainScore.setBTPPublicKey(w, DSA_SECP256K1, w.getPublicKey().toByteArray());
            assertSuccess(result);
        }
    }

    private BigInteger openBTPNetwork(String ntName, String name, Address owner) throws Exception {
        boolean newNT = BigInteger.ZERO.equals(chainScore.getBTPNetworkTypeID(ntName));
        TransactionResult result;
        result = govScore.openBTPNetwork(ntName, name, owner);
        assertSuccess(result);
        BigInteger ntid = chainScore.getBTPNetworkTypeID(ntName);
        assertEquals(ntid.compareTo(BigInteger.ZERO), 1);

        BigInteger height = result.getBlockHeight();
        BigInteger nid = BigInteger.ZERO;
        boolean matchEvent = false;
        for (TransactionResult.EventLog e : result.getEventLogs()) {
            List<RpcItem> indexed = e.getIndexed();
            if (indexed.get(0).asString().equals("BTPNetworkOpened(int,int)")) {
                assertEquals(ntid, indexed.get(1).asInteger());
                nid = indexed.get(2).asInteger();
                matchEvent = true;
            }
        }
        assertTrue(matchEvent);

        if (newNT) {
            checkNetworkType(ntid, ntName, nid);
        } else {
            checkNetworkType(height, ntid, true, nid);
        }
        checkNetwork(nid, name, ntid, height);
        checkHeader(height.add(BigInteger.ONE), nid);
        return nid;
    }

    private void closeBTPNetwork(BigInteger id) throws Exception {
        TransactionResult result;
        result = govScore.closeBTPNetwork(id);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());

        BTPNetworkInfo nInfo = iconService.btpGetNetworkInfo(id).execute();
        var ntid = nInfo.getNetworkTypeID();
        boolean matchEvent = false;
        for (TransactionResult.EventLog e : result.getEventLogs()) {
            List<RpcItem> indexed = e.getIndexed();
            if (indexed.get(0).asString().equals("BTPNetworkClosed(int,int)")) {
                assertEquals(ntid, indexed.get(1).asInteger());
                assertEquals(id, indexed.get(2).asInteger());
                matchEvent = true;
            }
        }
        assertTrue(matchEvent);
        var height = result.getBlockHeight();
        checkNetwork(height, id, false);
        checkNetworkType(height, ntid, false, id);
    }

    // for openBTPNetwork - network type was activated
    private void checkNetworkType(BigInteger ntid, String name, BigInteger nid) throws Exception {
        BTPNetworkTypeInfo nInfo = iconService.btpGetNetworkTypeInfo(ntid).execute();
        assertEquals(name, nInfo.getNetworkTypeName());
        List<BigInteger> nIds = nInfo.getOpenNetworkIDs();
        assertEquals(1, nIds.size());
        assertEquals(nid, nIds.get(0));
    }

    // for open/closeBTPNetwork
    private void checkNetworkType(BigInteger height, BigInteger ntid, boolean open, BigInteger nid) throws Exception {
        BTPNetworkTypeInfo oInfo = iconService.btpGetNetworkTypeInfo(height, ntid).execute();
        BTPNetworkTypeInfo nInfo = iconService.btpGetNetworkTypeInfo(ntid).execute();
        List<BigInteger> oIds = oInfo.getOpenNetworkIDs();
        if (open) {
            oIds.add(nid);
        } else {
            oIds.remove(nid);
        }
        List<BigInteger> nIds = nInfo.getOpenNetworkIDs();
        assertEquals(oIds, nIds);
        if (nIds.size() == 0) {
            LOG.info("Check inactive network type");
            assertNull(nInfo.getNextProofContext());
        } else {
            assertNotNull(nInfo.getNextProofContext());
        }
    }

    // for public key modification
    private void checkNetworkType(BigInteger height, BigInteger ntid) throws Exception {
        BTPNetworkTypeInfo oInfo = iconService.btpGetNetworkTypeInfo(height, ntid).execute();
        BTPNetworkTypeInfo nInfo = iconService.btpGetNetworkTypeInfo(ntid).execute();
        assertEquals(oInfo.getOpenNetworkIDs(), nInfo.getOpenNetworkIDs());
        assertNotEquals(oInfo.getNextProofContext(), nInfo.getNextProofContext());
    }

    private void checkNetworkTypeNotChanged(BigInteger height, BigInteger ntid) throws Exception {
        BTPNetworkTypeInfo oInfo = iconService.btpGetNetworkTypeInfo(height, ntid).execute();
        BTPNetworkTypeInfo nInfo = iconService.btpGetNetworkTypeInfo(height.add(BigInteger.ONE), ntid).execute();
        assertEquals(oInfo, nInfo);
    }

    // for openBTPNetwork
    private void checkNetwork(BigInteger nid, String name, BigInteger ntid, BigInteger startHeight) throws Exception {
        BTPNetworkInfo nInfo = iconService.btpGetNetworkInfo(nid).execute();
        assertEquals(nid, nInfo.getNetworkID());
        assertEquals(name, nInfo.getNetworkName());
        assertEquals(ntid, nInfo.getNetworkTypeID());
        assertEquals(startHeight, nInfo.getStartHeight());
        assertEquals(BigInteger.ONE, nInfo.getOpen());
        assertNull(nInfo.getPrevNSHash());
    }

    // for closeBTPNetwork and public key modification
    private void checkNetwork(BigInteger height, BigInteger nid, boolean open) throws Exception {
        BTPNetworkInfo oInfo = iconService.btpGetNetworkInfo(height, nid).execute();
        BTPNetworkInfo nInfo = iconService.btpGetNetworkInfo(nid).execute();
        if (open) {
            assertEquals(BigInteger.ONE, nInfo.getOpen());
            assertEquals(oInfo.getLastNSHash(), nInfo.getPrevNSHash());
        } else {
            assertEquals(BigInteger.ZERO, nInfo.getOpen());
            assertEquals(oInfo.getPrevNSHash(), nInfo.getPrevNSHash());
            assertEquals(oInfo.getLastNSHash(), nInfo.getLastNSHash());
        }
    }

    // for sendBTPMessage
    private void checkNetwork(BigInteger height, BigInteger nid, int msgCount) throws Exception {
        BTPNetworkInfo oInfo = iconService.btpGetNetworkInfo(height, nid).execute();
        BTPNetworkInfo nInfo = iconService.btpGetNetworkInfo(nid).execute();
        assertEquals(BigInteger.ONE, nInfo.getOpen());
        assertEquals(oInfo.getLastNSHash(), nInfo.getPrevNSHash());
        assertEquals(oInfo.getNextMessageSN().add(BigInteger.valueOf(msgCount)), nInfo.getNextMessageSN());
    }

    private void checkNetworkNotChanged(BigInteger height, BigInteger nid) throws Exception {
        BTPNetworkInfo oInfo = iconService.btpGetNetworkInfo(height, nid).execute();
        BTPNetworkInfo nInfo = iconService.btpGetNetworkInfo(height.add(BigInteger.ONE), nid).execute();
        assertEquals(oInfo, nInfo);
    }

    // for openBTPHeader and public key modification
    private void checkHeader(BigInteger height, BigInteger nid) throws Exception {
        Base64 blkB64 = iconService.btpGetHeader(height, nid).execute();
        var header = new BTPBlockHeader(blkB64.decode(), Codec.rlp);
        assertTrue(header.getNextProofContextChanged());
        assertNotNull(header.getNextProofContext());
        assertEquals(0, header.getMessageCount());
    }

    // for sendBTPMessage
    private void checkHeader(BigInteger height, BigInteger nid, int firstMsgSN, int msgCount) throws Exception {
        Base64 blkB64 = iconService.btpGetHeader(height, nid).execute();
        var header = new BTPBlockHeader(blkB64.decode(), Codec.rlp);
        assertEquals(firstMsgSN, header.getFirstMessageSN());
        assertEquals(msgCount, header.getMessageCount());
        assertFalse(header.getNextProofContextChanged());
        assertNull(header.getNextProofContext());
    }

    private void checkMessage(BigInteger height, BigInteger nid, byte[][] msgs) throws Exception {
        Base64[] msgsB64 = iconService.btpGetMessages(height, nid).execute();
        assertEquals(msgs.length, msgsB64.length);
        for (int i = 0; i < msgs.length; i++) {
            assertArrayEquals(msgs[i], msgsB64[i].decode());
        }
    }
}
