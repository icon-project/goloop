package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.fail;

@Tag(Constants.TAG_NORMAL)
public class InvokeTestNormal {
    private static IconService iconService;
    private static Env.Chain chain;
    private static KeyWallet ownerWallet;
    private static KeyWallet callerWallet;
    private static Score testScore;
    private static final String PATH = Constants.SCORE_HELLOWORLD_PATH;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        initScore();
    }

    private static void initScore() throws Exception {
        ownerWallet = KeyWallet.create();
        callerWallet = KeyWallet.create();
        Address []addrs = {ownerWallet.getAddress(), callerWallet.getAddress(), chain.governorWallet.getAddress()};
        Utils.transferAndCheck(iconService, chain, chain.godWallet, addrs, Constants.DEFAULT_BALANCE);

        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        Address sAddr = Score.install(iconService, chain, ownerWallet, PATH, params);
        testScore = new Score(iconService, chain, sAddr);
    }

    @Test
    public void invalidMethodName() throws Exception {
        LOG.infoEntering("invalidMethodName");
        final String correctMethod = "helloWithName";
        for(String method : new String[]{correctMethod, "helloWithName2", "hi"}) {
            try {
                RpcObject params = new RpcObject.Builder()
                        .put("name", new RpcValue("ICONLOOP"))
                        .build();
                LOG.infoEntering( "invoke");
                TransactionResult result =
                        testScore.invokeAndWaitResult(callerWallet, method,
                                params, BigInteger.valueOf(0), BigInteger.valueOf(100));
                LOG.infoExiting();
                assertEquals(Constants.STATUS_SUCCESS.equals(result.getStatus()), method.equals(correctMethod));
            } catch (ResultTimeoutException ex) {
                assertTrue(!method.equals(correctMethod));
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void invalidParamName() throws Exception {
        LOG.infoEntering("invalidParamName");
        for(String param : new String[]{"name", "nami"}) {
            try {
                RpcObject params = new RpcObject.Builder()
                        .put(param, new RpcValue("ICONLOOP"))
                        .build();
                LOG.infoEntering( "invoke");
                TransactionResult result =
                        testScore.invokeAndWaitResult(callerWallet, "helloWithName",
                                params, BigInteger.valueOf(0), BigInteger.valueOf(100));
                LOG.infoExiting();
                assertEquals(Constants.STATUS_SUCCESS.equals(result.getStatus()), param.equals("name"));
            } catch (ResultTimeoutException ex) {
                assertTrue(!param.equals("name"));
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void unexpectedParam() throws Exception {
        LOG.infoEntering("invalidParamNum");
        String[][] params = new String[][]{{}, {"age"}, {"name"}, {"name", "age"}, {"name", "etc"}, {"name", "age", "etc"}};
        for(int i = 0; i < params.length; i++) {
            try {
                RpcObject.Builder builder = new RpcObject.Builder();
                for(String param: params[i]){
                    builder.put(param, new RpcValue("ICONLOOP"));
                }
                RpcObject objParam = builder.build();
                LOG.infoEntering("invoke");
                TransactionResult result = testScore.invokeAndWaitResult(callerWallet,
                        "helloWithName", objParam, BigInteger.valueOf(0), BigInteger.valueOf(100));
                assertEquals(i == 2 || i == 3, Constants.STATUS_SUCCESS.equals(result.getStatus()));
                LOG.infoExiting();
            } catch (ResultTimeoutException ex) {
                fail("Should not be reached");
            }
        }
        LOG.infoExiting();
    }

    /*
     * If Governance SCORE has not been deployed, anyone can initially install Governance SCORE.
     */
    @Test
    public void deployGovScore() throws Exception {
        LOG.infoEntering("setGovernance");
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .put("value", new RpcValue(BigInteger.ONE))
                .build();

        // deploy tx to install governance
        KeyWallet govOwner = KeyWallet.create();
        LOG.infoEntering("install governance score");
        Bytes txHash = Utils.deployScore(iconService, chain.networkId,
                govOwner, Constants.GOV_ADDRESS, Constants.SCORE_GOV_PATH, params);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        LOG.infoExiting("result : " + result);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());

        // check install result
        boolean updated = Utils.icxCall(iconService, Constants.GOV_ADDRESS, "updated", null).asBoolean();
        assertFalse(updated);

        // failed when deploy tx with another address
        LOG.infoEntering("update governance score with not owner");
        txHash = Utils.deployScore(iconService, chain.networkId,
                KeyWallet.create(), Constants.GOV_ADDRESS, Constants.SCORE_GOV_UPDATE_PATH, null);
        result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        LOG.infoExiting("result : " + result);
        assertEquals(Constants.STATUS_FAIL, result.getStatus());
        updated = Utils.icxCall(iconService, Constants.GOV_ADDRESS, "updated", null).asBoolean();
        assertFalse(updated);

        // success when deploy tx with owner
        LOG.infoEntering("update governance score with owner");
        txHash = Utils.deployScore(iconService, chain.networkId,
                govOwner, Constants.GOV_ADDRESS, Constants.SCORE_GOV_UPDATE_PATH, null);
        result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        LOG.infoExiting("result : " + result);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());

        // check update result
        updated = Utils.icxCall(iconService, Constants.GOV_ADDRESS, "updated", null).asBoolean();
        assertTrue(updated);
        LOG.infoExiting();
    }
}
