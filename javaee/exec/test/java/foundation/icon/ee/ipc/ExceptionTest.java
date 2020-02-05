package foundation.icon.ee.ipc;

import avm.Address;
import avm.Blockchain;
import avm.RevertException;
import avm.ScoreRevertException;
import foundation.icon.ee.test.GoldenTest;
import foundation.icon.ee.tooling.abi.External;
import org.junit.jupiter.api.Test;

public class ExceptionTest extends GoldenTest {
    public static class RevertScore {
        @External
        public void run() {
            Blockchain.revert(1, "user revert");
        }
    }

    public static class Score {
        @External
        public void run(Address addrGood, Address addrBad) {
            try {
                throw new ScoreRevertException("test");
            } catch (ScoreRevertException e) {
                Blockchain.println("OK");
            }

            try {
                Blockchain.call(addrGood,"run");
            } catch (RevertException e) {
                Blockchain.println("OK");
            }

            try {
                Blockchain.call(addrBad,"run");
            } catch (Exception e) {
                Blockchain.println("OK : " + e);
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
