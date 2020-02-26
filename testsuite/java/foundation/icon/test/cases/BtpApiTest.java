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
import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.crypto.IconKeys;
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
import org.bouncycastle.asn1.x9.X9ECParameters;
import org.bouncycastle.asn1.x9.X9IntegerConverter;
import org.bouncycastle.crypto.ec.CustomNamedCurves;
import org.bouncycastle.crypto.params.ECDomainParameters;
import org.bouncycastle.jcajce.provider.digest.SHA3;
import org.bouncycastle.math.ec.ECAlgorithms;
import org.bouncycastle.math.ec.ECPoint;
import org.bouncycastle.math.ec.custom.sec.SecP256K1Curve;
import org.bouncycastle.util.BigIntegers;
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
        byte []resHeaderBytes = resBlkHeader.decode();
        byte []blkHash = getHash(resHeaderBytes);
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
        byte []valHeaderBytes = valBlkHeader.decode();
        List<Object> dvBlkHeader = objectMapper.readValue(valHeaderBytes, new TypeReference<List<Object>>() {});
        byte []validatorHash = (byte[])dvBlkHeader.get(NEXTVALIDATORHASH_INDEX);
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
            byte[] msgHash = getHash(message);
            BigInteger[] sig = new BigInteger[2];
            sig[0] = BigIntegers.fromUnsignedByteArray(sign, 0, 32);
            sig[1] = BigIntegers.fromUnsignedByteArray(sign, 32, 32);

            byte[] recover = new byte[21];
            recover[0] = 0;
            byte[] pubKey = recoverFromSignature(sign[64], sig, msgHash);
            if(pubKey == null) {
                LOG.info("redId(" + sign[64] + "), sig[0](" + sig[0] +
                        "), sig[1](" + sig[1] + ")" + ", msgHash(" + byteArrayToHex(message) + ")");
                LOG.infoExiting();
                fail("cannot recover pubkey from signature");
            }
            System.arraycopy(pubKey, 0, recover, 1, 20);
            for (Object vo : validatorsList) {
                if (Arrays.equals((byte[]) vo, recover)) {
                    match++;
                    validatorsList.remove(vo);
                    break;
                }
            }
        }
        if (validatorsList.size() != 0){
            for(Object vo : validatorsList) {
                LOG.info("No vote validator  : " + byteArrayToHex((byte[])vo));
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
        byte []resHeaderBytes = resBlkHeader.decode();
        byte []blkHash = getHash(resHeaderBytes);
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
        byte[] voteHash2 = getHash(votes.decode());
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
        byte[] vHash = getHash(nextValidator.decode());
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

    static String byteArrayToHex(byte[] array) {
        StringBuilder sb = new StringBuilder();
        sb.append("0x");
        for(byte v : array) {
            sb.append(String.format("%02x", v));
        }
        return sb.toString();
    }

    static byte[] getHash(byte[] data) {
        return new SHA3.Digest256().digest(data);
    }

    // below codes are from foundation.icon.icx.crypto.ECDSASignature
    private final static X9ECParameters curveParams = CustomNamedCurves.getByName("secp256k1");
    private final static ECDomainParameters curve = new ECDomainParameters(
            curveParams.getCurve(), curveParams.getG(), curveParams.getN(), curveParams.getH());
    private static ECPoint decompressKey(BigInteger xBN, boolean yBit) {
        X9IntegerConverter x9 = new X9IntegerConverter();
        byte[] compEnc = x9.integerToBytes(xBN, 1 + x9.getByteLength(curve.getCurve()));
        compEnc[0] = (byte) (yBit ? 0x03 : 0x02);
        return curve.getCurve().decodePoint(compEnc);
    }

    static byte[] recoverFromSignature(int recId, BigInteger[] sig, byte[] message) {
        BigInteger r = sig[0];
        BigInteger s = sig[1];

        BigInteger n = curve.getN();  // Curve order.
        BigInteger i = BigInteger.valueOf((long) recId / 2);
        BigInteger x = r.add(i.multiply(n));
        BigInteger prime = SecP256K1Curve.q;
        if (x.compareTo(prime) >= 0) {
            return null;
        }
        ECPoint ecPoint = decompressKey(x, (recId & 1) == 1);
        if (!ecPoint.multiply(n).isInfinity()) {
            return null;
        }
        BigInteger e = new BigInteger(1, message);
        BigInteger eInv = BigInteger.ZERO.subtract(e).mod(n);
        BigInteger rInv = r.modInverse(n);
        BigInteger srInv = rInv.multiply(s).mod(n);
        BigInteger eInvrInv = rInv.multiply(eInv).mod(n);
        ECPoint q = ECAlgorithms.sumOfTwoMultiplies(curve.getG(), eInvrInv, ecPoint, srInv);

        byte [] encoded = q.getEncoded(false);
        return IconKeys.getAddressHash(encoded);
    }
}
