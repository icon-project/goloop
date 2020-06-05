package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.EventLog;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.Score;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.List;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;

@Tag(Constants.TAG_PY_SCORE)
@Tag(Constants.TAG_PY_GOV)
public class ScoreMethodTest {
    private static final String SCORE1_PATH = "method_caller";
    private static TransactionHandler txHandler;
    private static Wallet owner;

    @BeforeAll
    static void init() {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
        owner = chain.godWallet;
    }

    @Test
    public void callInternalsDirectly() throws Exception {
        LOG.infoEntering("callInternalsDirectly");

        LOG.infoEntering("deployScore", SCORE1_PATH);
        Score testScore = txHandler.deploy(owner, Score.getFilePath(SCORE1_PATH), null);
        LOG.infoExiting();

        LOG.infoEntering("send transactions");
        var txs = new ArrayList<Bytes>();
        txs.add(testScore.invoke(owner, "on_install", null));
        txs.add(testScore.invoke(owner, "on_update", null));
        txs.add(testScore.invoke(owner, "fallback", null));
        txs.add(testScore.invoke(owner, "fallback", null, 100, 1000));
        LOG.infoExiting();

        LOG.infoEntering("check results");
        for (var tx : txs) {
            var result = txHandler.getResult(tx);
            assertEquals(result.getStatus(), Constants.STATUS_FAILURE);
        }
        LOG.infoExiting();

        LOG.infoExiting();
    }

    @Test
    public void checkInternalCalls() throws Exception {
        LOG.infoEntering("checkInternalCalls");

        LOG.infoEntering("on_install");
        var dtx = txHandler.deployOnly(owner, Score.getFilePath(SCORE1_PATH), null);
        var result = txHandler.getResult(dtx);
        assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);

        // If the audit is enabled, it should be accepted to check
        // events.
        var score_addr = result.getScoreAddress();
        var acceptResult = txHandler.acceptScoreIfAuditEnabled(dtx);
        if (acceptResult != null) {
            result = acceptResult;
        }

        assertEquals(true, EventLog.checkScenario(List.of(
                new EventLog(score_addr, "Called(str,int)", "on_install")
        ), result));
        LOG.infoExiting();

        LOG.infoEntering("on_update");
        dtx = txHandler.deployOnly(owner, new Address(score_addr), Score.getFilePath(SCORE1_PATH), null);
        result = txHandler.getResult(dtx);
        assertEquals(result.getStatus(), Constants.STATUS_SUCCESS);
        acceptResult = txHandler.acceptScoreIfAuditEnabled(dtx);
        if (acceptResult != null) {
            result = acceptResult;
        }
        assertEquals(true, EventLog.checkScenario(List.of(
                new EventLog(score_addr, "Called(str,int)", "on_update")
        ), result));
        LOG.infoExiting();

        LOG.infoEntering("fallback");
        Bytes tx = txHandler.transfer(new Address(score_addr), BigInteger.valueOf(1000));
        result = txHandler.getResult(tx);
        assertEquals(true, EventLog.checkScenario(List.of(
                new EventLog(score_addr, "Called(str,int)", "fallback")
        ), result));
        LOG.infoExiting();

        LOG.infoExiting();
    }
}
