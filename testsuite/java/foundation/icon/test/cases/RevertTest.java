package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.StepCounterScore;
import org.junit.BeforeClass;
import org.junit.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static junit.framework.TestCase.assertEquals;
import static org.hamcrest.CoreMatchers.not;
import static org.hamcrest.MatcherAssert.assertThat;

public class RevertTest {
    private static Env.Chain chain;
    private static IconService iconService;

    @BeforeClass
    public static void setUp() {
        Env.Node node = Env.getInstance().nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
    }

    @Test
    public void testRevert() throws Exception {
        KeyWallet ownerWallet = Utils.createAndStoreWallet();

        LOG.infoEntering("deploy", "SCORE1");
        StepCounterScore score1 = StepCounterScore.mustDeploy(iconService,
                chain, ownerWallet);
        LOG.infoExiting("deployed:" + score1);
        LOG.infoEntering("deploy", "SCORE2");
        StepCounterScore score2 = StepCounterScore.mustDeploy(iconService,
                chain, ownerWallet);
        LOG.infoExiting("deployed:" + score2);

        TransactionResult txr;
        BigInteger v1, v2, v, v1new, v2new;

        LOG.infoEntering("call", score1 + ".getStep()");
        v1 = score1.getStep(ownerWallet.getAddress());
        LOG.infoExiting(v1.toString());
        LOG.infoEntering("call", score2 + ".getStep()");
        v2 = score2.getStep(ownerWallet.getAddress());
        LOG.infoExiting(v2.toString());

        v = v1.add(BigInteger.ONE);

        LOG.infoEntering("call", score2 + ".setStepOf(" + score1 + "," + v + ")");
        txr = score2.setStepOf(ownerWallet, score1.getAddress(), v);
        assertEquals(Constants.STATUS_SUCCESS, txr.getStatus());
        LOG.infoExiting("Result:" + txr);

        v1 = score1.getStep(ownerWallet.getAddress());
        assertEquals(v, v1);

        LOG.infoEntering("call", score2 + ".setStepOf(" + score1 + "," + v + ")");
        txr = score2.setStepOf(ownerWallet, score1.getAddress(), v);
        assertThat(Constants.STATUS_SUCCESS, not(txr.getStatus()));
        LOG.infoExiting("Result:" + txr);

        LOG.infoEntering("call", score1 + ".getStep()");
        v1 = score1.getStep(ownerWallet.getAddress());
        LOG.infoExiting(v1.toString());
        LOG.infoEntering("call", score2 + ".getStep()");
        v2 = score2.getStep(ownerWallet.getAddress());
        LOG.infoExiting(v2.toString());

        v = v.add(BigInteger.ONE);

        LOG.infoEntering("call", score1 + ".trySetStepWith(" + score2 + "," + v + ")");
        txr = score1.trySetStepWith(ownerWallet, score2.getAddress(), v);
        LOG.infoExiting("Result:" + txr);
        if (!Constants.STATUS_SUCCESS.equals(txr.getStatus())) {
            LOG.warning("It should SUCCEED");
            return;
        }

        LOG.infoEntering("call", score2 + ".getStep()");
        v2new = score2.getStep(ownerWallet.getAddress());
        if (!v2.equals(v2new)) {
            LOG.warning(score2 + ".getValue()=>" + v2new + " expect=" + v2);
            return;
        }
        LOG.infoExiting(v2new.toString());

        LOG.infoEntering("call", score1 + ".getStep()");
        v1new = score1.getStep(ownerWallet.getAddress());
        if (!v.equals(v1new)) {
            LOG.warning(score1 + ".getValue()=>" + v1new + " expect=" + v);
            return;
        }
        LOG.infoExiting(v1new.toString());
    }
}
