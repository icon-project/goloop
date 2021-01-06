package foundation.icon.ee;

import foundation.icon.ee.test.GoldenTest;
import org.junit.jupiter.api.Test;
import score.Address;
import score.Context;
import score.annotation.External;

public class ReenterTest extends GoldenTest {
    public static class Score {
        private static final int MAX_COUNTER = 3;
        private int counter = 0;
        private Address self;

        public Score() {
            self = Context.getAddress();
        }

        @External
        public void reenter() {
            Context.println("counter=" + counter);
            if (counter < MAX_COUNTER) {
                counter++;
                Context.call(self, "reenter");
            }
        }
    }

    @Test
    public void test() {
        var score = sm.mustDeploy(Score.class);
        score.invoke("reenter");
    }
}
