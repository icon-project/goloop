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
        Env.Node node = Env.getInstance().nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        chain.governorWallet = KeyWallet.create();
        govScore = new GovScore(iconService, chain);
        initDeploy();
    }

    public static void initDeploy() throws Exception {
        RpcObject params = new RpcObject.Builder()
                .put("contextType", new RpcValue("invoke"))
                .build();
        defMaxStepLimit = Utils.icxCall(iconService, chain.networkId, chain.governorWallet, Constants.CHAINSCORE_ADDRESS,
                "getMaxStepLimit", params).asInteger();


        defStepCostCC = Utils.icxCall(iconService, chain.networkId, chain.governorWallet, Constants.CHAINSCORE_ADDRESS,
                "getStepCosts", null).asObject().getItem("contractCreate").asInteger();

        defStepPrice = Utils.icxCall(iconService, chain.networkId, chain.governorWallet, Constants.CHAINSCORE_ADDRESS,
                "getStepPrice", null).asInteger();

        Bytes txHash = Utils.transfer(iconService, chain.networkId, chain.godWallet, chain.governorWallet.getAddress(), 9999999);
        try {
            TransactionResult result = Utils.getTransactionResult(iconService, txHash, 5000);
            assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);
        }
        catch (ResultTimeoutException ex) {
            throw ex;
        }
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
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        if(bal.compareTo(BigInteger.valueOf(0)) != 0) {
            throw new Exception();
        }

        try {
            HelloWorld.install(iconService, chain, owner);
        }
        catch(ResultTimeoutException ex) {
            return;
        }
        fail();
    }

    @Test
    public void notEnoughStepLimit() throws Exception {
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        if(bal.compareTo(BigInteger.valueOf(0)) != 0) {
            throw new Exception();
        }

        long value = 10;
        Utils.transfer(iconService, chain.networkId, chain.godWallet, owner.getAddress(), value);
        while(true) {
            bal = iconService.getBalance(owner.getAddress()).execute();
            if(bal.compareTo(BigInteger.valueOf(value)) == 0) {
                break;
            }
        }

        try {
            HelloWorld.install(iconService, chain, owner, 1);
        }
        catch(TransactionFailureException ex) {
            return;
        }
        fail();
    }

    @Test
    public void installWithInvalidParams() throws Exception {
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        if(bal.compareTo(BigInteger.valueOf(0)) != 0) {
            throw new Exception();
        }

        long value = 100000000;
        Utils.transfer(iconService, chain.networkId, chain.godWallet, owner.getAddress(), value);
        while(true) {
            bal = iconService.getBalance(owner.getAddress()).execute();
            if(bal.compareTo(BigInteger.valueOf(value)) == 0) {
                break;
            }
        }
        RpcObject params = new RpcObject.Builder()
                .put("invalidParam", new RpcValue("invalid"))
                .build();
        try {
            HelloWorld.install(iconService, chain, owner, params, -1);
        }
        catch(TransactionFailureException ex) {
            return;
        }
        fail();
    }

    @Test
    public void updateWithInvalidParams() throws Exception {
        KeyWallet owner = KeyWallet.create();
        BigInteger bal = iconService.getBalance(owner.getAddress()).execute();
        if(bal.compareTo(BigInteger.valueOf(0)) != 0) {
            throw new Exception();
        }
        long value = 999999999;
        Utils.transfer(iconService, chain.networkId, chain.godWallet, owner.getAddress(), value);
        while(true) {
            bal = iconService.getBalance(owner.getAddress()).execute();
            if(bal.compareTo(BigInteger.valueOf(value)) == 0) {
                break;
            }
        }

        HelloWorld score = HelloWorld.install(iconService, chain, owner, 10000);

        score.invokeHello(owner);

        RpcObject params = new RpcObject.Builder()
                .put("invalidParam", new RpcValue("invalid"))
                .build();
        try {
            score.update(iconService, chain, owner, params);
        }
        catch (TransactionFailureException ex) {
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
