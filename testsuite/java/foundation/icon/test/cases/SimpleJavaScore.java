package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.ScoreApi;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.util.List;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

@Tag(Constants.TAG_GOVERNANCE)
class SimpleJavaScore {
    private static IconService iconService;
    private static Env.Chain chain;
    private static KeyWallet ownerWallet;
    private static KeyWallet calleeWallet;
    private static GovScore govScore;
    private static GovScore.Fee fee;
    private static Score testScore;

    @BeforeAll
    static void init() throws Exception {
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
        calleeWallet = KeyWallet.create();
        Address[] addrs = {ownerWallet.getAddress(), calleeWallet.getAddress(), chain.governorWallet.getAddress()};
        Utils.transferAndCheck(iconService, chain, chain.godWallet, addrs, Constants.DEFAULT_BALANCE);

        govScore.setMaxStepLimit("invoke", BigInteger.valueOf(1000000));
        govScore.setMaxStepLimit("query", BigInteger.valueOf(1000000));
    }

    @AfterAll
    static void destroy() throws Exception {
        govScore.setFee(fee);
    }

    @Test
    void testCheckDefaultParam() throws Exception {
        Address scoreAddr = deploySampleToken(BigInteger.valueOf(18), BigInteger.valueOf(1000));

        LOG.infoEntering("checkDefaultParam");
        List<ScoreApi> apis = iconService.getScoreApi(scoreAddr).execute();
        for (ScoreApi api: apis) {
            if (api.getName().equals("transfer")) {
                for (ScoreApi.Param p : api.getInputs()) {
                    if (p.getName().equals("_data")) {
                        assertTrue(p.getDefault().isNull());
                    } else {
                        assertNull(p.getDefault());
                    }
                }
            }
        }
        LOG.infoExiting();
    }

    @Test
    void testSampleToken() throws Exception {
        // 1. deploy
        BigInteger decimals = BigInteger.valueOf(18);
        BigInteger initialSupply = BigInteger.valueOf(1000);
        Address scoreAddr = deploySampleToken(decimals, initialSupply);

        // 2. balanceOf
        LOG.infoEntering("balanceOf", "owner (initial)");
        BigInteger oneToken = BigInteger.TEN.pow(decimals.intValue());
        BigInteger totalSupply = oneToken.multiply(initialSupply);
        BigInteger bal = callBalanceOf(ownerWallet.getAddress()).asInteger();
        LOG.info("expected (" + totalSupply + "), result (" + bal + ")");
        assertEquals(totalSupply, bal);
        LOG.infoExiting();

        // 3. transfer #1
        LOG.infoEntering("transfer", "#1");
        TransactionResult result = invokeTransfer(scoreAddr, ownerWallet, calleeWallet.getAddress(), oneToken, true);
        LOG.info("result(" + result + ")");
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        LOG.infoExiting();

        // 3.1 transfer #2
        LOG.infoEntering("transfer", "#2");
        result = invokeTransfer(scoreAddr, ownerWallet, calleeWallet.getAddress(), oneToken, false);
        LOG.info("result(" + result + ")");
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        LOG.infoExiting();

        // 4. check balance of callee
        LOG.infoEntering("balanceOf", "callee");
        BigInteger expected = oneToken.add(oneToken);
        bal = callBalanceOf(calleeWallet.getAddress()).asInteger();
        LOG.info("expected (" + expected + "), result (" + bal + ")");
        assertEquals(expected, bal);
        LOG.infoExiting();

        // 5. check balance of owner
        LOG.infoEntering("balanceOf", "owner");
        expected = totalSupply.subtract(expected);
        bal = callBalanceOf(ownerWallet.getAddress()).asInteger();
        LOG.info("expected (" + expected + "), result (" + bal + ")");
        assertEquals(expected, bal);
        LOG.infoExiting();
    }

    private Address deploySampleToken(BigInteger decimals, BigInteger initialSupply) throws Exception {
        LOG.infoEntering("deploy", "SampleToken");
        RpcObject params = new RpcObject.Builder()
                .put("_name", new RpcValue("MySampleToken"))
                .put("_symbol", new RpcValue("MST"))
                .put("_decimals", new RpcValue(decimals))
                .put("_initialSupply", new RpcValue(initialSupply))
                .build();
        Address scoreAddr = Score.install(iconService, chain, ownerWallet, Constants.JSCORE_MYSAMPLETOKEN,
                                          params, 1000000, Constants.CONTENT_TYPE_JAVA);
        LOG.info("scoreAddr = " + scoreAddr);
        testScore = new Score(iconService, chain, scoreAddr);
        LOG.infoExiting();
        return scoreAddr;
    }

    private RpcItem callBalanceOf(Address addr) throws Exception {
        RpcObject params = new RpcObject.Builder()
                .put("_owner", new RpcValue(addr.toString()))
                .build();
        return testScore.call(KeyWallet.create().getAddress(), "balanceOf", params);
    }

    private TransactionResult invokeTransfer(Address score, Wallet from, Address to, BigInteger value,
                                             boolean includeData) throws Exception {
        RpcObject.Builder builder = new RpcObject.Builder()
                .put("_to", new RpcValue(to))
                .put("_value", new RpcValue(value));
        if (includeData) {
            builder.put("_data", new RpcValue("Hello".getBytes()));
        }
        return Utils.sendTransactionWithCall(iconService, chain.networkId,
                    from, score, "transfer", builder.build());
    }

