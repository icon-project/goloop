package foundation.icon.test.cases;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
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
import static org.junit.jupiter.api.Assertions.*;

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
    private static GovScore.Fee fee;
    private static Address scoreAddr;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        govScore = new GovScore(iconService, chain);
        fee = govScore.getFee();
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
        Address sAddr = Score.install(iconService, chain, ownerWallet, PATH, params);
        testScore = new Score(iconService, chain, sAddr);
        scoreAddr = sAddr;

        govScore.setMaxStepLimit("invoke", BigInteger.valueOf(1000));
        govScore.setMaxStepLimit("query", BigInteger.valueOf(1000));
        govScore.setStepCost("contractCall", BigInteger.valueOf(contractCallStep));
        govScore.setStepCost("default", BigInteger.valueOf(defaultStep));
        govScore.setStepPrice(BigInteger.valueOf(stepPrice));
    }

    @AfterAll
    public static void destroy() throws Exception {
        govScore.setFee(fee);
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
    public void timeoutCallInfiniteLoop() throws Exception {
        LOG.infoEntering( "timeout");
        LOG.infoEntering( "invoke");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "infiniteLoop",
                        null, BigInteger.valueOf(0), BigInteger.valueOf(100));
        assertEquals(Constants.STATUS_FAIL, result.getStatus());
        LOG.infoExiting();
        LOG.infoExiting();
    }

    @Test
    public void infiniteInterCall() throws Exception {
        LOG.infoEntering( "infiniteInterCall");
        LOG.infoEntering( "deploy 2 score with same source");
        Score[] scores = new Score[2];
        KeyWallet[] wallets = new KeyWallet[2];
        for(int i = 0; i < scores.length; i++) {
            RpcObject params = new RpcObject.Builder()
                    .put("name", new RpcValue("HelloWorld"))
                    .build();
            wallets[i] = ownerWallet;
            Address sAddr = Score.install(iconService, chain, wallets[i], PATH, params);
            scores[i] = new Score(iconService, chain, sAddr);
        }
        LOG.infoExiting();

        KeyWallet sender = KeyWallet.create();
        Utils.transferAndCheck(iconService, chain, chain.godWallet, sender.getAddress(), Constants.DEFAULT_BALANCE);

        BigInteger []limits = {Constants.DEFAULT_BALANCE, BigInteger.valueOf(10)};
        for(BigInteger l : limits) {
            LOG.infoEntering( "sendTransaction with (" + l + ") stepLimit");
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(scores[1].getAddress().toString()))
                    .put("call_cnt", new RpcValue(BigInteger.ZERO))
                    .build();
            TransactionResult result =
                    scores[0].invokeAndWaitResult(sender, "infinite_intercall",
                            params, BigInteger.valueOf(0), l);
            LOG.infoExiting();
            // Maximum recursion depth exceeded and OutOfStep are expected
            LOG.info("result : " + result);
            assertEquals(Constants.STATUS_FAIL, result.getStatus());
        }
        BigInteger bal = iconService.getBalance(sender.getAddress()).execute();
        LOG.info("sender's balance : " + bal);
        LOG.infoExiting();
    }

    @Test
    public void invalidSignature() throws Exception {
        LOG.infoEntering( "invalidSignature");
        KeyWallet []testWallets = new KeyWallet[10];
        for(int i = 0; i < testWallets.length; i++) {
            testWallets[i] = KeyWallet.create();
            Utils.transferAndCheck(iconService, chain, chain.godWallet, testWallets[i].getAddress(), BigInteger.ONE);
        }

        for(int i = 0; i < testWallets.length; i++) {
            KeyWallet wallet = testWallets[i];
            Transaction t = TransactionBuilder.newBuilder()
                    .nid(BigInteger.valueOf(chain.networkId))
                    .from(wallet.getAddress())
                    .to(scoreAddr)
                    .nonce(BigInteger.TEN)
                    .stepLimit(BigInteger.valueOf(10))
                    .call("transfer").build();

            try {
                iconService
                        .sendTransaction(new SignedTransaction(t, testWallets[0]))
                        .execute();
                assertEquals(0, i);
            }
            catch (RpcError ex) {
                assertNotEquals(0, i);
                continue;
            }
        }
        LOG.infoExiting();

    }
}
