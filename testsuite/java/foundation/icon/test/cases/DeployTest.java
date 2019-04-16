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
    private static KeyWallet govWallet;
    private static KeyWallet godWallet;
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
        chain = Env.nodes[0].chains[0];
        godWallet = chain.godWallet;
        iconService = new IconService(new HttpProvider(node.endpointUrl));
        govWallet = KeyWallet.create();
        govScore = new GovScore(iconService, Env.nodes[0].chains[0].networkId, govWallet);
        initDeploy();
    }

    public static void initDeploy() throws Exception {
        RpcObject params = new RpcObject.Builder()
                .put("contextType", new RpcValue("invoke"))
                .build();
        defMaxStepLimit = Utils.icxCall(iconService, BigInteger.valueOf(0), govWallet, Constants.CHAINSCORE_ADDRESS,
                "getMaxStepLimit", params).asInteger();


        defStepCostCC = Utils.icxCall(iconService, BigInteger.valueOf(0), govWallet, Constants.CHAINSCORE_ADDRESS,
                "getStepCosts", null).asObject().getItem("contractCreate").asInteger();

        defStepPrice = Utils.icxCall(iconService, BigInteger.valueOf(0), govWallet, Constants.CHAINSCORE_ADDRESS,
                "getStepPrice", null).asInteger();


        Bytes txHash = Utils.transfer(iconService, godWallet, govWallet.getAddress(), 9999999);
        try {
            TransactionResult result = Utils.getTransactionResult(iconService, txHash, 5000);
            assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);
        }
        catch (ResultTimeoutException ex) {
            throw ex;
        }
        BigInteger bal = iconService.getBalance(govWallet.getAddress()).execute();

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
            HelloWorld.mustDeploy(iconService, owner, Env.nodes[0].chains[0].networkId);
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
        Utils.transfer(iconService, godWallet, owner.getAddress(), value);
        while(true) {
            bal = iconService.getBalance(owner.getAddress()).execute();
            if(bal.compareTo(BigInteger.valueOf(value)) == 0) {
                break;
            }
        }

        try {
            HelloWorld.mustDeploy(iconService, owner, Env.nodes[0].chains[0].networkId, 1);
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
        Utils.transfer(iconService, godWallet, owner.getAddress(), value);
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
            HelloWorld.mustDeploy(iconService, owner, params, Env.nodes[0].chains[0].networkId, -1);
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
        Utils.transfer(iconService, godWallet, owner.getAddress(), value);
        while(true) {
            bal = iconService.getBalance(owner.getAddress()).execute();
            if(bal.compareTo(BigInteger.valueOf(value)) == 0) {
                break;
            }
        }

        HelloWorld score = HelloWorld.mustDeploy(iconService, owner, Env.nodes[0].chains[0].networkId, 10000);

        score.invokeHello(owner);

        RpcObject params = new RpcObject.Builder()
                .put("invalidParam", new RpcValue("invalid"))
                .build();
        try {
            score.update(iconService, owner, params, Env.nodes[0].chains[0].networkId);
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
