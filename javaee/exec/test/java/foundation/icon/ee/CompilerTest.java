package foundation.icon.ee;

import foundation.icon.ee.test.GoldenTest;
import foundation.icon.ee.tooling.abi.ABICompilerException;
import org.junit.jupiter.api.Test;
import score.annotation.EventLog;
import score.annotation.External;

import static org.junit.jupiter.api.Assertions.fail;

public class CompilerTest extends GoldenTest {
    @Test
    public void testNoInit() {
        sm.deploy(ScoreWithoutInit.class);
    }

    public static class ScoreWithoutInit {
    }

    @Test
    public void testMultipleInits() {
        try {
            sm.deploy(ScoreWithMultipleInits.class, "Hello");
            fail();
        } catch (ABICompilerException e) {
            System.err.println("Expected " + e.getMessage());
        }
    }

    public static class ScoreWithMultipleInits {
        public ScoreWithMultipleInits() {}
        public ScoreWithMultipleInits(String s) {}
    }

    @Test
    public void testMultipleSameNames() {
        try {
            sm.deploy(ScoreWithMultipleSameExternals.class);
            fail();
        } catch (ABICompilerException e) {
            System.err.println("Expected " + e.getMessage());
        }
        try {
            sm.deploy(ScoreWithMultipleSameEvents.class);
            fail();
        } catch (ABICompilerException e) {
            System.err.println("Expected " + e.getMessage());
        }
    }

    public static class ScoreWithMultipleSameExternals {
        @External
        public void sameMethod() {}
        @External
        public void sameMethod(String s) {}
    }

    public static class ScoreWithMultipleSameEvents {
        @EventLog
        void sameEvent(String s) {}
        @EventLog
        void sameEvent(String a, String b) {}
    }

    @Test
    public void testParamType() {
        try {
            sm.deploy(ScoreWithFloatParam.class);
            fail();
        } catch (ABICompilerException e) {
            System.err.println("Expected " + e.getMessage());
        }
        try {
            sm.deploy(ScoreWithDoubleParam.class);
            fail();
        } catch (ABICompilerException e) {
            System.err.println("Expected " + e.getMessage());
        }
    }

    public static class ScoreWithFloatParam {
        @External
        public void methodFloat(float f) {}
    }

    public static class ScoreWithDoubleParam {
        @External
        public void methodDouble(double d) {}
    }

    @Test
    public void testReturnType() {
        try {
            sm.deploy(ScoreWithFloatReturn.class);
            fail();
        } catch (ABICompilerException e) {
            System.err.println("Expected " + e.getMessage());
        }
        try {
            sm.deploy(ScoreWithDoubleReturn.class);
            fail();
        } catch (ABICompilerException e) {
            System.err.println("Expected " + e.getMessage());
        }
    }

    public static class ScoreWithFloatReturn {
        @External(readonly=true)
        public float returnFloat() {return 1.0f;}
    }

    public static class ScoreWithDoubleReturn {
        @External(readonly=true)
        public double returnDouble() {return 1.0d;}
    }
}
