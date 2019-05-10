package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.IconAmount;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.CrowdSaleScore;
import foundation.icon.test.score.SampleTokenScore;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;

// TODO What about adding annotation indicating requirements. For example,
// "@require(nodeNum=4,chainNum=1)" indicates it requires at least 4 nodes and
// 1 chain for each.
@Tag(Constants.TAG_NORMAL)
public class BasicScoreTest {
    private static Env.Chain chain;
    private static IconService iconService;

    @BeforeAll
    public static void setUp() {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
    }

    @Test
    public void basicScoreTest() throws Exception {
        KeyWallet ownerWallet = KeyWallet.create();
        KeyWallet aliceWallet = KeyWallet.create();
        KeyWallet bobWallet = KeyWallet.create();

        // transfer initial icx to owner address
        LOG.infoEntering("transfer", "initial icx to owner address");
        Utils.transferIcx(iconService, chain.networkId, chain.godWallet, ownerWallet.getAddress(), "100");
        Utils.ensureIcxBalance(iconService, ownerWallet.getAddress(), 0, 100);
        LOG.infoExiting();

        // deploy sample token
        LOG.infoEntering("deploy", "sample token SCORE");
        long initialSupply = 1000;
        SampleTokenScore sampleTokenScore = SampleTokenScore.mustDeploy(iconService, chain, ownerWallet,
                BigInteger.valueOf(initialSupply), 18);
        LOG.infoExiting();

        // deploy crowd sale
        LOG.infoEntering("deploy", "crowd sale SCORE");
        CrowdSaleScore crowdSaleScore = CrowdSaleScore.mustDeploy(iconService, chain, ownerWallet,
                    new BigInteger("100"), sampleTokenScore.getAddress(), 10);
        LOG.infoExiting();

        // send 50 icx to Alice
        LOG.infoEntering("transfer", "50 to Alice; 100 to Bob");
        Utils.transferIcx(iconService, chain.networkId, chain.godWallet, aliceWallet.getAddress(), "50");
        Utils.transferIcx(iconService, chain.networkId, chain.godWallet, bobWallet.getAddress(), "100");
        Utils.ensureIcxBalance(iconService, aliceWallet.getAddress(), 0, 50);
        Utils.ensureIcxBalance(iconService, bobWallet.getAddress(), 0, 100);
        LOG.infoExiting();

        // transfer all tokens to crowd sale score
        LOG.infoEntering("transfer", "all to crowdSaleScore from owner");
        sampleTokenScore.transfer(ownerWallet, crowdSaleScore.getAddress(), BigInteger.valueOf(initialSupply));
        LOG.infoExiting();

        // Alice: send icx to crowd sale score from Alice and Bob
        LOG.infoEntering("transfer", "to crowdSaleScore(40 from Alice, 60 from Bob)");
        Utils.transferIcx(iconService, chain.networkId, aliceWallet, crowdSaleScore.getAddress(), "40");
        Utils.transferIcx(iconService, chain.networkId, bobWallet, crowdSaleScore.getAddress(), "60");
        sampleTokenScore.ensureTokenBalance(aliceWallet, 40);
        sampleTokenScore.ensureTokenBalance(bobWallet, 60);
        LOG.infoExiting();

        // check if goal reached
        LOG.infoEntering("call", "checkGoalReached() and goalReached()");
        crowdSaleScore.ensureCheckGoalReached(ownerWallet);
        LOG.infoExiting();

        // do safe withdrawal
        LOG.infoEntering("call", "safeWithdrawal()");
        TransactionResult result = crowdSaleScore.safeWithdrawal(ownerWallet);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new IOException("Failed to execute safeWithdrawal.");
        }
        BigInteger amount = IconAmount.of("100", IconAmount.Unit.ICX).toLoop();
        sampleTokenScore.ensureFundTransfer(result, crowdSaleScore.getAddress(), ownerWallet.getAddress(), amount);

        // check the final icx balance of owner
        Utils.ensureIcxBalance(iconService, ownerWallet.getAddress(), 100, 200);
        LOG.infoExiting();
    }
}
