package foundation.icon.test.cases;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.*;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.*;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.msgpack.jackson.dataformat.MessagePackFactory;

import java.math.BigInteger;
import java.util.Arrays;
import java.util.List;

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
        final int votesHashIndex = 5;
        KeyWallet wallet = KeyWallet.create();
        Address addr = wallet.getAddress();

        Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, addr, new BigInteger("1"));
        TransactionResult result = Utils.getTransactionResult(
                iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        BigInteger blkHeight = result.getBlockHeight();
        Base64 blkHeader = iconService.getBlockHeaderByHeight(blkHeight).execute();
        byte headerBytes[] = blkHeader.decode();
        ObjectMapper objectMapper = new ObjectMapper(new MessagePackFactory());
        List<Object> dBlkHeader = objectMapper.readValue(headerBytes, new TypeReference<List<Object>>() {});
        byte votesHash[] = (byte[])dBlkHeader.get(votesHashIndex);

        // get votes by hash of the votes
        Base64 vote1 = iconService.getDataByHash(new Bytes(votesHash)).execute();
        Base64 vote2 = iconService.getVotesByHeight(blkHeight.subtract(BigInteger.ONE)).execute();
        if (!Arrays.equals(vote1.decode(), vote2.decode())) {
            throw new Exception();
        }

        // get block header by hash of the block
        Base64 blkHeader2 = iconService.getDataByHash(result.getBlockHash()).execute();
        if (!Arrays.equals(headerBytes, blkHeader2.decode())) {
            throw new Exception();
        }
    }
}
