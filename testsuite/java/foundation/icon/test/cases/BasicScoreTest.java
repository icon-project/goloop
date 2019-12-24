package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

// TODO What about adding annotation indicating requirements. For example,
// "@require(nodeNum=4,chainNum=1)" indicates it requires at least 4 nodes and
// 1 chain for each.
@Tag(Constants.TAG_NORMAL)
public class BasicScoreTest {
    private static Env.Chain chain;
    private static IconService iconService;

    @BeforeAll
    public static void setUp() {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
    }

    @Test
    public void deployGovScore() throws Exception {
        LOG.infoEntering("setGovernance");
        final String gPath = Constants.SCORE_GOV_PATH;
        final String guPath = Constants.SCORE_GOV_UPDATE_PATH;

        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .put("value", new RpcValue("0x1"))
                .build();

        // deploy tx to install governance
        KeyWallet govOwner = KeyWallet.create();
        LOG.infoEntering("install governance score");
        Bytes txHash = Utils.deployScore(iconService, chain.networkId,
                govOwner, Constants.GOV_ADDRESS, gPath, params);
        TransactionResult result = Utils.getTransactionResult(iconService,
                txHash, Constants.DEFAULT_WAITING_TIME);
        LOG.infoExiting("result : " + result);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());

        // check install result
        boolean updated = Utils.icxCall(iconService,
                Constants.GOV_ADDRESS, "updated",null).asBoolean();
        assertTrue(!updated);

        // failed when deploy tx with another address
        LOG.infoEntering("update governance score with not owner");
        txHash = Utils.deployScore(iconService, chain.networkId,
                KeyWallet.create(), Constants.GOV_ADDRESS, guPath, null);
        result = Utils.getTransactionResult(iconService,
                txHash, Constants.DEFAULT_WAITING_TIME);
        LOG.infoExiting("result : " + result);
        assertEquals(Constants.STATUS_FAIL, result.getStatus());
        updated = Utils.icxCall(iconService, Constants.GOV_ADDRESS,
                "updated",null).asBoolean();
        assertTrue(!updated);

        // success when deploy tx with owner
        LOG.infoEntering("update governance score with owner");
        txHash = Utils.deployScore(iconService, chain.networkId,
                govOwner, Constants.GOV_ADDRESS, guPath, null);
        result = Utils.getTransactionResult(iconService,
                txHash, Constants.DEFAULT_WAITING_TIME);
        LOG.infoExiting("result : " + result);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());

        // check update result
        updated = Utils.icxCall(iconService, Constants.GOV_ADDRESS,
                "updated",null).asBoolean();
        assertTrue(updated);
        LOG.infoExiting();

    }
}
