package foundation.icon.test.cases;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.*;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.score.HelloWorld;
import org.junit.AfterClass;
import org.junit.BeforeClass;
import org.junit.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static junit.framework.TestCase.fail;
import static org.junit.Assert.*;

/*
test cases
1. audit
2. not enough balance for deploy.
 - setStepPrice
3. not enough stepLimit for deploy.
4. content
 - no root file.
 - not zip
 - too large - takes too long time for uncompress 5. sendTransaction with invalid/valid params 6. sendTransaction for update with invalid score address 7. change destination url.
8. sendTransaction with invalid signature
 */
public class DeployTest {
    private static IconService iconService;
    private static Env.Chain chain;
    private static GovScore govScore;
    private static final BigInteger stepCostCC = BigInteger.valueOf(10);
    private static final BigInteger stepPrice = BigInteger.valueOf(10);
    private static final BigInteger invokeMaxStepLimit = BigInteger.valueOf(100000);
    private static BigInteger defStepCostCC;
    private static BigInteger defMaxStepLimit;
    private static BigInteger defStepPrice;

    @BeforeClass
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        govScore = new GovScore(iconService, chain);
        initDeploy();
    }

    public static void initDeploy() throws Exception {
        RpcObject params = new RpcObject.Builder()
                .put("contextType", new RpcValue("invoke"))
                .build();
        defMaxStepLimit = Utils.icxCall(iconService, Constants.CHAINSCORE_ADDRESS,
                "getMaxStepLimit", params).asInteger();


        defStepCostCC = Utils.icxCall(iconService, Constants.CHAINSCORE_ADDRESS,
                "getStepCosts", null).asObject().getItem("contractCreate").asInteger();

        defStepPrice = Utils.icxCall(iconService, Constants.CHAINSCORE_ADDRESS,
                "getStepPrice", null).asInteger();

        Utils.transferAndCheck(iconService, chain, chain.godWallet, chain.governorWallet.getAddress(), Constants.DEFAULT_BALANCE);
        govScore.setMaxStepLimit("invoke", invokeMaxStepLimit);
        govScore.setStepCost("contractCreate", stepCostCC);
        govScore.setStepPrice(stepPrice);
    }

    @AfterClass
    public static void destroy() throws Exception {
        govScore.setMaxStepLimit("invoke", defMaxStepLimit);
        govScore.setStepCost("contractCreate", defStepCostCC);
        govScore.setStepPrice(defStepPrice);
    }

    @Test
    public void notEnoughBalance() throws Exception {
        LOG.infoEntering( "disableEnableScore");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);
        try {
            LOG.infoEntering("deploy SCORE");
            HelloWorld.install(iconService, chain, owner);
        }
        catch(ResultTimeoutException ex) {
            LOG.infoExiting();
            LOG.info("FAIL to get result");
            LOG.infoExiting();
            return;
        }
        fail();
    }

    @Test
    public void notEnoughStepLimit() throws Exception {
        LOG.infoEntering( "notEnoughStepLimit");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);

        long value = 10;
        Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, owner.getAddress(), value);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.valueOf(value), bal);
        try {
            HelloWorld.install(iconService, chain, owner, 1);
        }
        catch(TransactionFailureException ex) {
            LOG.infoExiting();
            return;
        }
        fail();
    }

    @Test
    public void installWithInvalidParams() throws Exception {
        LOG.infoEntering( "installWithInvalidParams");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);
        long value = 100000000;
        Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, owner.getAddress(), value);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.valueOf(value), bal);
        RpcObject params = new RpcObject.Builder()
                .put("invalidParam", new RpcValue("invalid"))
                .build();
        try {
            HelloWorld.install(iconService, chain, owner, params, Constants.DEFAULT_STEP_LIMIT);
        }
        catch(TransactionFailureException ex) {
            LOG.infoExiting();
            return;
        }
        fail();
    }

    @Test
    public void updateWithInvalidParams() throws Exception {
        LOG.infoEntering( "installWithInvalidParams");
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(BigInteger.ZERO, bal);
        Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, owner.getAddress(), Constants.DEFAULT_BALANCE);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        bal = iconService.getBalance(owner.getAddress()).execute();
        assertEquals(Constants.DEFAULT_BALANCE, bal);
        LOG.infoEntering( "install");
        HelloWorld score = HelloWorld.install(iconService, chain, owner);
        LOG.infoExiting();
        LOG.infoEntering( "invoke");
        score.invokeHello(owner);
        LOG.infoExiting();
        RpcObject params = new RpcObject.Builder()
                .put("invalidParam", new RpcValue("invalid"))
                .build();
        try {
            LOG.infoEntering( "update");
            score.update(iconService, chain, owner, params);
            LOG.infoExiting();
        }
        catch (TransactionFailureException ex) {
            LOG.infoExiting();
            return;
        }
        fail();
    }

    public void udpateScoreAndCall() {
    }

    public void updateWithInvalidScoreAddress() {
    }

    public void invalidContentNoRootFile() {
    }

    public void invalidContentNotZip() {
    }

    public void invalidContentTooBig() {
    }
}
