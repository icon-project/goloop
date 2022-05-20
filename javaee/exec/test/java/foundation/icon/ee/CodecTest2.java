package foundation.icon.ee;

import foundation.icon.ee.test.SimpleTest;
import org.junit.jupiter.api.Test;

public class CodecTest2 extends SimpleTest {
    @Test
    public void testMultipleDeploy() {
        final int N = 10;
        for (int i=0; i<N; i++) {
            var score2 = sm.mustDeploy(new Class<?>[]{CodecTest.Score.class, CodecTest.User.class});
            score2.invoke("run");
        }
    }
}
