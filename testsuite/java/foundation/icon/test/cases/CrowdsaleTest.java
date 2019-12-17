package foundation.icon.test.cases;

import foundation.icon.icx.Call;
import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.IconAmount;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.CrowdSaleScore;
import foundation.icon.test.score.GovScore;
import foundation.icon.test.score.SampleTokenScore;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.math.BigDecimal;
import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;

@Tag(Constants.TAG_GOVERNANCE)
class CrowdsaleTest {
    private static IconService iconService;
    private static Env.Chain chain;
    private static KeyWallet ownerWallet;
    private static GovScore govScore;
    private static GovScore.Fee fee;

    private static final String SAMPLETOKEN_JAR = Constants.JAVA_SCORE_ROOT + "sampleToken.jar";
    private static final String CROWDSALE_JAR = Constants.JAVA_SCORE_ROOT + "crowdsale.jar";

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
        Address[] addrs = {ownerWallet.getAddress(), chain.governorWallet.getAddress()};
        Utils.transferAndCheck(iconService, chain, chain.godWallet, addrs, BigInteger.TEN.pow(20));

        govScore.setMaxStepLimit("invoke", BigInteger.valueOf(1000000));
        govScore.setMaxStepLimit("query", BigInteger.valueOf(1000000));
    }

    @AfterAll
    static void destroy() throws Exception {
        govScore.setFee(fee);
    }

    @Test
    void testCrowdsale() throws Exception {
        KeyWallet aliceWallet = KeyWallet.create();
        KeyWallet bobWallet = KeyWallet.create();
        BigInteger ownerBalance = iconService.getBalance(ownerWallet.getAddress()).execute();

        // deploy token SCORE
        BigInteger decimals = BigInteger.valueOf(18);
        BigInteger initialSupply = BigInteger.valueOf(1000);
        SampleTokenScore tokenScore = new SampleTokenScore(iconService, chain,
                deploySampleToken(decimals, initialSupply));

        // deploy crowdsale SCORE
        BigInteger fundingGoalInIcx = BigInteger.valueOf(100);
        CrowdSaleScore crowdsaleScore = new CrowdSaleScore(iconService, chain,
                deployCrowdsale(tokenScore.getAddress(), fundingGoalInIcx));

        // send 50 icx to Alice, 100 to Bob
        LOG.infoEntering("transfer icx", "50 to Alice; 100 to Bob");
        Utils.transferIcx(iconService, chain.networkId, chain.godWallet, aliceWallet.getAddress(), "50");
        Utils.transferIcx(iconService, chain.networkId, chain.godWallet, bobWallet.getAddress(), "100");
        Utils.ensureIcxBalance(iconService, aliceWallet.getAddress(), 0, 50);
        Utils.ensureIcxBalance(iconService, bobWallet.getAddress(), 0, 100);
        LOG.infoExiting();

        // transfer all tokens to crowdsale score
        LOG.infoEntering("transfer token", "all tokens to crowdsale score from owner");
        Bytes txHash = tokenScore.transfer(ownerWallet, crowdsaleScore.getAddress(), initialSupply);
        ensureFundingGoal(txHash, crowdsaleScore.getAddress(), fundingGoalInIcx);
        ensureTokenBalance(tokenScore, crowdsaleScore.getAddress(), initialSupply.longValue());
        LOG.infoExiting();

        // send icx to crowdsale score from Alice and Bob
        LOG.infoEntering("transfer icx", "to crowdsale score (40 from Alice, 60 from Bob)");
        Utils.transferIcx(iconService, chain.networkId, aliceWallet, crowdsaleScore.getAddress(), "40");
        Utils.transferIcx(iconService, chain.networkId, bobWallet, crowdsaleScore.getAddress(), "60");
        tokenScore.ensureTokenBalance(aliceWallet, 40);
        tokenScore.ensureTokenBalance(bobWallet, 60);
        LOG.infoExiting();

        // check if goal reached
        LOG.infoEntering("call", "checkGoalReached()");
        crowdsaleScore.ensureCheckGoalReached(ownerWallet);
        LOG.infoExiting();

        // do safe withdrawal
        LOG.infoEntering("call", "safeWithdrawal()");
        TransactionResult result = crowdsaleScore.safeWithdrawal(ownerWallet);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new IOException("Failed to execute safeWithdrawal.");
        }
        BigInteger amount = IconAmount.of("100", IconAmount.Unit.ICX).toLoop();
        tokenScore.ensureFundTransfer(result, crowdsaleScore.getAddress(), ownerWallet.getAddress(), amount);

        // check the final icx balance of owner
        LOG.info("Initial ICX balance of owner: " + ownerBalance);
        ensureIcxBalance(iconService, ownerWallet.getAddress(), ownerBalance, ownerBalance.add(amount));
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
        Address scoreAddr = Score.install(iconService, chain, ownerWallet, SAMPLETOKEN_JAR,
                                          params, 1000000, Constants.CONTENT_TYPE_JAVA);
        LOG.info("scoreAddr = " + scoreAddr);
        LOG.infoExiting();
        return scoreAddr;
    }

    private Address deployCrowdsale(Address tokenScore, BigInteger fundingGoalInIcx) throws Exception {
        LOG.infoEntering("deploy", "Crowdsale");
        RpcObject params = new RpcObject.Builder()
                .put("_fundingGoalInIcx", new RpcValue(fundingGoalInIcx))
                .put("_tokenScore", new RpcValue(tokenScore))
                .put("_durationInBlocks", new RpcValue(BigInteger.valueOf(10)))
                .build();
        Address scoreAddr = Score.install(iconService, chain, ownerWallet, CROWDSALE_JAR,
                                          params, 1000000, Constants.CONTENT_TYPE_JAVA);
        LOG.info("scoreAddr = " + scoreAddr);
        LOG.infoExiting();
        return scoreAddr;
    }

    private static void ensureFundingGoal(Bytes txHash, Address scoreAddress, BigInteger fundingGoalInIcx)
            throws IOException, ResultTimeoutException {
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, scoreAddress, "CrowdsaleStarted(int)");
        if (event != null) {
            BigInteger fundingGoalInLoop = IconAmount.of(fundingGoalInIcx, IconAmount.Unit.ICX).toLoop();
            BigInteger fundingGoalFromScore = event.getData().get(0).asInteger();
            assertEquals(fundingGoalInLoop, fundingGoalFromScore);
        } else {
            throw new IOException("ensureFundingGoal failed.");
        }
    }

    // TODO: integrate into Utils class
    private static void ensureIcxBalance(IconService iconService, Address address,
                                        BigInteger oldVal, BigInteger newVal) throws Exception {
        long limitTime = System.currentTimeMillis() + Constants.DEFAULT_WAITING_TIME;
        while (true) {
            BigInteger icxBalance = iconService.getBalance(address).execute();
            String msg = "ICX balance of " + address + ": " + icxBalance;
            if (icxBalance.equals(oldVal)) {
                if (limitTime < System.currentTimeMillis()) {
                    throw new ResultTimeoutException();
                }
                try {
                    // wait until block confirmation
                    LOG.debug(msg + "; Retry in 1 sec.");
                    Thread.sleep(1000);
                } catch (InterruptedException e) {
                    e.printStackTrace();
                }
            } else if (icxBalance.equals(newVal)) {
                LOG.info(msg);
                break;
            } else {
                throw new IOException("ICX balance mismatch!");
            }
        }
    }

    // TODO: integrate into SampleTokenScore
    private static void ensureTokenBalance(SampleTokenScore tokenScore, Address address, long value) throws ResultTimeoutException, IOException {
        long limitTime = System.currentTimeMillis() + Constants.DEFAULT_WAITING_TIME;
        while (true) {
            RpcObject params = new RpcObject.Builder()
                    .put("_owner", new RpcValue(address))
                    .build();
            Call<RpcItem> call = new Call.Builder()
                    .to(tokenScore.getAddress())
                    .method("balanceOf")
                    .params(params)
                    .build();
            BigInteger balance = iconService.call(call).execute().asInteger();

            String msg = "Token balance of " + address + ": " + balance;
            if (balance.equals(BigInteger.valueOf(0))) {
                try {
                    if (limitTime < System.currentTimeMillis()) {
                        throw new ResultTimeoutException();
                    }
                    // wait until block confirmation
                    LOG.info(msg + "; Retry in 1 sec.");
                    Thread.sleep(1000);
                } catch (InterruptedException e) {
                    e.printStackTrace();
                }
            } else if (balance.equals(BigInteger.valueOf(value).multiply(BigDecimal.TEN.pow(18).toBigInteger()))) {
                LOG.info(msg);
                break;
            } else {
                throw new IOException("Token balance mismatch!");
            }
        }
    }
}
