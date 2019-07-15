package foundation.icon.test.cases;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.*;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.*;
import org.bouncycastle.jcajce.provider.digest.SHA3;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.msgpack.jackson.dataformat.MessagePackFactory;

import java.math.BigInteger;
import java.util.Arrays;
import java.util.List;

import static foundation.icon.test.common.Env.LOG;

@Tag(Constants.TAG_NORMAL)
public class BtpApiTest {
    private static IconService iconService;
    private static Env.Chain chain;

    @BeforeAll
    public static void init() {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
    }

    @Test
    public void ApiTest() throws Exception {
        LOG.infoEntering("BtpAPI");
        final int votesHashIndex = 5;
        KeyWallet wallet = KeyWallet.create();
        Address addr = wallet.getAddress();
        LOG.info("addr : " + addr);

        Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, addr, new BigInteger("1"));
        LOG.info("txHash : " + txHash);
        TransactionResult result = Utils.getTransactionResult(
                iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        BigInteger blkHeight = result.getBlockHeight();
        Base64 blkHeader = iconService.getBlockHeaderByHeight(blkHeight).execute();
        byte headerBytes[] = blkHeader.decode();
        byte blkHash[] = new SHA3.Digest256().digest(headerBytes);
        if (!Arrays.equals(result.getBlockHash().toByteArray(), blkHash)) {
            LOG.info("blkHeight (" + blkHeight + ")");
            LOG.info("headerBytes (" + Utils.byteArrayToHex(headerBytes) + ")");
            LOG.info("blkHash (" + Utils.byteArrayToHex(blkHash) + ")");
            LOG.info("result.getBlockHash() (" + result.getBlockHash() + ")");
            LOG.infoExiting();
            throw new Exception();
        }

        ObjectMapper objectMapper = new ObjectMapper(new MessagePackFactory());
        List<Object> dBlkHeader = objectMapper.readValue(headerBytes, new TypeReference<List<Object>>() {});
        byte votesHash[] = (byte[])dBlkHeader.get(votesHashIndex);

        // below test has to be guaranteed to receive votes from finalized block
        // get votes by hash of the votes
        Base64 vote1 = iconService.getDataByHash(new Bytes(votesHash)).execute();
        byte[] vote1Hash = new SHA3.Digest256().digest(vote1.decode());
        if (!Arrays.equals(votesHash, vote1Hash)) {
            LOG.info("blkHeight (" + blkHeight + ")");
            LOG.info("headerBytes (" + Utils.byteArrayToHex(headerBytes) + ")");
            LOG.info("votes (" + Utils.byteArrayToHex(vote1.decode()) + ")");
            LOG.info("votesHash (" + Utils.byteArrayToHex(votesHash) + ")");
            LOG.info("vote1Hash (" + Utils.byteArrayToHex(vote1Hash) + ")");
            LOG.infoExiting();
            throw new Exception();
        }

        // get block header by hash of the block
        Base64 blkHeader2 = iconService.getDataByHash(result.getBlockHash()).execute();
        if (!Arrays.equals(headerBytes, blkHeader2.decode())) {
            LOG.info("blkHeight (" + blkHeight + ")");
            LOG.info("headerBytes (" + Utils.byteArrayToHex(headerBytes) + ")");
            LOG.info("blkHeader2 (" + Utils.byteArrayToHex(blkHeader2.decode()) + ")");
            LOG.info("getBlockHash (" + result.getBlockHash() + ")");
            LOG.infoExiting();
            throw new Exception();
        }
        LOG.infoExiting();
    }
}
