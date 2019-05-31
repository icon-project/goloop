package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;

/*
test methods
    callInt
    callStr
    callBytes
    callBool
    callAddress
    callAll
    interCallBool
    interCallAddress
    interCallInt
    interCallBytes
    interCallStr
    interCallAll
    invalidInterCallBool
    invalidInterCallAddress
    invalidInterCallBytes
    invalidInterCallStr
    invalidInterCallInt
    callDefaultParam
    interCallDefaultParam
    interCallWithNull
    interCallWithMoreParams
    invalidAddUndefinedParam
    interCallWithEmptyString
    interCallWithDefaultParam
 */
@Tag(Constants.TAG_NORMAL)
public class ScoreParamTest {
    private static IconService iconService;
    private static Env.Chain chain;
    private static KeyWallet ownerWallet;
    private static KeyWallet callerWallet;
    private static Score testScore;
    private static Score interCallScore;
    private static final String PATH = Constants.SCORE_CHECKPARAMS_PATH;

    private static final int TYPE_BOOL = 0;
    private static final int TYPE_ADDRESS = 1;
    private static final int TYPE_INT = 2;
    private static final int TYPE_BYTES = 3;
    private static final int TYPE_STR = 4;

    // true if blockchain ignores undefined params
    // false if blockhain returns failure when undefined params passes
    private static final boolean IGNORE_ADDITIONAL_PARAM = true;

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
        Address[]addrs = {ownerWallet.getAddress(), callerWallet.getAddress(), chain.governorWallet.getAddress()};
        Utils.transferAndCheck(iconService, chain, chain.godWallet, addrs, Constants.DEFAULT_BALANCE);

        RpcObject params = new RpcObject.Builder()
                .build();
        Address interCallAddr = Score.install(iconService, chain, ownerWallet, PATH, params);
        interCallScore = new Score(iconService, chain, interCallAddr);

