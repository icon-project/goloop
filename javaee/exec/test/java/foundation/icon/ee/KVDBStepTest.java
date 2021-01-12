package foundation.icon.ee;

import foundation.icon.ee.test.SimpleTest;
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.Test;
import score.Context;
import score.VarDB;
import score.annotation.External;

import java.math.BigInteger;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class KVDBStepTest extends SimpleTest {
    public static class Score {
        private final VarDB<byte[]> varDB = Context.newVarDB("varDB",
                byte[].class);

        @External
        public void set(byte[] v) {
            varDB.set(v);
        }

        @External
        public void set2(byte[] v1, byte[] v2) {
            varDB.set(v1);
            varDB.set(v2);
        }
    }

    @Test
    void testSetStepCharge() {
        var score = sm.mustDeploy(Score.class);
        var nonNull = new byte[]{(byte)0};

        // null -> null
        var stepUsed = score.invoke("set", (Object) null)
                .getStepUsed();
        var res = score.invoke(BigInteger.ZERO, stepUsed, "set",
                (Object) null);
        assertEquals(stepUsed, res.getStepUsed());
        res = score.tryInvoke(BigInteger.ZERO,
                stepUsed.subtract(BigInteger.ONE), "set", (Object) null);
        assertEquals(Status.OutOfStep, res.getStatus());

        // null -> non-null
        stepUsed = score.invoke("set", (Object) nonNull).getStepUsed();
        score.invoke(BigInteger.ZERO, stepUsed, "set", (Object) null);
        res = score.invoke(BigInteger.ZERO, stepUsed, "set",
                (Object) nonNull);
        assertEquals(stepUsed, res.getStepUsed());
        score.invoke(BigInteger.ZERO, stepUsed, "set", (Object) null);
        res = score.tryInvoke(BigInteger.ZERO,
                stepUsed.subtract(BigInteger.ONE), "set",
                (Object) nonNull);
        assertEquals(Status.OutOfStep, res.getStatus());

        // non-null -> non-null
        score.invoke("set", (Object) nonNull);
        stepUsed = score.invoke("set", (Object) nonNull).getStepUsed();
        res = score.invoke(BigInteger.ZERO, stepUsed, "set",
                (Object) nonNull);
        assertEquals(stepUsed, res.getStepUsed());
        res = score.tryInvoke(BigInteger.ZERO,
                stepUsed.subtract(BigInteger.ONE), "set",
                (Object) nonNull);
        assertEquals(Status.OutOfStep, res.getStatus());

        // non-null -> null
        // actually we do null -> non-null -> null to avoid negative stepUsed
        score.invoke(BigInteger.ZERO, stepUsed, "set", (Object) null);
        stepUsed = score.invoke("set2", null, nonNull)
                .getStepUsed();
        score.invoke(BigInteger.ZERO, stepUsed, "set", (Object) null);
        res = score.invoke(BigInteger.ZERO, stepUsed, "set2",
                null, nonNull);
        assertEquals(stepUsed, res.getStepUsed());
        score.invoke(BigInteger.ZERO, stepUsed, "set", (Object) null);
        res = score.tryInvoke(BigInteger.ZERO,
                stepUsed.subtract(BigInteger.ONE), "set2",
                null, nonNull);
        assertEquals(Status.OutOfStep, res.getStatus());
    }
}
