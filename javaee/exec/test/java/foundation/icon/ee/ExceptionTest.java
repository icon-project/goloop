package foundation.icon.ee;

import foundation.icon.ee.test.GoldenTest;
import org.junit.jupiter.api.Test;
import score.Address;
import score.Context;
import score.RevertedException;
import score.UserRevertedException;
import score.annotation.External;

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
                throw new UserRevertedException("test");
            } catch (UserRevertedException e) {
                Context.println("OK");
            }

            try {
                Context.call(addrGood,"run");
            } catch (RevertedException e) {
                if (e instanceof UserRevertedException) {
                    Context.println("OK code=" + ((UserRevertedException)e).getCode());
                } else {
                    Context.println("NOT OK");
                }
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
        var score = sm.mustDeploy(Score.class);
        var revertScore = sm.mustDeploy(RevertScore.class);
        score.invoke("run", revertScore.getAddress(), sm.newScoreAddress());
    }
}