        Address scoreAddr = Score.install(iconService, chain, ownerWallet, PATH, params);
        testScore = new Score(iconService, chain, scoreAddr);
    }

    @AfterAll
    public static void destroy()  {
    }

    @Test
    public void callInt() throws Exception {
        LOG.infoEntering( "callInt");
        for(BigInteger p : new BigInteger[]{
                BigInteger.ZERO, BigInteger.ONE, BigInteger.valueOf(0x1FFFFFFFFL), new BigInteger("1FFFFFFFFFFFFFFFF", 16)
        }) {
            RpcObject params = new RpcObject.Builder()
                    .put("param", new RpcValue(p))
                    .build();
            LOG.infoEntering("invoke call_int");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "call_int",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item =
                    testScore.call(callerWallet.getAddress(), "check_int", null);
            assertEquals(p.toString(), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    public void callStr() throws Exception {
        LOG.infoEntering( "callStr");
        for(String p : new String[]{"0", "1", "ZERO", "ONE", "100000000000000000000000000000"}) {
            RpcObject params = new RpcObject.Builder()
                    .put("param", new RpcValue(p))
                    .build();
            LOG.infoEntering("invoke call_str");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "call_str",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = testScore.call(callerWallet.getAddress(), "check_str", null);
            assertEquals(p, item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    public void callBytes() throws Exception {
        LOG.infoEntering( "callBytes");
        for(byte[] p : new byte[][]{{0}, {1}}) {
            RpcObject params = new RpcObject.Builder()
                    .put("param", new RpcValue(p))
                    .build();
            LOG.infoEntering("invoke call_bytes");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "call_bytes",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item =
                    testScore.call(callerWallet.getAddress(), "check_bytes", null);
            assertEquals(String.valueOf(p[0]), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    public void callBool() throws Exception {
        LOG.infoEntering( "callBool");
        for(boolean p : new boolean[]{true, false}) {
            RpcObject params = new RpcObject.Builder()
                    .put("param", new RpcValue(p))
                    .build();
            LOG.infoEntering("invoke call_bool");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "call_bool",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item =
                    testScore.call(callerWallet.getAddress(), "check_bool", null);
            LOG.info("item : " + item);
            assertEquals(String.valueOf(p), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    public void callAddress() throws Exception {
        LOG.infoEntering( "callAddress");
        for(Address p : new Address[]{KeyWallet.create().getAddress(), KeyWallet.create().getAddress()}) {
            RpcObject params = new RpcObject.Builder()
                    .put("param", new RpcValue(p))
                    .build();
            LOG.infoEntering("invoke call_address");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "call_address",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item =
                    testScore.call(callerWallet.getAddress(), "check_address", null);
            assertEquals(p.toString(), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    public void callAll() throws Exception {
        LOG.infoEntering( "callAll");
        RpcObject params = new RpcObject.Builder()
                .put("p_bool", new RpcValue(true))
                .put("p_addr", new RpcValue(KeyWallet.create().getAddress()))
                .put("p_int", new RpcValue(BigInteger.ONE))
                .put("p_str", new RpcValue("HELLO"))
                .put("p_bytes", new RpcValue(new byte[]{0x12}))
                .build();
        LOG.infoEntering("invoke call_all");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "call_all",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        RpcItem item =
                testScore.call(callerWallet.getAddress(), "check_all", null);
        assertEquals("all", item.asString());
        LOG.infoExiting();
    }


    @Test
    public void interCallBool() throws Exception {
        LOG.infoEntering( "interCallBool");
        for(boolean p : new boolean[]{true, false}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(p))
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_BOOL)))
                    .build();
            LOG.infoEntering("invoke inter_call_bool");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_bool",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = interCallScore.call(callerWallet.getAddress(), "check_bool", null);
            assertEquals(String.valueOf(p), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    public void interCallAddress() throws Exception {
        LOG.infoEntering( "interCallAddress");
        for(Address p : new Address[]{KeyWallet.create().getAddress(), KeyWallet.create().getAddress()}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(p))
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_ADDRESS)))
                    .build();
            LOG.infoEntering("invoke inter_call_address");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_address",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = interCallScore.call(callerWallet.getAddress(), "check_address", null);
            assertEquals(p.toString(), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    public void interCallInt() throws Exception {
        LOG.infoEntering( "interCallInt");
        for(BigInteger p : new BigInteger[]{
                    BigInteger.ZERO, BigInteger.ONE, BigInteger.valueOf(0x1FFFFFFFFL), new BigInteger("1FFFFFFFFFFFFFFFF", 16)
            }) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(p))
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_INT)))
                    .build();
            LOG.infoEntering("invoke inter_call_int");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_int",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = interCallScore.call(callerWallet.getAddress(), "check_int", null);
            assertEquals(p.toString(), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    public void interCallBytes() throws Exception {
        LOG.infoEntering( "interCallBytes");
        for(byte[] p : new byte[][]{{0}, {1}}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(p))
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_BYTES)))
                    .build();
            LOG.infoEntering("invoke inter_call_bytes");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_bytes",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = interCallScore.call(callerWallet.getAddress(), "check_bytes", null);
            assertEquals(String.valueOf(p[0]), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    public void interCallStr() throws Exception {
        LOG.infoEntering( "interCallStr");
        for(String p : new String[]{"0", "1", "ZERO", "ONE", "100000000000000000000000000000"}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(p))
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_STR)))
                    .build();

            LOG.infoEntering("invoke inter_call_str");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_str",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = interCallScore.call(callerWallet.getAddress(), "check_str", null);
            assertEquals(p, item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    public void interCallAll() throws Exception {
        LOG.infoEntering( "interCallAll");
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .put("p_bool", new RpcValue(true))
                .put("p_addr", new RpcValue(KeyWallet.create().getAddress()))
                .put("p_int", new RpcValue(BigInteger.ONE))
                .put("p_str", new RpcValue("HELLO"))
                .put("p_bytes", new RpcValue(new byte[]{0x12}))
                .build();

        LOG.infoEntering("invoke inter_call_all");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "inter_call_all",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        RpcItem item = interCallScore.call(callerWallet.getAddress(), "check_all", null);
        assertEquals("all", item.asString());
        LOG.infoExiting();
    }

    @Test
    public void invalidInterCallBool() throws Exception {
        interCallBool();
        LOG.infoEntering( "invalidInterCallBool");
        for(int t : new int[]{TYPE_ADDRESS, TYPE_INT, TYPE_BYTES, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(true))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke inter_call_bool");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_bool",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_FAIL, result.getStatus());
        }
        LOG.infoExiting();
    }

    @Test
    public void invalidInterCallAddress() throws Exception {
        interCallAddress();
        LOG.infoEntering( "invalidInterCallAddress");
        for(int t : new int[]{TYPE_BOOL, TYPE_INT, TYPE_BYTES, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(KeyWallet.create().getAddress()))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke inter_call_address");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_address",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_FAIL, result.getStatus());
        }
        LOG.infoExiting();
    }

    @Test
    public void invalidInterCallBytes() throws Exception {
        interCallBytes();
        LOG.infoEntering( "invalidInterCallBytes");
        for(int t : new int[]{TYPE_BOOL, TYPE_INT, TYPE_ADDRESS, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(new byte[]{10}))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke inter_call_bytes");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_bytes",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_FAIL, result.getStatus());
        }
        LOG.infoExiting();
    }

    @Test
    public void invalidInterCallStr() throws Exception {
        interCallStr();
        LOG.infoEntering( "invalidInterCallStr");
        for(int t : new int[]{TYPE_BOOL, TYPE_INT, TYPE_ADDRESS, TYPE_BYTES}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue("HI"))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke inter_call_str");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_str",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_FAIL, result.getStatus());
        }
        LOG.infoExiting();
    }

    @Test
    public void invalidInterCallInt() throws Exception {
        interCallInt();
        LOG.infoEntering( "invalidInterCallInt");
        for(int t : new int[]{TYPE_BOOL, TYPE_BYTES, TYPE_ADDRESS, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(BigInteger.ONE))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke inter_call_int");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_int",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_FAIL, result.getStatus());
        }
        LOG.infoExiting();
    }

    @Test
    public void callDefaultParam() throws Exception {
        LOG.infoEntering( "callDefaultParam");
        RpcObject params = new RpcObject.Builder()
                .put("default_param", new RpcValue(new byte[]{0x10}))
                .build();
        LOG.infoEntering("invoke call_default_param with param");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "call_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        RpcItem item =
                testScore.call(callerWallet.getAddress(), "check_default", null);
        assertEquals("default", item.asString());

        params = new RpcObject.Builder()
                .build();
        LOG.infoEntering("invoke call_default_param with no param");
        result =
                testScore.invokeAndWaitResult(callerWallet, "call_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        item = testScore.call(callerWallet.getAddress(), "check_default", null);
        assertEquals("None", item.asString());
        LOG.infoExiting();
    }

    @Test
    public void interCallDefaultParam() throws Exception {
        LOG.infoEntering( "interCallDefaultParam");
        RpcObject params = new RpcObject.Builder()
                .put("default_param", new RpcValue(new byte[]{0x10}))
                .build();
        LOG.infoEntering("invoke call_default_param with param");
        TransactionResult result =
                interCallScore.invokeAndWaitResult(callerWallet, "call_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        RpcItem item =
                interCallScore.call(callerWallet.getAddress(), "check_default", null);
        assertEquals("default", item.asString());

        params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .build();
        LOG.infoEntering("invoke inter_call_default_param");
        result = testScore.invokeAndWaitResult(callerWallet, "inter_call_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        item = interCallScore.call(callerWallet.getAddress(), "check_default", null);
        assertEquals("None", item.asString());
        LOG.infoExiting();
    }

    @Test
    public void interCallWithNull() throws Exception {
        LOG.infoEntering( "interCallWithNull");
        for(int t : new int[]{TYPE_BOOL, TYPE_ADDRESS, TYPE_INT, TYPE_BYTES, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke inter_call_with_none");
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_with_none",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            if(t == TYPE_ADDRESS || t == TYPE_BYTES) {
                assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
                String method = null;
                if(t == TYPE_ADDRESS) {
                    method = "check_address";
                }else {
                    method = "check_bytes";
                }
                RpcItem item =
                        interCallScore.call(callerWallet.getAddress(), method, null);
                assertEquals("None", item.asString());
            } else {
                assertEquals(Constants.STATUS_FAIL, result.getStatus());
            }
        }
        LOG.infoExiting();
    }

    @Test
    public void interCallWithMoreParams() throws Exception {
        LOG.infoEntering( "interCallWithMore");
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .build();
        LOG.infoEntering("invoke inter_call_with_more_params");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "inter_call_with_more_params",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_FAIL, result.getStatus());
        LOG.infoExiting();
    }

    @Test
    public void invalidAddUndefinedParam() throws Exception {
        LOG.infoEntering( "invalidAddUndefinedParam");
        RpcObject params = new RpcObject.Builder()
                .put("undefined1", new RpcValue(true))
                .put("undefined2", new RpcValue(BigInteger.ONE))
                .build();
        LOG.infoEntering("invoke call_default_param");
        boolean timeout = false;
        TransactionResult result = null;
        try {
            result = testScore.invokeAndWaitResult(callerWallet, "call_default_param",
                    params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        }
        catch (ResultTimeoutException ex) {
            timeout = true;
        }
        if(!IGNORE_ADDITIONAL_PARAM) {
            assertEquals(true, timeout);
        } else {
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        }
        LOG.infoExiting();
        LOG.infoExiting();
    }

    @Test
    public void interCallWithEmptyString() throws Exception {
        LOG.infoEntering( "interCallWithEmptyString");
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .build();
        LOG.infoEntering("invoke inter_call_empty_str");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "inter_call_empty_str",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        LOG.infoExiting();
    }

    @Test
    public void interCallWithDefaultParam() throws Exception {
        LOG.infoEntering( "interCallWithDefaultParam");
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .build();
        LOG.infoEntering("invoke inter_call_with_default_param");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "inter_call_with_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        LOG.infoExiting();
    }
}
