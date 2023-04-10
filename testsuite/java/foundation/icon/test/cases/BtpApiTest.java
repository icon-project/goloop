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

import foundation.icon.ee.util.Crypto;
import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Base64;
import foundation.icon.icx.data.Block;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.test.common.*;
import foundation.icon.test.score.ChainScore;
import foundation.icon.test.score.EventGen;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.util.Arrays;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.fail;

@Tag(Constants.TAG_JAVA_SCORE)
public class BtpApiTest extends TestBase {
    private static TransactionHandler txHandler;
    private static IconService iconService;
    private static Codec codec;
    private static ChainScore chainScore;

    @BeforeAll
    static void init() {
        Env.Node node = Env.nodes[0];
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
    }

    /*
    send transaction
    get result by hash
    get block header
    get votes hash from the block header
    get votes by votes hash -> this votes is for previous block
    get validators from previous previous block
    verify votes with validators
     */
    @Test
    public void verifyVotes() throws Exception {
        LOG.infoEntering("verifyVotes");
        KeyWallet wallet = KeyWallet.create();

        LOG.infoEntering("sendTransaction");
        Bytes txHash = txHandler.transfer(wallet.getAddress(), BigInteger.ONE);
        TransactionResult result = txHandler.getResult(txHash);
        LOG.infoExiting();

        BigInteger rBlkHeight = result.getBlockHeight();
        Base64 rBlkB64 = iconService.getBlockHeaderByHeight(rBlkHeight).execute();
        byte[] rBlkBytes = rBlkB64.decode();
        byte[] rBlkHash = Crypto.sha3_256(rBlkBytes);
        if (!Arrays.equals(result.getBlockHash().toByteArray(), rBlkHash)) {
            LOG.info("blkHeight (" + rBlkHeight + ")");
            LOG.info("headerBytes (" + byteArrayToHex(rBlkBytes) + ")");
            LOG.info("blkHash (" + byteArrayToHex(rBlkHash) + ")");
            LOG.info("result.getBlockHash() (" + result.getBlockHash() + ")");
            LOG.infoExiting();
            throw new Exception();
        }
        var rBlk = new BlockHeader(rBlkBytes, codec);

        Base64 rVotesB64 = iconService.getDataByHash(new Bytes(rBlk.getVotesHash())).execute();
        var rVotes = new Votes(rVotesB64.decode(), codec);

        // get nextValidator from pprev block
        Base64 vBlkB64 = iconService.getBlockHeaderByHeight(BigInteger.valueOf(rBlk.getHeight()-2)).execute();
        var vBlk = new BlockHeader(vBlkB64.decode(), codec);
        Base64 vValidatorsB64 = iconService.getDataByHash(new Bytes(vBlk.getNextValidatorHash())).execute();
        var vValidators = new ValidatorList(vValidatorsB64.decode(), codec);

        LOG.info("validator number = " + vValidators.size());

        // verify votes.
        int twoThirds = vValidators.size() * 2 / 3;
        var verified = rVotes.verifyVotes(rBlk, vValidators, codec);
        if (verified <= twoThirds) {
            fail("match must be bigger than twoThirds but verified (" + verified + "), twoThirds (" + twoThirds + ")");
        }
        LOG.infoExiting();
    }

    @Test
    public void apiTest() throws Exception {
        LOG.infoEntering("apiTest");
        KeyWallet wallet = KeyWallet.create();

        LOG.infoEntering("sendTransaction");
        Bytes txHash = txHandler.transfer(wallet.getAddress(), BigInteger.ONE);
        TransactionResult result = txHandler.getResult(txHash);
        LOG.infoExiting();

        BigInteger resBlkHeight = result.getBlockHeight();
        Base64 resBlkHeader = iconService.getBlockHeaderByHeight(resBlkHeight).execute();
        byte[] resHeaderBytes = resBlkHeader.decode();
        byte[] blkHash = Crypto.sha3_256(resHeaderBytes);
        if (!Arrays.equals(result.getBlockHash().toByteArray(), blkHash)) {
            LOG.info("blkHeight (" + resBlkHeight + ")");
            LOG.info("headerBytes (" + byteArrayToHex(resHeaderBytes) + ")");
            LOG.info("blkHash (" + byteArrayToHex(blkHash) + ")");
            LOG.info("result.getBlockHash() (" + result.getBlockHash() + ")");
            throw new Exception();
        }

        var dBlk = new BlockHeader(resHeaderBytes, codec);
        byte []votesHash = dBlk.getVotesHash();

        // get votes by hash of the votes
        Base64 votes = iconService.getDataByHash(new Bytes(votesHash)).execute();
        byte[] voteHash2 = Crypto.sha3_256(votes.decode());
        if (!Arrays.equals(votesHash, voteHash2)) {
            LOG.info("blkHeight (" + resBlkHeight + ")");
            LOG.info("headerBytes (" + byteArrayToHex(resHeaderBytes) + ")");
            LOG.info("votes (" + byteArrayToHex(votes.decode()) + ")");
            LOG.info("votesHash (" + byteArrayToHex(votesHash) + ")");
            LOG.info("vote1Hash (" + byteArrayToHex(voteHash2) + ")");
            throw new Exception();
        }

        byte[] nextValidatorHash = dBlk.getNextValidatorHash();
        Base64 nextValidator = iconService.getDataByHash(new Bytes(nextValidatorHash)).execute();
        byte[] vHash = Crypto.sha3_256(nextValidator.decode());
        if(!Arrays.equals(vHash, nextValidatorHash)) {
            LOG.info("blkHeight (" + resBlkHeight + ")");
            LOG.info("headerBytes (" + byteArrayToHex(resHeaderBytes) + ")");
            LOG.info("votesHash (" + byteArrayToHex(votesHash) + ")");
            LOG.info("vHash (" + byteArrayToHex(vHash) + ")");
            LOG.info("nextValidatorHash (" + byteArrayToHex(nextValidatorHash) + ")");
            LOG.infoExiting();
            throw new Exception();
        }

        // get block header by hash of the block
        Base64 blkHeader2 = iconService.getDataByHash(result.getBlockHash()).execute();
        if (!Arrays.equals(resHeaderBytes, blkHeader2.decode())) {
            LOG.info("blkHeight (" + resBlkHeight + ")");
            LOG.info("headerBytes (" + byteArrayToHex(resHeaderBytes) + ")");
            LOG.info("blkHeader2 (" + byteArrayToHex(blkHeader2.decode()) + ")");
            LOG.info("getBlockHash (" + result.getBlockHash() + ")");
            LOG.infoExiting();
            throw new Exception();
        }
        LOG.infoExiting();
    }

