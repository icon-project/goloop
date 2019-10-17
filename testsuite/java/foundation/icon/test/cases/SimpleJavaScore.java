package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.Wallet;
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

// TODO SKip if audit enabled
@Tag(Constants.TAG_GOVERNANCE)
public class SimpleJavaScore {
    private static IconService iconService;
    private static Env.Chain chain;
    private static KeyWallet ownerWallet;
    private static KeyWallet callerWallet;
    private static GovScore govScore;
    private static Score testScore;
    private static final String PATH = Constants.JSCORE_MYSAMPLETOKEN;
    private static final long contractCallStep = 10;
    private static final long defaultStep = 2;
    private static final long stepPrice = 1;
    private static GovScore.Fee fee;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        govScore = new GovScore(iconService, chain);
        Thread.sleep(10000);
        fee = govScore.getFee();
        initScoreTest();
    }

    private static void initScoreTest() throws Exception {
        ownerWallet = KeyWallet.create();
        callerWallet = KeyWallet.create();
        Address []addrs = {ownerWallet.getAddress(), callerWallet.getAddress(), chain.governorWallet.getAddress()};
        Utils.transferAndCheck(iconService, chain, chain.godWallet, addrs, Constants.DEFAULT_BALANCE);

        govScore.setMaxStepLimit("invoke", BigInteger.valueOf(1000000));
        govScore.setMaxStepLimit("query", BigInteger.valueOf(1000000));
        govScore.setStepCost("contractCall", BigInteger.valueOf(contractCallStep));
        govScore.setStepCost("default", BigInteger.valueOf(defaultStep));
        govScore.setStepPrice(BigInteger.valueOf(stepPrice));
    }

    @AfterAll
    public static void destroy() throws Exception {
        govScore.setFee(fee);
    }

    @Test
    public void tokenSample() throws Exception {
        // TODO
        // 1. deploy
        LOG.infoEntering("deploy");
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("MySampleToken"))
                .put("symbol", new RpcValue("MySampleToken"))
                .put("decimals", new RpcValue("0x9"))
                .put("initialSupply", new RpcValue("0x3E8"))
                .build();

        Address scoreAddr = Score.install(iconService, chain, ownerWallet, PATH, params, 10000000, Constants.CONTENT_TYPE_JAVA);
        LOG.info("scoreAddr " + scoreAddr);
        testScore = new Score(iconService, chain, scoreAddr);
        LOG.infoExiting();

        // 2. getBalanceOf
        LOG.infoEntering("getBalance");
        BigInteger initialSupply = BigInteger.valueOf(0x3e8).pow(4);
        BigInteger bal = callBalanceOf(ownerWallet.getAddress()).asInteger();
        LOG.info("expected (" + initialSupply + "result (" + bal + ")");
        assertTrue(initialSupply.equals(bal));
        LOG.infoExiting();

        // 3. transfer
        LOG.infoEntering("transfer");
        BigInteger val = BigInteger.ONE;
        TransactionResult result = invokeTransfer(scoreAddr, ownerWallet, callerWallet.getAddress(), val);
        LOG.info("result(" + result + ")");
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        LOG.infoExiting();

        // 4. getBalanceOf - check balance of caller
        LOG.infoEntering("getBalanceOf caller");
        bal = callBalanceOf(callerWallet.getAddress()).asInteger();
        LOG.info("expected (" + val + "), result (" + bal + ")");
        assertTrue(val.equals(bal));
        LOG.infoExiting();

        // 5. getBalanceOf - check balance of User
        LOG.infoEntering("getBalanceOf caller");
        bal = callBalanceOf(ownerWallet.getAddress()).asInteger();
        LOG.info("expected (" + initialSupply.subtract(val) + "), result (" + bal + ")");
        assertTrue(initialSupply.subtract(val).equals(bal));
        LOG.infoExiting();
    }

    private RpcItem callBalanceOf(Address addr) throws Exception {
        RpcObject.Builder builder = new RpcObject.Builder();
        builder.put("_owner", new RpcValue(addr.toString()));
        return testScore.call(KeyWallet.create().getAddress(), "balanceOf", builder.build());
    }

    private TransactionResult invokeTransfer(Address score, Wallet from, Address to, BigInteger value) throws Exception {
        //address _to, integer _value, byte
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(to))
                .put("_value", new RpcValue(value))
                .put("_data", new RpcValue(new Bytes(BigInteger.ONE)))
                .build();
        return Utils.sendTransactionWithCall(iconService, chain.networkId,
                    from, score, "transfer", params);
    }
}
