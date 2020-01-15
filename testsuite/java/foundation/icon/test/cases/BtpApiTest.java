package foundation.icon.test.cases;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.*;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.*;
import org.bouncycastle.util.BigIntegers;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.msgpack.jackson.dataformat.MessagePackFactory;
import static org.junit.jupiter.api.Assertions.*;

import java.math.BigInteger;
import java.util.Arrays;
import java.util.LinkedList;
import java.util.List;

import static foundation.icon.test.common.Env.LOG;


@Tag(Constants.TAG_PY_SCORE)
public class BtpApiTest {
    private static IconService iconService;
    private static Env.Chain chain;
    private final static int PREVID_INDEX = 4;
    private final static int VOTESHASH_INDEX = 5;
    private final static int NEXTVALIDATORHASH_INDEX = 6;

    @BeforeAll
    public static void init() {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
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
        KeyWallet wallet = KeyWallet.create();
        Address addr = wallet.getAddress();
        LOG.infoEntering("verifyVotes");

        LOG.infoEntering("sendTransaction");
        Bytes txHash = Utils.transfer(iconService, chain.networkId,
                chain.godWallet, addr, new BigInteger("1"));
        LOG.infoExiting();
        TransactionResult result = Utils.getTransactionResult(
                iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        BigInteger resBlkHeight = result.getBlockHeight();
        Base64 resBlkHeader = iconService.getBlockHeaderByHeight(resBlkHeight).execute();
        byte []resHeaderBytes = resBlkHeader.decode();
        byte []blkHash = Utils.getHash(resHeaderBytes);
        if (!Arrays.equals(result.getBlockHash().toByteArray(), blkHash)) {
            LOG.info("blkHeight (" + resBlkHeight + ")");
            LOG.info("headerBytes (" + Utils.byteArrayToHex(resHeaderBytes) + ")");
            LOG.info("blkHash (" + Utils.byteArrayToHex(blkHash) + ")");
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
        for(Object voteItem : voteItems) {
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
            byte[] msgHash = Utils.getHash(message);
            BigInteger[] sig = new BigInteger[2];
            sig[0] = BigIntegers.fromUnsignedByteArray(sign, 0, 32);
            sig[1] = BigIntegers.fromUnsignedByteArray(sign, 32, 32);

            byte[] recover = new byte[21];
            recover[0] = 0;
            byte[] pubKey = Utils.recoverFromSignature((int) sign[64], sig, msgHash);
            if(pubKey == null) {
                LOG.info("redId(" + sign[64] + "), sig[0](" + sig[0] +
                        "), sig[1](" + sig[1] + ")" + ", msgHash(" + Utils.byteArrayToHex(message) + ")");
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
        if(validatorsList.size() != 0){
            for(Object vo : validatorsList) {
                LOG.info("No vote validator  : " + Utils.byteArrayToHex((byte[])vo));
            }
        }
        if(twoThirds >= match) {
            fail("match must be bigger than twoThirds but match (" + match + "), twoThrids (" + twoThirds + ")");
        }
        LOG.infoExiting();
    }

    @Test
    public void ApiTest() throws Exception {
        KeyWallet wallet = KeyWallet.create();
        Address addr = wallet.getAddress();

        LOG.infoEntering("ApiTest");
        LOG.infoEntering("sendTransaction");
        Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, addr, new BigInteger("1"));
        LOG.infoExiting();
        TransactionResult result = Utils.getTransactionResult(
                iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        BigInteger resBlkHeight = result.getBlockHeight();
        Base64 resBlkHeader = iconService.getBlockHeaderByHeight(resBlkHeight).execute();
        byte []resHeaderBytes = resBlkHeader.decode();
        byte []blkHash = Utils.getHash(resHeaderBytes);
        if (!Arrays.equals(result.getBlockHash().toByteArray(), blkHash)) {
            LOG.info("blkHeight (" + resBlkHeight + ")");
            LOG.info("headerBytes (" + Utils.byteArrayToHex(resHeaderBytes) + ")");
            LOG.info("blkHash (" + Utils.byteArrayToHex(blkHash) + ")");
            LOG.info("result.getBlockHash() (" + result.getBlockHash() + ")");
            throw new Exception();
        }

        ObjectMapper objectMapper = new ObjectMapper(new MessagePackFactory());
        List<Object> dBlkHeader = objectMapper.readValue(resHeaderBytes, new TypeReference<List<Object>>() {});
        byte []votesHash = (byte[])dBlkHeader.get(VOTESHASH_INDEX);

        // get votes by hash of the votes
        Base64 votes = iconService.getDataByHash(new Bytes(votesHash)).execute();
        byte[] voteHash2 = Utils.getHash(votes.decode());
        if (!Arrays.equals(votesHash, voteHash2)) {
            LOG.info("blkHeight (" + resBlkHeight + ")");
            LOG.info("headerBytes (" + Utils.byteArrayToHex(resHeaderBytes) + ")");
            LOG.info("votes (" + Utils.byteArrayToHex(votes.decode()) + ")");
            LOG.info("votesHash (" + Utils.byteArrayToHex(votesHash) + ")");
            LOG.info("vote1Hash (" + Utils.byteArrayToHex(voteHash2) + ")");
            throw new Exception();
        }

        byte[] nextValidatorHash = (byte[])dBlkHeader.get(NEXTVALIDATORHASH_INDEX);
        Base64 nextValidator = iconService.getDataByHash(new Bytes(nextValidatorHash)).execute();
        byte[] vHash = Utils.getHash(nextValidator.decode());
        if(!Arrays.equals(vHash, nextValidatorHash)) {
            LOG.info("blkHeight (" + resBlkHeight + ")");
            LOG.info("headerBytes (" + Utils.byteArrayToHex(resHeaderBytes) + ")");
            LOG.info("votesHash (" + Utils.byteArrayToHex(votesHash) + ")");
            LOG.info("vHash (" + Utils.byteArrayToHex(vHash) + ")");
            LOG.info("nextValidatorHash (" + Utils.byteArrayToHex(nextValidatorHash) + ")");
            LOG.infoExiting();
            throw new Exception();
        }

        // get block header by hash of the block
        Base64 blkHeader2 = iconService.getDataByHash(result.getBlockHash()).execute();
        if (!Arrays.equals(resHeaderBytes, blkHeader2.decode())) {
            LOG.info("blkHeight (" + resBlkHeight + ")");
            LOG.info("headerBytes (" + Utils.byteArrayToHex(resHeaderBytes) + ")");
            LOG.info("blkHeader2 (" + Utils.byteArrayToHex(blkHeader2.decode()) + ")");
            LOG.info("getBlockHash (" + result.getBlockHash() + ")");
            LOG.infoExiting();
            throw new Exception();
        }
        LOG.infoExiting();
    }
}
