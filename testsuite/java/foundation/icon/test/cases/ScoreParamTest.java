package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.*;

@Tag(Constants.TAG_PY_SCORE)
class ScoreParamTest {
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

    private static final String[] VALUES_FOR_STR = {
            "hello", "ZERO", "ONE",
            "0x0", "0x1", "0x12", "0xdd",
            "true", "false", "",
    };

    private static final byte[][] VALUES_FOR_BYTES = {
            {0x22, 0x33, 0x7f},
            {0}, {1}, "Hello".getBytes(), {},
    };

    private static final BigInteger[] VALUES_FOR_INT = {
            BigInteger.ONE, BigInteger.ZERO,
            BigInteger.valueOf(0x1FFFFFFFFL),
            new BigInteger("1FFFFFFFFFFFFFFFF", 16),
    };

    private static final boolean[] VALUES_FOR_BOOL = {true, false};

    private static final Address[] VALUES_FOR_ADDRESS = {
            new Address("cxd2a525388459fab5f3107e230c9868d118b8d15d"),
            new Address("hxb37a4fc334b472e4b13d5d67087deaab9a85a324"),
            new Address("cx0000000000000000000000000000000000000000"),
            new Address("hx0000000000000000000000000000000000000000"),
    };

    // true if blockchain ignores undefined params
    // false if blockchain returns failure when undefined params passes

