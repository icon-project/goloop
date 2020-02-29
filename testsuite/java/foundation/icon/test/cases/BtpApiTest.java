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

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import foundation.icon.ee.util.Crypto;
import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Base64;
import foundation.icon.icx.data.Block;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.msgpack.jackson.dataformat.MessagePackFactory;

import java.math.BigInteger;
import java.util.Arrays;
import java.util.LinkedList;
import java.util.List;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.fail;

@Tag(Constants.TAG_PY_SCORE)
public class BtpApiTest extends TestBase {
    private static TransactionHandler txHandler;
    private static IconService iconService;

    private final static int PREVID_INDEX = 4;
    private final static int VOTESHASH_INDEX = 5;
    private final static int NEXTVALIDATORHASH_INDEX = 6;

    @BeforeAll
    static void init() {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
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
        TransactionResult result = txHandler.getResult(txHash, Constants.DEFAULT_WAITING_TIME);
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
            LOG.infoExiting();
            throw new Exception();
        }

        ObjectMapper objectMapper = new ObjectMapper(new MessagePackFactory());
        List<Object> dBlkHeader = objectMapper.readValue(resHeaderBytes, new TypeReference<List<Object>>() {});
        byte []votesHash = (byte[])dBlkHeader.get(VOTESHASH_INDEX);

        // get votes by hash of the votes
        Base64 votes = iconService.getDataByHash(new Bytes(votesHash)).execute();

        // extract vote
        byte[] bVotes = votes.decode();
        List<Object> dVotes = objectMapper.readValue(bVotes, new TypeReference<List<Object>>() {});

        // votes : 0 - round, 1 - partSetID, 2 - voteItems
        int round = (int)dVotes.get(0);
        @SuppressWarnings("unchecked")
        List<Object> bPartSetId = (List<Object>)dVotes.get(1);
        @SuppressWarnings("unchecked")
        List<Object> voteItems = (List<Object>)dVotes.get(2);

        // get nextValidator from pprev block
        BigInteger valBlkHeight = resBlkHeight.subtract(BigInteger.valueOf(2)); // block height for validators
        Base64 valBlkHeader = iconService.getBlockHeaderByHeight(valBlkHeight).execute();
        byte[] valHeaderBytes = valBlkHeader.decode();
        List<Object> dvBlkHeader = objectMapper.readValue(valHeaderBytes, new TypeReference<List<Object>>() {});
        byte[] validatorHash = (byte[])dvBlkHeader.get(NEXTVALIDATORHASH_INDEX);
        Base64 validator = iconService.getDataByHash(new Bytes(validatorHash)).execute();
        List<Object> validatorsList = objectMapper.readValue(validator.decode(), new TypeReference<List<Object>>() {});
        byte[] prevBlockID = (byte[])dBlkHeader.get(PREVID_INDEX);
        int twoThirds = validatorsList.size() * 2 / 3;
        int match = 0;
        for (Object voteItem : voteItems) {
            List<Object> vSign = new LinkedList<>();
            vSign.add(resBlkHeight.subtract(BigInteger.ONE));
            vSign.add(round);
            vSign.add(1); // voteTypePrecommit
            vSign.add(prevBlockID);
            vSign.add(bPartSetId);
            // voteItem : 0 - Timestamp, 1 - signature
            @SuppressWarnings("unchecked")
            List<Object> voteItemList = (List<Object>) voteItem;
            vSign.add(voteItemList.get(0));
            byte[] sign = (byte[]) voteItemList.get(1);
            byte[] message = objectMapper.writeValueAsBytes(vSign);
            byte[] msgHash = Crypto.sha3_256(message);
            byte[] pubKey = Crypto.recoverKey(msgHash, sign, false);
            if (pubKey == null) {
                LOG.info("recId(" + sign[64] + "), " +
                        "sign(" + byteArrayToHex(sign) + "), " +
                        "msgHash(" + byteArrayToHex(message) + ")");
                LOG.infoExiting();
                fail("cannot recover pubkey from signature");
            }
            byte[] recovered = Crypto.getAddressBytesFromKey(pubKey);
            for (Object vo : validatorsList) {
                if (Arrays.equals((byte[]) vo, recovered)) {
                    match++;
                    validatorsList.remove(vo);
                    break;
                }
            }
        }
        if (validatorsList.size() != 0) {
            for (Object vo : validatorsList) {
                LOG.info("No vote validator : " + byteArrayToHex((byte[])vo));
            }
        }
        if (twoThirds >= match) {
            fail("match must be bigger than twoThirds but match (" + match + "), twoThirds (" + twoThirds + ")");
        }
        LOG.infoExiting();
    }

    @Test
    public void apiTest() throws Exception {
        LOG.infoEntering("apiTest");
        KeyWallet wallet = KeyWallet.create();

        LOG.infoEntering("sendTransaction");
        Bytes txHash = txHandler.transfer(wallet.getAddress(), BigInteger.ONE);
        TransactionResult result = txHandler.getResult(txHash, Constants.DEFAULT_WAITING_TIME);
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

        ObjectMapper objectMapper = new ObjectMapper(new MessagePackFactory());
        List<Object> dBlkHeader = objectMapper.readValue(resHeaderBytes, new TypeReference<List<Object>>() {});
        byte []votesHash = (byte[])dBlkHeader.get(VOTESHASH_INDEX);

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

        byte[] nextValidatorHash = (byte[])dBlkHeader.get(NEXTVALIDATORHASH_INDEX);
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
