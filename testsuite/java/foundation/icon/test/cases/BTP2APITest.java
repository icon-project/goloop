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
import foundation.icon.test.score.BTP2;
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

@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
public class BTP2APITest extends TestBase {
    private static final String DSA_SECP256K1 = "ecdsa/secp256k1";
    private static final String NT_ETH = "eth";
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

    @Tag(Constants.TAG_PY_GOV)
    @Tag(Constants.TAG_JAVA_GOV)
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
        var caller1PubKey = caller1.getPublicKey().toByteArray();
        var caller2PubKey = caller2.getPublicKey().toByteArray();

        var cases = new Case[]{
                new Case(caller1, NT_ETH, caller1.getPublicKey().toByteArray(), false, "Invalid name"),
                new Case(caller1, DSA_SECP256K1, "a023bd9e".getBytes(), false, "Invalid public key"),
                new Case(caller1, DSA_SECP256K1, caller1PubKey, true, "Set public key"),
                new Case(caller1, DSA_SECP256K1, caller1PubKey, true, "Set same public key again"),
                new Case(caller1, DSA_SECP256K1, caller2PubKey, true, "Modify public key"),
                new Case(caller1, DSA_SECP256K1, caller1PubKey, true, "Restore public key"),
                new Case(caller2, DSA_SECP256K1, caller1PubKey, false, "Set public key with already exist"),
                new Case(caller2, DSA_SECP256K1, caller2PubKey, true, "Set with deleted public key"),
                new Case(caller1, NT_ETH, pubKeyEmpty, false, "Delete with Invalid name"),
                new Case(caller1, DSA_SECP256K1, pubKeyEmpty, true, "Delete public key"),
                new Case(caller2, DSA_SECP256K1, pubKeyEmpty, true, "Delete public key"),
                new Case(caller1, DSA_SECP256K1, pubKeyEmpty, true, "Delete empty public key"),
        };

