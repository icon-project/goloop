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
        Env.Node node = Env.getInstance().nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        ownerWallet = KeyWallet.create();
        initScoreTest();
    }

    private static void initScoreTest() throws Exception {
        Address []addrs = {ownerWallet.getAddress(), chain.governorWallet.getAddress()};
        for(Address addr : addrs){
            Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, addr, 90000000);
            try {
                Utils.getTransactionResult(iconService, txHash, 5000);
            }
            catch (ResultTimeoutException ex) {
                System.out.println("Failed to transfer");
                throw ex;
            }
        }

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
        String []testParams = {"name", "nami"};
        for(String param : testParams) {
            try {
                RpcObject params = new RpcObject.Builder()
                        .put(param, new RpcValue("ICONLOOP"))
                        .build();
                TransactionResult result = testScore.invokeAndWaitResult(chain.godWallet, "helloWithName", params
                        , BigInteger.valueOf(0), BigInteger.valueOf(100));
                assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);
            } catch (ResultTimeoutException ex) {
                assertTrue(!param.equals("name"));
            } catch (Exception ex) {
                fail();
            }
        }
    }

    @Ignore
    @Test
    public void invalidParamNum() {
        String [][]testParams = {
                {},
                {"name"},
                {"name", "age"},
        };
        for(String []params : testParams) {
            try {
                RpcObject.Builder builder = new RpcObject.Builder();
                for(String param: params){
                        builder.put(param, new RpcValue("ICONLOOP"));
                }
                RpcObject objParam = builder.build();
                TransactionResult result = testScore.invokeAndWaitResult(chain.godWallet,
                        "helloWithName", objParam, BigInteger.valueOf(0), BigInteger.valueOf(100));
                assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);
                assertTrue(params.length == 1);
            } catch (ResultTimeoutException ex) {
                System.out.println("ResultTimeoutException. paramLen : " + params.length);
                assertTrue(params.length != 1);
            } catch (Exception ex) {
                fail();
            }
        }
    }

    @Test
    public void notEnoughStepLimit() throws Exception{
        KeyWallet testWallet = KeyWallet.create();
        long needValue = testCCValue * testStepPrice;

        long []testSteps = {testCCValue - 1, testCCValue};
        for(long step : testSteps) {
            try {
                BigInteger sub = BigInteger.valueOf(needValue).subtract(iconService.getBalance(testWallet.getAddress()).execute());
                if(sub.compareTo(BigInteger.ZERO) > 0) {
                    Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, testWallet.getAddress(), sub.longValue());
                    try {
                        Utils.getTransactionResult(iconService, txHash, 5000);
                    }
                    catch (Exception ex) {
                        fail();
                    }
                    assertEquals(iconService.getBalance(testWallet.getAddress()).execute()
                            , BigInteger.valueOf(needValue));
                }
                TransactionResult result = testScore.invokeAndWaitResult(testWallet, "hello",
                        null, BigInteger.valueOf(0), BigInteger.valueOf(step));
                if(step < testCCValue) {
                    assertTrue(result.getStatus().equals(Constants.STATUS_FAIL));
                } else {
                    assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);
                }
            } catch (ResultTimeoutException ex) {
                assertTrue(step < testCCValue);
            } catch (Exception ex) {
                fail();
            }
        }
    }

    @Test
    public void notEnoughBalance() throws Exception {
        KeyWallet testWallet = KeyWallet.create();
        long needValue = testCCValue * testStepPrice;
        long []values = {needValue, needValue - 1};
        for(long value : values) {
            Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, testWallet.getAddress(), value);
            try {
                Utils.getTransactionResult(iconService, txHash, 5000);
            }
            catch (Exception ex) {
                fail();
            }
            assertEquals(iconService.getBalance(testWallet.getAddress()).execute()
                    , BigInteger.valueOf(value));
            try {
                TransactionResult result = testScore.invokeAndWaitResult(testWallet, "hello", null
                        , BigInteger.valueOf(0), BigInteger.valueOf(testCCValue));
                assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);
            } catch (ResultTimeoutException ex) {
                assertTrue(value < needValue);
            } catch (Exception ex) {
                fail();
            }
        }
    }

    @Test
    public void callWithValue() throws Exception {
        long needValue = testCCValue * testStepPrice ; // invoke & query
        final long testVal = 987;
        KeyWallet testWallet = KeyWallet.create();
        Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, testWallet.getAddress(), testVal + needValue);
        try {
            Utils.getTransactionResult(iconService, txHash, 5000);
        }
        catch (Exception ex) {
            fail();
        }
        assertEquals(iconService.getBalance(testWallet.getAddress()).execute()
                , BigInteger.valueOf(testVal + needValue));
        try {
            TransactionResult result = testScore.invokeAndWaitResult(testWallet, "transfer",
                    null, BigInteger.valueOf(testVal), BigInteger.valueOf(testCCValue));
            assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);

            RpcObject params = new RpcObject.Builder()
                    .put("_owner", new RpcValue(testWallet.getAddress()))
                    .build();
            assertTrue(Utils.icxCall(iconService, testScore.getAddress(),
                    "balanceOf",params).asInteger().equals(BigInteger.valueOf(testVal)));
        } catch (ResultTimeoutException ex) {
            fail();
        } catch (Exception ex) {
            fail();
        }
    }

    public void invalidAddress() {
    }

}
