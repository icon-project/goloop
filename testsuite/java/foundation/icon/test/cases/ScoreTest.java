package foundation.icon.test.cases;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.*;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.score.Score;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.Assert.*;

import org.junit.AfterClass;
import org.junit.BeforeClass;
import org.junit.Ignore;
import org.junit.Test;

import java.math.BigInteger;

/*
sendTransaction with call
icx_call
stepUsed is bigger than specified stepLimit
 */
public class ScoreTest {
    private static IconService iconService;
    private static Env.Chain chain;
    private static KeyWallet ownerWallet;
    private static GovScore govScore;
    private static Score testScore;
    private static final String PATH = Constants.SCORE_HELLOWORLD_PATH;
    private static final long testCCValue = 10;
    private static final long testStepPrice = 1;

    @BeforeClass
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        initScoreTest();
    }

    private static void initScoreTest() throws Exception {
        ownerWallet = KeyWallet.create();
        Address []addrs = {ownerWallet.getAddress(), chain.governorWallet.getAddress()};
        Utils.transferAndCheck(iconService, chain, chain.godWallet, addrs, Constants.DEFAULT_BALANCE);

        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        Address scoreAddr = Score.install(iconService, chain, ownerWallet, PATH, params);
        testScore = new Score(iconService, chain, scoreAddr);

        govScore = new GovScore(iconService, chain);
        govScore.setMaxStepLimit("invoke", BigInteger.valueOf(1000));
        govScore.setMaxStepLimit("query", BigInteger.valueOf(1000));
        govScore.setStepCost("contractCall", BigInteger.valueOf(testCCValue));
        govScore.setStepPrice(BigInteger.valueOf(testStepPrice));
    }

    @AfterClass
    public static void destroy() throws Exception {
        // TODO set initial value not 0
        govScore.setStepCost("contractCall", BigInteger.valueOf(0));
        govScore.setStepPrice(BigInteger.valueOf(0));
        govScore.setMaxStepLimit("invoke", BigInteger.valueOf(1000));
    }

    @Test
    public void invalidParamName() {
        LOG.infoEntering( "invalidParamName");
        for(String param : new String[]{"name", "nami"}) {
            try {
                RpcObject params = new RpcObject.Builder()
                        .put(param, new RpcValue("ICONLOOP"))
                        .build();
                LOG.infoEntering( "invoke");
                TransactionResult result =
                        testScore.invokeAndWaitResult(chain.godWallet, "helloWithName",
                                params, BigInteger.valueOf(0), BigInteger.valueOf(100));
                LOG.infoExiting();
                assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);
            } catch (ResultTimeoutException ex) {
                assertTrue(!param.equals("name"));
            } catch (Exception ex) {
                fail();
            }
        }
        LOG.infoExiting();
    }

    @Ignore
    @Test
    public void invalidParamNum() {
        LOG.infoEntering( "invalidParamNum");
        for(String []params : new String[][]{{}, {"name"}, {"name", "age"}}) {
            try {
                RpcObject.Builder builder = new RpcObject.Builder();
                for(String param: params){
                        builder.put(param, new RpcValue("ICONLOOP"));
                }
                RpcObject objParam = builder.build();
                LOG.info("invoke param[" + params + "]");
                TransactionResult result = testScore.invokeAndWaitResult(chain.godWallet,
                        "helloWithName", objParam, BigInteger.valueOf(0), BigInteger.valueOf(100));
                assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);
                assertTrue(params.length == 1);
                LOG.infoExiting();
            } catch (ResultTimeoutException ex) {
                assertTrue(params.length != 1);
                LOG.infoExiting();
            } catch (Exception ex) {
                fail();
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void notEnoughStepLimit() throws Exception{
        LOG.infoEntering( "notEnoughStepLimit");
        KeyWallet testWallet = KeyWallet.create();
        long needValue = testCCValue * testStepPrice;
        for(long step : new long[]{testCCValue - 1, testCCValue}) {
            try {
                BigInteger sub = BigInteger.valueOf(needValue).subtract(iconService.getBalance(testWallet.getAddress()).execute());
                if(sub.compareTo(BigInteger.ZERO) > 0) {
                    Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, testWallet.getAddress(), sub.longValue());
                    TransactionResult result = Utils.getTransactionResult(iconService, txHash, 5000);
                    Utils.assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
                    Utils.assertEquals(iconService.getBalance(testWallet.getAddress()).execute()
                            , BigInteger.valueOf(needValue));
                }
                LOG.infoEntering("invoke");
                TransactionResult result = testScore.invokeAndWaitResult(testWallet, "hello",
                        null, BigInteger.valueOf(0), BigInteger.valueOf(step));
                LOG.infoExiting();
                if(step < testCCValue) {
                    Utils.assertEquals(Constants.STATUS_FAIL, result.getStatus());
                } else {
                    Utils.assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
                }
            } catch (ResultTimeoutException ex) {
                assertTrue(step < testCCValue);
            } catch (Exception ex) {
                fail();
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void notEnoughBalance() throws Exception {
        LOG.infoEntering( "notEnoughBalance");
        KeyWallet testWallet = KeyWallet.create();
        long needValue = testCCValue * testStepPrice;
        long []values = {needValue, needValue - 1};
        for(long value : values) {
            Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, testWallet.getAddress(), value);
            TransactionResult result = Utils.getTransactionResult(iconService, txHash, 5000);
            Utils.assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            Utils.assertEquals(iconService.getBalance(testWallet.getAddress()).execute()
                    , BigInteger.valueOf(value));
            try {
                result = testScore.invokeAndWaitResult(testWallet, "hello", null
                        , BigInteger.valueOf(0), BigInteger.valueOf(testCCValue));
                Utils.assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            } catch (ResultTimeoutException ex) {
                assertTrue(value < needValue);
            } catch (Exception ex) {
                fail();
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void callWithValue() throws Exception {
        LOG.infoEntering( "callWithValue");
        long needValue = testCCValue * testStepPrice ; // invoke & query
        final long testVal = 987;
        KeyWallet testWallet = KeyWallet.create();
        Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, testWallet.getAddress(), testVal + needValue);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, 5000);
        Utils.assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        assertEquals(iconService.getBalance(testWallet.getAddress()).execute()
                , BigInteger.valueOf(testVal + needValue));
        result = testScore.invokeAndWaitResult(testWallet, "transfer",
                null, BigInteger.valueOf(testVal), BigInteger.valueOf(testCCValue));
        Utils.assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);

        RpcObject params = new RpcObject.Builder()
                .put("_owner", new RpcValue(testWallet.getAddress()))
                .build();
        assertTrue(Utils.icxCall(iconService, testScore.getAddress(),
                "balanceOf",params).asInteger().equals(BigInteger.valueOf(testVal)));
        LOG.infoExiting();
    }

    public void invalidAddress() {
    }

}
