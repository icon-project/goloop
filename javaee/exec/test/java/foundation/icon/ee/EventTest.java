package foundation.icon.ee;

import foundation.icon.ee.test.GoldenTest;
import org.junit.jupiter.api.Test;
import score.Address;
import score.annotation.EventLog;
import score.annotation.External;

import java.math.BigInteger;

public class EventTest extends GoldenTest {
    public static class Score {
        @EventLog(indexed = 0)
        private void event0(boolean a0, byte a1, char a2, short a3, int a4,
                            long a5, BigInteger a6, String a7, Address a8,
                            byte[] a9) {
        }

        @EventLog(indexed = 1)
        private void event1(boolean a0, byte a1, char a2, short a3, int a4,
                            long a5, BigInteger a6, String a7, Address a8,
                            byte[] a9) {
        }

        @EventLog(indexed = 2)
        private void event2(boolean a0, byte a1, char a2, short a3, int a4,
                            long a5, BigInteger a6, String a7, Address a8,
                            byte[] a9) {
        }

        @EventLog(indexed = 3)
        private void event3(boolean a0, byte a1, char a2, short a3, int a4,
                            long a5, BigInteger a6, String a7, Address a8,
                            byte[] a9) {
        }

        @External
        public void logEvent(boolean a0, byte a1, char a2, short a3, int a4,
                             long a5, BigInteger a6, String a7, Address a8,
                             byte[] a9) {
            event0(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9);
            event1(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9);
            event2(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9);
            event3(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9);
        }
    }

    @Test
    void testLogEvents() {
        var score = sm.mustDeploy(Score.class);
        score.invoke("logEvent", true, 1, 2, 3, 4, 5, 6, "7",
                sm.newExternalAddress(), new byte[]{0, 1, 2, 3});
    }
}