        for (Case c : cases) {
            LOG.infoEntering(c.title);
            TransactionResult result;
            result = chainScore.setBTPPublicKey(c.caller, c.name, c.pubKey);
            if (c.success) {
                assertSuccess(result);
                byte[] retPubKey = chainScore.getBTPPublicKey(c.caller.getAddress(), c.name);
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

    @Tag(Constants.TAG_PY_GOV)
    @Tag(Constants.TAG_JAVA_GOV)
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
                new Case(true, NT_ETH, DSA_SECP256K1, "ethereum", wallet),
                new Case(false, NT_ETH, DSA_SECP256K1, "bsc", wallet),
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

    @Tag(Constants.TAG_PY_GOV)
    @Tag(Constants.TAG_JAVA_GOV)
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

        LOG.infoEntering("Open BTP Networks 'ethereum' and 'bsc' for test");
        BigInteger nidEth = openBTPNetwork(NT_ETH, "ethereum", wallet.getAddress());
        BigInteger nidBSC = openBTPNetwork(NT_ETH, "bsc", wallet.getAddress());
        var ntidEth = iconService.getBTPNetworkInfo(nidEth).execute().getNetworkTypeID();
        var ntidBSC = iconService.getBTPNetworkInfo(nidBSC).execute().getNetworkTypeID();
        BigInteger[] ntids = {ntidEth, ntidBSC};
        BigInteger[] nids = {nidEth, nidBSC};
        LOG.infoExiting();

        byte[] pubKeyEmpty = new byte[0];

        LOG.infoEntering("Modify public key : all network type and network changed");
        KeyWallet nWallet = KeyWallet.create();
        result = chainScore.setBTPPublicKey(wallet, DSA_SECP256K1, nWallet.getPublicKey().toByteArray());
        assertSuccess(result);
        BigInteger height = result.getBlockHeight();
        for (BigInteger ntid : ntids) {
            checkNetworkType(height, ntid);
        }
        for (BigInteger nid : nids) {
            checkNetwork(height, nid, true);
            checkHeader(height.add(BigInteger.ONE), nid);
        }
        LOG.infoExiting();

        LOG.infoEntering("Set public key with same one");
        result = chainScore.setBTPPublicKey(wallet, DSA_SECP256K1, nWallet.getPublicKey().toByteArray());
        assertSuccess(result);
        height = result.getBlockHeight();
        for (BigInteger ntid : ntids) {
            checkNetworkTypeNotChanged(height, ntid);
        }
        for (BigInteger nid : nids) {
            checkNetworkNotChanged(height, nid);
        }
        LOG.infoExiting();

        LOG.infoEntering("Delete public key");
        result = chainScore.setBTPPublicKey(wallet, DSA_SECP256K1, pubKeyEmpty);
        assertSuccess(result);
        height = result.getBlockHeight();
        for (BigInteger ntid : ntids) {
            checkNetworkType(height, ntid);
        }
        for (BigInteger nid : nids) {
            checkNetwork(height, nid, true);
        }
        LOG.infoExiting();

        LOG.infoEntering("Modify public key with new one");
        nWallet = KeyWallet.create();
        result = chainScore.setBTPPublicKey(wallet, DSA_SECP256K1, nWallet.getPublicKey().toByteArray());
        assertSuccess(result);
        height = result.getBlockHeight();
        for (BigInteger ntid : ntids) {
            checkNetworkType(height, ntid);
        }
        for (BigInteger nid : nids) {
            checkNetwork(height, nid, true);
            checkHeader(height.add(BigInteger.ONE), nid);
        }
        LOG.infoExiting();

        LOG.infoEntering("Reset public keys");
        setNodePublicKeys();
        LOG.infoExiting();

        LOG.infoExiting();
    }

    @Tag(Constants.TAG_PY_GOV)
    @Tag(Constants.TAG_JAVA_GOV)
    @Test
    @Order(103)
    public void sendBTPMessage() throws Exception {
        LOG.infoEntering("Send BTP message");

        KeyWallet wallet = node.wallet;
        TransactionResult result;

        LOG.infoEntering("Open BTP Networks for test");
        BigInteger nid = openBTPNetwork(NT_ETH, "send_msg_test", wallet.getAddress());
        var ntid = iconService.getBTPNetworkInfo(nid).execute().getNetworkTypeID();
        LOG.infoExiting();

        var msgSN = 0;
        var msgCount = 1;

        LOG.infoEntering("Send first BTP message to " + nid);
        byte[] msg = getRandomBytes(10);
        byte[][] firstMsgs = {msg};
        result = chainScore.sendBTPMessage(wallet, nid, msg);
        checkEventLog(result.getEventLogs(), nid, msgSN);
        var height = result.getBlockHeight();
        checkNetworkTypeNotChanged(height, ntid);
        checkNetwork(height, nid, msgCount);
        height = height.add(BigInteger.ONE);
        checkHeader(height, nid, msgSN, msgCount);
        checkMessage(height, nid, firstMsgs);
        msgSN = msgSN + msgCount;
        LOG.infoExiting();

        LOG.infoEntering("Send BTP message again");
        msg = getRandomBytes(20);
        byte[][] secondMsgs = {msg};
        result = chainScore.sendBTPMessage(wallet, nid, msg);
        checkEventLog(result.getEventLogs(), nid, msgSN);
        height = result.getBlockHeight();
        checkNetworkTypeNotChanged(height, ntid);
        checkNetwork(height, nid, msgCount);
        height = height.add(BigInteger.ONE);
        checkHeader(height, nid, msgSN, msgCount);
        checkMessage(height, nid, secondMsgs);
        LOG.infoExiting();

        LOG.infoExiting();
    }

    @Tag(Constants.TAG_JAVA_GOV)
    @Test
    @Order(104)
    public void sendBTPMessageAndRevert() throws Exception {
        KeyWallet wallet = node.wallet;
        TransactionResult result;

        LOG.infoEntering("Deploy SCOREs for test");
        var bmc = txHandler.deploy(wallet, testcases.BTP2BMC.class, null);
        BTP2 testScore = BTP2.install(txHandler, wallet, bmc.getAddress());
        LOG.infoExiting();

        LOG.infoEntering("Open BTP Networks for test");
        BigInteger nid = openBTPNetwork(NT_ETH, "send_msg", bmc.getAddress());
        BigInteger nidRevert = openBTPNetwork(NT_ETH, "send_msg_revert", bmc.getAddress());
        LOG.infoExiting();

        LOG.infoEntering("Send BTP message and revert");
        byte[] msg = getRandomBytes(10);
        int msgCount = 3;
        result = testScore.sendAndRevert(wallet, nid, msg, BigInteger.valueOf(msgCount), nidRevert);
        var height = result.getBlockHeight();
        checkNetwork(height, nid, msgCount);
        checkNetworkNotChanged(height, nidRevert);

        height = height.add(BigInteger.ONE);
        checkHeader(height, nid, 0, msgCount);
        byte[][] msgs = new byte[msgCount][];
        for (int i = 0; i < msgCount; i++) {
            msgs[i] = msg;
        }
        checkMessage(height, nid, msgs);
        LOG.infoExiting();
    }

    @Tag(Constants.TAG_PY_GOV)
    @Tag(Constants.TAG_JAVA_GOV)
    @Test
    @Order(110)
    public void grantValidator() throws Exception {
        LOG.infoEntering("Grant validator without public keys");
        KeyWallet wallet = KeyWallet.create();
        TransactionResult result;
        result = govScore.grantValidator(wallet.getAddress());
        assertFailure(result);
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
        checkNetwork(nid, name, ntid, height, owner);
        checkHeader(height.add(BigInteger.ONE), nid);
        return nid;
    }

    private void closeBTPNetwork(BigInteger id) throws Exception {
        TransactionResult result;
        result = govScore.closeBTPNetwork(id);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());

        BTPNetworkInfo nInfo = iconService.getBTPNetworkInfo(id).execute();
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
        BTPNetworkTypeInfo nInfo = iconService.getBTPNetworkTypeInfo(ntid).execute();
        assertEquals(name, nInfo.getNetworkTypeName());
        List<BigInteger> nIds = nInfo.getOpenNetworkIDs();
        assertEquals(1, nIds.size());
        assertEquals(nid, nIds.get(0));
    }