    private Address deployAPITest() throws Exception {
        LOG.infoEntering("deploy", "apiTest");
        Address scoreAddr = Score.install(iconService, chain, ownerWallet, Constants.JSCORE_APITEST,
                                          null, 1000000, Constants.CONTENT_TYPE_JAVA);
        LOG.info("scoreAddr = " + scoreAddr);
        testScore = new Score(iconService, chain, scoreAddr);
        LOG.infoExiting();
        return scoreAddr;
    }

    @Test
    void testAPITestForAddress() throws Exception {
        Address scoreAddr = deployAPITest();
        KeyWallet caller = KeyWallet.create();
        TransactionResult tr;

        // getAddress
        LOG.infoEntering("getAddress", "invoke");
        tr = testScore.invokeAndWaitResult(caller, "getAddress",
                new RpcObject.Builder().put("addr", new RpcValue(testScore.getAddress())).build(),
                0, 100000);
        assertEquals(BigInteger.ONE, tr.getStatus());
        tr = testScore.invokeAndWaitResult(caller, "getAddress",
                new RpcObject.Builder().put("addr", new RpcValue(caller.getAddress())).build(),
                0, 100000);
        assertEquals(BigInteger.ZERO, tr.getStatus());
        LOG.infoExiting();

        LOG.infoEntering("getAddress", "query");
        RpcItem result = testScore.call(caller.getAddress(), "getAddressQuery", null);
        LOG.info("expected (" + scoreAddr + "), result (" + result.asAddress() + ")");
        assertEquals(scoreAddr, result.asAddress());
        LOG.infoExiting();

        // getCaller
        LOG.infoEntering("getCaller", "invoke");
        tr = testScore.invokeAndWaitResult(caller, "getCaller",
                new RpcObject.Builder().put("caller", new RpcValue(caller.getAddress())).build(),
                0, 100000);
        assertEquals(BigInteger.ONE, tr.getStatus());
        tr = testScore.invokeAndWaitResult(caller, "getCaller",
                new RpcObject.Builder().put("caller", new RpcValue(testScore.getAddress())).build(),
                0, 100000);
        assertEquals(BigInteger.ZERO, tr.getStatus());
        LOG.infoExiting();

        LOG.infoEntering("getCaller", "query");
        result = testScore.call(caller.getAddress(), "getCallerQuery", null);
        LOG.info("expected (" + "null" + "), result (" + result + ")");
        assertNull(result);
        LOG.infoExiting();

        // getOrigin
        LOG.infoEntering("getOrigin", "invoke");
        tr = testScore.invokeAndWaitResult(caller, "getOrigin",
                new RpcObject.Builder().put("origin", new RpcValue(caller.getAddress())).build(),
                0, 100000);
        assertEquals(BigInteger.ONE, tr.getStatus());
        tr = testScore.invokeAndWaitResult(caller, "getOrigin",
                new RpcObject.Builder().put("origin", new RpcValue(testScore.getAddress())).build(),
                0, 100000);
        assertEquals(BigInteger.ZERO, tr.getStatus());
        LOG.infoExiting();

        LOG.infoEntering("getOrigin", "query");
        result = testScore.call(caller.getAddress(), "getOriginQuery", null);
        LOG.info("expected (" + "null" + "), result (" + result + ")");
        assertNull(result);
        LOG.infoExiting();

        // getOwner
        LOG.infoEntering("getOwner", "invoke");
        tr = testScore.invokeAndWaitResult(caller, "getOwner",
                new RpcObject.Builder().put("owner", new RpcValue(ownerWallet.getAddress())).build(),
                0, 100000);
        assertEquals(BigInteger.ONE, tr.getStatus());
        tr = testScore.invokeAndWaitResult(caller, "getOwner",
                new RpcObject.Builder().put("owner", new RpcValue(caller.getAddress())).build(),
                0, 100000);
        assertEquals(BigInteger.ZERO, tr.getStatus());
        LOG.infoExiting();

        LOG.infoEntering("getOwner", "query");
        result = testScore.call(caller.getAddress(), "getOwnerQuery", null);
        LOG.info("expected (" + ownerWallet.getAddress() + "), result (" + result.asAddress() + ")");
        assertEquals(ownerWallet.getAddress(), result.asAddress());
        LOG.infoExiting();
    }

    @Test
    void testAPITestForBlock() throws Exception {
        Address scoreAddr = deployAPITest();
        KeyWallet caller = KeyWallet.create();
        TransactionResult tr;

        // getBlockHeight
        LOG.infoEntering("getBlockHeight", "invoke");
        tr = testScore.invokeAndWaitResult(caller, "getBlockHeight", null, 0, 100000);
        assertEquals(BigInteger.ONE, tr.getStatus());
        LOG.infoExiting();

        LOG.infoEntering("getBlockHeight", "query");
        RpcItem result = testScore.call(caller.getAddress(), "getBlockHeightQuery", null);
        LOG.info("expected (" + "?" + "), result (" + result.asInteger() + ")");
        assertTrue(BigInteger.ONE.compareTo(result.asInteger()) < 0);
        LOG.infoExiting();

        // getBlockTimestamp
        LOG.infoEntering("getBlockTimestamp", "invoke");
        BigInteger currentTimestamp = BigInteger.valueOf(System.currentTimeMillis());
        tr = testScore.invokeAndWaitResult(caller, "getBlockTimestamp", null, 0, 100000);
        assertEquals(BigInteger.ONE, tr.getStatus());
        LOG.infoExiting();

        LOG.infoEntering("getBlockTimestamp", "query");
        result = testScore.call(caller.getAddress(), "getBlockTimestampQuery", null);
        LOG.info("expected (" + currentTimestamp + "), result (" + result.asInteger() + ")");
        assertTrue(currentTimestamp.compareTo(result.asInteger()) < 0);
        LOG.infoExiting();
    }
}
