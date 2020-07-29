package foundation.icon.ee;

import foundation.icon.ee.test.SimpleTest;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.Context;
import score.annotation.External;

public class IntercallTest2 extends SimpleTest {
    public static class BadParam {
        @External
        public void run() {
            try {
                Context.call(Context.getAddress(), "test", new Object());
            } catch (Exception ignored) {
            }
        }
    }

    @Test
    void testBadParam() {
        var c = sm.deploy(BadParam.class);
        var res = c.invoke("run");
        Assertions.assertEquals(0, res.getStatus());
    }
}