    // for open/closeBTPNetwork
    private void checkNetworkType(BigInteger height, BigInteger ntid, boolean open, BigInteger nid) throws Exception {
        BTPNetworkTypeInfo oInfo = iconService.getBTPNetworkTypeInfo(ntid, height).execute();
        BTPNetworkTypeInfo nInfo = iconService.getBTPNetworkTypeInfo(ntid).execute();
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
        BTPNetworkTypeInfo oInfo = iconService.getBTPNetworkTypeInfo(ntid, height).execute();
        BTPNetworkTypeInfo nInfo = iconService.getBTPNetworkTypeInfo(ntid).execute();
        assertEquals(oInfo.getOpenNetworkIDs(), nInfo.getOpenNetworkIDs());
        assertNotEquals(oInfo.getNextProofContext(), nInfo.getNextProofContext());
    }

    private void checkNetworkTypeNotChanged(BigInteger height, BigInteger ntid) throws Exception {
        BTPNetworkTypeInfo oInfo = iconService.getBTPNetworkTypeInfo(ntid, height).execute();
        BTPNetworkTypeInfo nInfo = iconService.getBTPNetworkTypeInfo(ntid, height.add(BigInteger.ONE)).execute();
        assertEquals(oInfo, nInfo);
    }

    // for openBTPNetwork
    private void checkNetwork(BigInteger nid, String name, BigInteger ntid, BigInteger startHeight, Address owner) throws Exception {
        BTPNetworkInfo nInfo = iconService.getBTPNetworkInfo(nid).execute();
        assertEquals(nid, nInfo.getNetworkID());
        assertEquals(name, nInfo.getNetworkName());
        assertEquals(ntid, nInfo.getNetworkTypeID());
        assertEquals(startHeight, nInfo.getStartHeight());
        assertTrue(nInfo.getOpen());
        assertEquals(owner, nInfo.getOwner());
        assertNull(nInfo.getPrevNSHash());
    }