    @BeforeAll
    static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        initScoreTest();
    }

    private static void initScoreTest() throws Exception {
        ownerWallet = KeyWallet.create();
        callerWallet = KeyWallet.create();
        Address[] addrs = {ownerWallet.getAddress(), callerWallet.getAddress(), chain.governorWallet.getAddress()};
        Utils.transferAndCheck(iconService, chain, chain.godWallet, addrs, Constants.DEFAULT_BALANCE);

        RpcObject params = new RpcObject.Builder()
                .build();
        Address interCallAddr = Score.install(iconService, chain, ownerWallet, PATH, params);
        interCallScore = new Score(iconService, chain, interCallAddr);

        Address scoreAddr = Score.install(iconService, chain, ownerWallet, PATH, params);
        testScore = new Score(iconService, chain, scoreAddr);
    }

    @AfterAll
    static void destroy()  {
    }

    @Test
    void callInt() throws Exception {
        LOG.infoEntering("callInt");
        for (BigInteger p : VALUES_FOR_INT) {
            RpcObject params = new RpcObject.Builder()
                    .put("param", new RpcValue(p))
                    .build();
            LOG.infoEntering("invoke", p.toString());
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "call_int",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = testScore.call("check_int", null);
            assertEquals(p.toString(), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    void callStr() throws Exception {
        LOG.infoEntering("callStr");
        for (String p : VALUES_FOR_STR) {
            RpcObject params = new RpcObject.Builder()
                    .put("param", new RpcValue(p))
                    .build();
            LOG.infoEntering("invoke", p);
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "call_str",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = testScore.call("check_str", null);
            assertEquals(p, item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    void callBytes() throws Exception {
        LOG.infoEntering("callBytes");
        for (byte[] p : VALUES_FOR_BYTES) {
            RpcValue pv = new RpcValue(p);
            RpcObject params = new RpcObject.Builder()
                    .put("param", pv)
                    .build();

            LOG.infoEntering("invoke", pv.asString());
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "call_bytes",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = testScore.call("check_bytes", null);
            assertEquals(pv.asString(), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    void callBool() throws Exception {
        LOG.infoEntering("callBool");
        for (boolean p : VALUES_FOR_BOOL) {
            RpcObject params = new RpcObject.Builder()
                    .put("param", new RpcValue(p))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(p));
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "call_bool",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = testScore.call("check_bool", null);
            assertEquals(String.valueOf(p), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    void callAddress() throws Exception {
        LOG.infoEntering("callAddress");
        for (Address p : VALUES_FOR_ADDRESS) {
            RpcObject params = new RpcObject.Builder()
                    .put("param", new RpcValue(p))
                    .build();
            LOG.infoEntering("invoke", p.toString());
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "call_address",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = testScore.call("check_address", null);
            assertEquals(p.toString(), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    void callAll() throws Exception {
        LOG.infoEntering("callAll");
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
        RpcItem item = testScore.call("check_all", null);
        assertEquals("all", item.asString());
        LOG.infoExiting();
    }

    @Test
    void interCallBool() throws Exception {
        LOG.infoEntering("interCallBool");
        for (boolean p : new boolean[]{true, false}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(p))
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_BOOL)))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(p));
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_bool",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = interCallScore.call("check_bool", null);
            assertEquals(String.valueOf(p), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    void interCallAddress() throws Exception {
        LOG.infoEntering("interCallAddress");
        for (Address p : new Address[]{ownerWallet.getAddress(), callerWallet.getAddress()}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(p))
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_ADDRESS)))
                    .build();
            LOG.infoEntering("invoke", p.toString());
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_address",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = interCallScore.call("check_address", null);
            assertEquals(p.toString(), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    void interCallInt() throws Exception {
        LOG.infoEntering("interCallInt");
        for (BigInteger p : new BigInteger[]{
                    BigInteger.ZERO, BigInteger.ONE, BigInteger.valueOf(0x1FFFFFFFFL), new BigInteger("1FFFFFFFFFFFFFFFF", 16)
            }) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(p))
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_INT)))
                    .build();
            LOG.infoEntering("invoke", p.toString());
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_int",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = interCallScore.call("check_int", null);
            assertEquals(p.toString(), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    void interCallBytes() throws Exception {
        LOG.infoEntering("interCallBytes");
        for (byte[] p : new byte[][]{{0}, {1}, "Hello".getBytes(), {}}) {
            RpcValue pv = new RpcValue(p);
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", pv)
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_BYTES)))
                    .build();
            LOG.infoEntering("invoke", pv.asString());
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_bytes",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = interCallScore.call("check_bytes", null);
            assertEquals(pv.asString(), item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    void interCallStr() throws Exception {
        LOG.infoEntering("interCallStr");
        for (String p : VALUES_FOR_STR) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(p))
                    .put("ptype", new RpcValue(BigInteger.valueOf(TYPE_STR)))
                    .build();
            LOG.infoEntering("invoke", p);
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_str",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
            RpcItem item = interCallScore.call("check_str", null);
            assertEquals(p, item.asString());
        }
        LOG.infoExiting();
    }

    @Test
    void interCallAll() throws Exception {
        LOG.infoEntering("interCallAll");
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
        RpcItem item = interCallScore.call("check_all", null);
        assertEquals("all", item.asString());
        LOG.infoExiting();
    }

    @Test
    void invalidInterCallBool() throws Exception {
        LOG.infoEntering("invalidInterCallBool");
        for (int t : new int[]{TYPE_ADDRESS, TYPE_INT, TYPE_BYTES, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(true))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(t));
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_bool",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_FAIL, result.getStatus());
        }
        LOG.infoExiting();
    }

    @Test
    void invalidInterCallAddress() throws Exception {
        LOG.infoEntering("invalidInterCallAddress");
        for (int t : new int[]{TYPE_BOOL, TYPE_INT, TYPE_BYTES, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(KeyWallet.create().getAddress()))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(t));
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_address",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_FAIL, result.getStatus());
        }
        LOG.infoExiting();
    }

    @Test
    void invalidInterCallBytes() throws Exception {
        LOG.infoEntering("invalidInterCallBytes");
        for (int t : new int[]{TYPE_BOOL, TYPE_INT, TYPE_ADDRESS, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(new byte[]{10}))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(t));
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_bytes",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_FAIL, result.getStatus());
        }
        LOG.infoExiting();
    }

    @Test
    void invalidInterCallStr() throws Exception {
        LOG.infoEntering("invalidInterCallStr");
        for (int t : new int[]{TYPE_BOOL, TYPE_INT, TYPE_ADDRESS, TYPE_BYTES}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue("HI"))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(t));
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_str",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_FAIL, result.getStatus());
        }
        LOG.infoExiting();
    }

    @Test
    void invalidInterCallInt() throws Exception {
        LOG.infoEntering("invalidInterCallInt");
        for (int t : new int[]{TYPE_BOOL, TYPE_BYTES, TYPE_ADDRESS, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("param", new RpcValue(BigInteger.ONE))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(t));
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_int",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_FAIL, result.getStatus());
        }
        LOG.infoExiting();
    }

    @Test
    void callDefaultParam() throws Exception {
        LOG.infoEntering("callDefaultParam");
        String param = "Hello";
        RpcObject params = new RpcObject.Builder()
                .put("default_param", new RpcValue(param.getBytes()))
                .build();
        LOG.infoEntering("invoke", param);
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "call_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        RpcItem item = testScore.call("check_default", null);
        assertEquals(param, item.asString());

        params = new RpcObject.Builder()
                .build();
        LOG.infoEntering("invoke", "without param");
        result = testScore.invokeAndWaitResult(callerWallet, "call_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        item = testScore.call("check_default", null);
        assertEquals("None", item.asString());
        LOG.infoExiting();
    }

    @Test
    void interCallDefaultParam() throws Exception {
        LOG.infoEntering("interCallDefaultParam");
        String param = "Hello";
        RpcObject params = new RpcObject.Builder()
                .put("default_param", new RpcValue(param.getBytes()))
                .build();
        LOG.infoEntering("invoke", param);
        TransactionResult result =
                interCallScore.invokeAndWaitResult(callerWallet, "call_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        RpcItem item = interCallScore.call("check_default", null);
        assertEquals(param, item.asString());

        params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .build();
        LOG.infoEntering("invoke inter_call_default_param");
        result = testScore.invokeAndWaitResult(callerWallet, "inter_call_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        item = interCallScore.call("check_default", null);
        assertEquals("None", item.asString());
        LOG.infoExiting();
    }

    @Test
    void interCallWithNull() throws Exception {
        LOG.infoEntering("interCallWithNull");
        for (int t : new int[]{TYPE_BOOL, TYPE_ADDRESS, TYPE_INT, TYPE_BYTES, TYPE_STR}) {
            RpcObject params = new RpcObject.Builder()
                    .put("_to", new RpcValue(interCallScore.getAddress()))
                    .put("ptype", new RpcValue(BigInteger.valueOf(t)))
                    .build();
            LOG.infoEntering("invoke", String.valueOf(t));
            TransactionResult result =
                    testScore.invokeAndWaitResult(callerWallet, "inter_call_with_none",
                            params, BigInteger.valueOf(0), BigInteger.valueOf(100));
            LOG.infoExiting();
            assertEquals(Constants.STATUS_FAIL, result.getStatus());
        }
        LOG.infoExiting();
    }

    @Test
    void interCallWithMoreParams() throws Exception {
        LOG.infoEntering("interCallWithMore");
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
    void invalidAddUndefinedParam() throws Exception {
        LOG.infoEntering("invalidAddUndefinedParam");
        RpcObject params = new RpcObject.Builder()
                .put("undefined1", new RpcValue(true))
                .put("undefined2", new RpcValue(BigInteger.ONE))
                .build();
        LOG.infoEntering("invoke call_default_param");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "call_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        assertEquals(Constants.STATUS_FAIL, result.getStatus());
        LOG.infoExiting();
        LOG.infoExiting();
    }

    @Test
    void interCallWithEmptyString() throws Exception {
        LOG.infoEntering("interCallWithEmptyString");
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .build();
        LOG.infoEntering("invoke inter_call_empty_str");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "inter_call_empty_str",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        RpcItem item = interCallScore.call("check_str", null);
        assertEquals("", item.asString());
        LOG.infoExiting();
    }

    @Test
    void interCallWithDefaultParam() throws Exception {
        LOG.infoEntering("interCallWithDefaultParam");
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .build();
        LOG.infoEntering("invoke inter_call_with_default_param");
        TransactionResult result =
                testScore.invokeAndWaitResult(callerWallet, "inter_call_with_default_param",
                        params, BigInteger.valueOf(0), BigInteger.valueOf(100));
        LOG.infoExiting();
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        // check the saved values
        RpcItem item = interCallScore.call("check_bool", null);
        assertEquals("true", item.asString());
        item = interCallScore.call("check_address", null);
        assertEquals("cx0000000000000000000000000000000000000000", item.asString());
        item = interCallScore.call("check_int", null);
        assertEquals("0", item.asString());
        item = interCallScore.call("check_str", null);
        assertEquals("", item.asString());
        item = interCallScore.call("check_bytes", null);
        assertEquals("0x00", item.asString());
        LOG.infoExiting();
    }

    @Test
    void checkSender() throws Exception {
        LOG.infoEntering("checkSender");
        RpcItem item = testScore.call("check_sender", null);
        assertNull(item);
        LOG.infoExiting();
    }

    @Test
    void callAllDefault() throws Exception {
        final int NUM = 5;
        final int CASES = 1 << NUM;

        LOG.infoEntering("callAllDefault");
        String[] names = {"_bool", "_int", "_str", "_addr", "_bytes"};
        RpcValue[] values = {
                new RpcValue(VALUES_FOR_BOOL[0]),
                new RpcValue(VALUES_FOR_INT[0]),
                new RpcValue(VALUES_FOR_STR[0]),
                new RpcValue(VALUES_FOR_ADDRESS[0]),
                new RpcValue(VALUES_FOR_BYTES[0]),
        };

        LOG.infoEntering("sending transactions");
        Bytes[] ids = new Bytes[CASES];
        for (int i = 0; i < CASES; i++) {
            RpcObject.Builder pb = new RpcObject.Builder();
            for (int idx = 0; idx < NUM; idx++) {
                if ((i & (1 << idx)) != 0) {
                    pb.put(names[idx], values[idx]);
                }
            }
            RpcObject params = pb.build();
            LOG.info("case=" + i + " sending tx param=" + params.toString());
            ids[i] = testScore.invoke(
                    callerWallet, "call_all_default",
                    params, 0, Constants.DEFAULT_STEP_LIMIT);
            LOG.info("txid=" + ids[i].toString());
        }
        LOG.infoExiting();

        for (int i = 0; i < CASES; i++) {
            LOG.infoEntering("checking case=" + i + " txid=" + ids[i]);

            TransactionResult result = testScore.getResult(ids[i]);
            assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);

            boolean checked = false;
            for (TransactionResult.EventLog el : result.getEventLogs()) {
                RpcItem sig = el.getIndexed().get(0);
                if (!sig.asString().equals("LogCallValue(bool,int,str,Address,bytes)")) {
                    continue;
                }
                for (int idx = 0; idx < NUM; idx++) {
                    RpcItem val = el.getData().get(idx);
                    if ((i & (1 << idx)) != 0) {
                        assertEquals(values[idx].asString(), val.asString());
                    } else {
                        assertTrue(val.isNull());
                    }
                }
                checked = true;
            }
            assertTrue(checked);
            LOG.infoExiting();
        }

        LOG.infoExiting();
    }

    @Test
    void interCallWithLessParams() throws Exception {
        LOG.infoEntering("interCallWithLessParams");

        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(interCallScore.getAddress()))
                .put("_bool", new RpcValue(true))
                .put("_int", new RpcValue(BigInteger.TEN))
                .build();

        TransactionResult result = testScore.invokeAndWaitResult(
                callerWallet, "inter_call_with_less_params",
                params, 0, Constants.DEFAULT_STEP_LIMIT);

        assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);

        boolean checked = false;
        for (TransactionResult.EventLog el : result.getEventLogs()) {
            if (!interCallScore.getAddress().toString().equals(el.getScoreAddress())) {
                continue;
            }
            RpcItem sig = el.getIndexed().get(0);
            if (!sig.asString().equals("LogCallValue(bool,int,str,Address,bytes)")) {
                continue;
            }
            assertEquals("0x1", el.getData().get(0).asString());
            assertEquals("0xa", el.getData().get(1).asString());
            assertTrue(el.getData().get(2).isNull());
            assertTrue(el.getData().get(3).isNull());
            assertTrue(el.getData().get(4).isNull());
            checked = true;
        }
        assertTrue(checked);

        LOG.infoExiting();
    }
}