    @Test
    public void negativeTest() throws Exception {
        final long ErrNotFound = -31004;

        LOG.infoEntering("getVotesByHeight");
        try {
            // test with non-existent height
            iconService.getVotesByHeight(BigInteger.valueOf(99999999)).execute();
            fail();
        } catch (RpcError e) {
            if (e.getCode() == ErrNotFound) {
                LOG.info("Expected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
            } else {
                LOG.info("Unexpected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
                fail();
            }
        }
        LOG.infoExiting();

        LOG.infoEntering("getProofForResult");
        KeyWallet wallet = KeyWallet.create();
        Bytes txHash = txHandler.transfer(wallet.getAddress(), BigInteger.ONE);
        TransactionResult txResult = txHandler.getResult(txHash);
        BigInteger height = txResult.getBlockHeight();
        Block txBlock = iconService.getBlock(height).execute();
        Block resultBlock = iconService.getBlock(height.add(BigInteger.ONE)).execute();
        try {
            // test with invalid block hash
            BigInteger index = txResult.getTxIndex();
            iconService.getProofForResult(txBlock.getBlockHash(), index).execute();
            fail();
        } catch (RpcError e) {
            if (e.getCode() == ErrNotFound) {
                LOG.info("Expected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
            } else {
                LOG.info("Unexpected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
                fail();
            }
        }
        try {
            // test with invalid index
            BigInteger invalidIndex = BigInteger.valueOf(txBlock.getTransactions().size() + 1);
            iconService.getProofForResult(resultBlock.getBlockHash(), invalidIndex).execute();
            fail();
        } catch (RpcError e) {
            if (e.getCode() == ErrNotFound) {
                LOG.info("Expected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
            } else {
                LOG.info("Unexpected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
                fail();
            }
        }
        LOG.infoExiting();

        LOG.infoEntering("getProofForEvents");
        KeyWallet ownerWallet = KeyWallet.create();
        EventGen eventGen = EventGen.install(txHandler, ownerWallet, Constants.CONTENT_TYPE_JAVA);
        txHash = eventGen.invokeGenerate(ownerWallet, ownerWallet.getAddress(), BigInteger.ONE, new byte[]{1});
        txResult = txHandler.getResult(txHash);
        height = txResult.getBlockHeight();
        txBlock = iconService.getBlock(height).execute();
        resultBlock = iconService.getBlock(height.add(BigInteger.ONE)).execute();
        Bytes blockHash = resultBlock.getBlockHash();
        BigInteger index = txResult.getTxIndex();
        BigInteger[] events = new BigInteger[txResult.getEventLogs().size()];
        for (int i = 0; i < txResult.getEventLogs().size(); i++) {
            events[i] = BigInteger.valueOf(i);
        }
        try {
            // test with invalid block hash
            Bytes invalidBlockHash = txBlock.getBlockHash();
            iconService.getProofForEvents(invalidBlockHash, index, events).execute();
            fail();
        } catch (RpcError e) {
            if (e.getCode() == ErrNotFound) {
                LOG.info("Expected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
            } else {
                LOG.info("Unexpected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
                fail();
            }
        }
        try {
            // test with invalid index
            BigInteger invalidIndex = BigInteger.valueOf(txBlock.getTransactions().size() + 1);
            iconService.getProofForEvents(blockHash, invalidIndex, events).execute();
            fail();
        } catch (RpcError e) {
            if (e.getCode() == ErrNotFound) {
                LOG.info("Expected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
            } else {
                LOG.info("Unexpected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
                fail();
            }
        }
        final int requiredRevision = 7;
        int revision = chainScore.getRevision();
        if (revision < requiredRevision) {
            LOG.info("Ignore invalid events test at revision : "+revision);
        } else {
            try {
                // test with invalid events
                BigInteger[] invalidEvetns = new BigInteger[]{BigInteger.valueOf(txResult.getEventLogs().size() + 1)};
                iconService.getProofForEvents(blockHash, index, invalidEvetns).execute();
                fail();
            } catch (RpcError e) {
                if (e.getCode() == ErrNotFound) {
                    LOG.info("Expected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
                } else {
                    LOG.info("Unexpected RpcError: code=" + e.getCode() + ", msg=" + e.getMessage());
                    fail();
                }
            }
        }
        LOG.infoExiting();
    }

    private static String byteArrayToHex(byte[] array) {
        StringBuilder sb = new StringBuilder();
        sb.append("0x");
        for (byte v : array) {
            sb.append(String.format("%02x", v));
        }
        return sb.toString();
    }
}
