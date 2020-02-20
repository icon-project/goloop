package foundation.icon.ee.ipc;

import score.Address;
import score.Context;
import score.RevertException;
import score.ScoreRevertException;
import foundation.icon.ee.test.GoldenTest;
import foundation.icon.ee.tooling.abi.External;
import org.junit.jupiter.api.Test;

public class ExceptionTest extends GoldenTest {
    public static class RevertScore {
        @External
        public void run() {
            Context.revert(1, "user revert");
        }
    }

    public static class Score {
        @External
        public void run(Address addrGood, Address addrBad) {
            try {
                throw new ScoreRevertException("test");
            } catch (ScoreRevertException e) {
                Context.println("OK");
            }

            try {
                Context.call(addrGood,"run");
            } catch (RevertException e) {
                Context.println("OK");
            }

            try {
                Context.call(addrBad,"run");
            } catch (Exception e) {
                Context.println("OK : " + e);
            }
        }
    }

    @Test
    public void test() {
        var score = sm.deploy(Score.class);
        var revertScore = sm.deploy(RevertScore.class);
        score.invoke("run", revertScore.getAddress(), sm.newScoreAddress());
    }
}