    // for closeBTPNetwork and public key modification
    private void checkNetwork(BigInteger height, BigInteger nid, boolean open) throws Exception {
        BTPNetworkInfo oInfo = iconService.getBTPNetworkInfo(nid, height).execute();
        BTPNetworkInfo nInfo = iconService.getBTPNetworkInfo(nid).execute();
        assertEquals(open, nInfo.getOpen());
        if (open) {
            assertEquals(oInfo.getLastNSHash(), nInfo.getPrevNSHash());
        } else {
            assertEquals(oInfo.getPrevNSHash(), nInfo.getPrevNSHash());
            assertEquals(oInfo.getLastNSHash(), nInfo.getLastNSHash());
        }
    }

    // for sendBTPMessage
    private void checkNetwork(BigInteger height, BigInteger nid, int msgCount) throws Exception {
        BTPNetworkInfo oInfo = iconService.getBTPNetworkInfo(nid, height).execute();
        BTPNetworkInfo nInfo = iconService.getBTPNetworkInfo(nid).execute();
        assertTrue(nInfo.getOpen());
        assertEquals(oInfo.getLastNSHash(), nInfo.getPrevNSHash());
        assertEquals(oInfo.getNextMessageSN().add(BigInteger.valueOf(msgCount)), nInfo.getNextMessageSN());
    }

    private void checkNetworkNotChanged(BigInteger height, BigInteger nid) throws Exception {
        BTPNetworkInfo oInfo = iconService.getBTPNetworkInfo(nid, height).execute();
        BTPNetworkInfo nInfo = iconService.getBTPNetworkInfo(nid, height.add(BigInteger.ONE)).execute();
        assertEquals(oInfo, nInfo);
    }

    // for openBTPHeader and public key modification
    private void checkHeader(BigInteger height, BigInteger nid) throws Exception {
        Base64 blkB64 = iconService.getBTPHeader(nid, height).execute();
        var header = new BTPBlockHeader(blkB64.decode(), Codec.rlp);
        assertTrue(header.getNextProofContextChanged());
        assertNotNull(header.getNextProofContext());
        assertEquals(0, header.getMessageCount());
    }

    // for sendBTPMessage
    private void checkHeader(BigInteger height, BigInteger nid, int firstMsgSN, int msgCount) throws Exception {
        Base64 blkB64 = iconService.getBTPHeader(nid, height).execute();
        var header = new BTPBlockHeader(blkB64.decode(), Codec.rlp);
        assertEquals(firstMsgSN, header.getFirstMessageSN());
        assertEquals(msgCount, header.getMessageCount());
        assertFalse(header.getNextProofContextChanged());
        assertNull(header.getNextProofContext());
    }

    private void checkMessage(BigInteger height, BigInteger nid, byte[][] msgs) throws Exception {
        Base64[] msgsB64 = iconService.getBTPMessages(nid, height).execute();
        assertEquals(msgs.length, msgsB64.length);
        for (int i = 0; i < msgs.length; i++) {
            assertArrayEquals(msgs[i], msgsB64[i].decode());
        }
    }

    private void checkEventLog(List<TransactionResult.EventLog> eventLogs, BigInteger nid, int msgSN) {
        boolean checked = false;
        for (TransactionResult.EventLog el : eventLogs) {
            if (el.getIndexed().get(0).asString().equals("BTPMessage(int,int)") &&
                    el.getIndexed().get(1).asInteger().equals(nid) &&
                    el.getIndexed().get(2).asInteger().equals(BigInteger.valueOf(msgSN))
            ) {
                checked = true;
                break;
            }
        }
        assertTrue(checked);
    }
}
