package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

/*
test methods
  positive
    callWithValue
  negative
    invalidParamName
    notEnoughStepLimit
    notEnoughBalance
    timeout
 */
@Tag(Constants.TAG_GOVERNANCE)
public class ScoreTest {
    private static IconService iconService;
    private static Env.Chain chain;
    private static KeyWallet ownerWallet;
    private static KeyWallet callerWallet;
    private static GovScore govScore;
    private static Score testScore;
    private static final String PATH = Constants.SCORE_HELLOWORLD_PATH;
    private static final long contractCallStep = 10;
    private static final long defaultStep = 2;
    private static final long stepPrice = 1;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        initScoreTest();
    }

    private static void initScoreTest() throws Exception {
        ownerWallet = KeyWallet.create();
        callerWallet = KeyWallet.create();
        Address []addrs = {ownerWallet.getAddress(), callerWallet.getAddress(), chain.governorWallet.getAddress()};
        Utils.transferAndCheck(iconService, chain, chain.godWallet, addrs, Constants.DEFAULT_BALANCE);

        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        Address scoreAddr = Score.install(iconService, chain, ownerWallet, PATH, params);
        testScore = new Score(iconService, chain, scoreAddr);

        govScore = new GovScore(iconService, chain);
        govScore.setMaxStepLimit("invoke", BigInteger.valueOf(1000));
        govScore.setMaxStepLimit("query", BigInteger.valueOf(1000));
        govScore.setStepCost("contractCall", BigInteger.valueOf(contractCallStep));
        govScore.setStepCost("default", BigInteger.valueOf(defaultStep));
        govScore.setStepPrice(BigInteger.valueOf(stepPrice));
    }

    @AfterAll
    public static void destroy() throws Exception {
        // TODO set initial value not 0
        govScore.setStepCost("contractCall", BigInteger.valueOf(0));
        govScore.setStepPrice(BigInteger.valueOf(0));
        govScore.setMaxStepLimit("invoke", BigInteger.valueOf(1000));
    }

    @Test
    public void invalidParamName() throws Exception {
        LOG.infoEntering( "invalidParamName");
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

    public void invalidParamNum() throws Exception {
        LOG.infoEntering( "invalidParamNum");
        for(String []params : new String[][]{{}, {"name"}, {"name", "age"}}) {
            try {
                RpcObject.Builder builder = new RpcObject.Builder();
                for(String param: params){
                        builder.put(param, new RpcValue("ICONLOOP"));
                }
                RpcObject objParam = builder.build();
                LOG.info("invoke param[" + params + "]");
                TransactionResult result = testScore.invokeAndWaitResult(callerWallet,
                        "helloWithName", objParam, BigInteger.valueOf(0), BigInteger.valueOf(100));
                assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
                assertTrue(params.length == 1);
                LOG.infoExiting();
            } catch (ResultTimeoutException ex) {
                assertTrue(params.length != 1);
                LOG.infoExiting();
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void notEnoughStepLimit() throws Exception{
        LOG.infoEntering( "notEnoughStepLimit");
        KeyWallet testWallet = KeyWallet.create();
        long needStep = contractCallStep + defaultStep;
        long needValue = needStep * stepPrice;
        long preValidationFailureStep = defaultStep - 1;
        // expected {prevalidation failure, transaction execution failre, transaction execution success}
        for(long step : new long[]{preValidationFailureStep, needStep - 1, needStep}) {
            try {
                BigInteger sub = BigInteger.valueOf(needValue).subtract(iconService.getBalance(testWallet.getAddress()).execute());
                if(sub.compareTo(BigInteger.ZERO) > 0) {
                    Utils.transferAndCheck(iconService, chain, chain.godWallet, testWallet.getAddress(), sub);
                }
                LOG.infoEntering("invoke");
                TransactionResult result = testScore.invokeAndWaitResult(testWallet, "hello",
                        null, BigInteger.valueOf(0), BigInteger.valueOf(step));
                LOG.infoExiting();
                if(step < needStep) {
                    assertEquals(Constants.STATUS_FAIL, result.getStatus());
                } else {
                    assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
                }
            } catch (ResultTimeoutException ex) {
                LOG.infoExiting();
                assertTrue(preValidationFailureStep == step);
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void notEnoughBalance() throws Exception {
        LOG.infoEntering( "notEnoughBalance");
        KeyWallet testWallet = KeyWallet.create();
        long needStep = contractCallStep + defaultStep;
        long needValue = needStep * stepPrice;
        long []values = {needValue, needValue - 1};
        for(long value : values) {
            Utils.transferAndCheck(iconService, chain, chain.godWallet, testWallet.getAddress(), BigInteger.valueOf(value));
            try {
                TransactionResult result = testScore.invokeAndWaitResult(testWallet, "hello", null
                        , BigInteger.valueOf(0), BigInteger.valueOf(needStep));
                assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            } catch (ResultTimeoutException ex) {
                assertTrue(value < needValue);
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void callWithValue() throws Exception {
        LOG.infoEntering( "callWithValue");
        long needStep = contractCallStep + defaultStep;
        long needValue = needStep * stepPrice;
        final long testVal = 10;
        KeyWallet testWallet;
        BigInteger expectedBal;
        do{
            testWallet = KeyWallet.create();
            expectedBal = iconService.getBalance(testWallet.getAddress()).execute();
        } while(expectedBal.signum() != 0);

        Utils.transferAndCheck(iconService, chain, chain.godWallet, testWallet.getAddress(), BigInteger.valueOf(testVal + needValue));
        TransactionResult result = testScore.invokeAndWaitResult(testWallet, "transfer",
                null, BigInteger.valueOf(testVal), BigInteger.valueOf(needStep));
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());

        RpcObject params = new RpcObject.Builder()
                .put("_owner", new RpcValue(testWallet.getAddress()))
                .build();
        expectedBal = Utils.icxCall(iconService, testScore.getAddress(), "balanceOf",params).asInteger();
        assertEquals(BigInteger.valueOf(testVal), expectedBal);
        assertEquals(BigInteger.ZERO, iconService.getBalance(testWallet.getAddress()).execute());
        LOG.infoExiting();
    }

    @Test
    public void timeout() throws Exception {
        LOG.infoEntering( "timeout");
        LOG.infoEntering( "invoke");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "infiniteLoop",
                        null, BigInteger.valueOf(0), BigInteger.valueOf(100));
        assertEquals(Constants.STATUS_FAIL, result.getStatus());
        LOG.infoExiting();
        LOG.infoExiting();
    }

}
