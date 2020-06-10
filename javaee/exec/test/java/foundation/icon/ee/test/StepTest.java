package foundation.icon.ee.test;

import foundation.icon.ee.tooling.abi.External;
import foundation.icon.ee.types.StepCost;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import score.Context;
import score.VarDB;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class StepTest extends SimpleTest {
    public static class Score {
        private final VarDB<byte[]> varDB = Context.newVarDB("varDB",
                byte[].class);

        @External
        public void get() {
            varDB.get();
        }

        @External
        public void set(byte[] v) {
            varDB.set(v);
        }

        @External
        public void emptyBody0() {
        }

        @External
        public void emptyBody1(byte[] v) {
        }
    }

    private Contract score;
    private StepCost stepCost;

    @BeforeEach
    public void setUp() {
        super.setUp();
        sm.enableClassMetering(false);
        score = sm.deploy(Score.class);
        stepCost = sm.getStepCost();
    }

    @Test
    void testSetCases() {
        // null -> null
        var baseStep = score.invoke("emptyBody0").getStepUsed().intValue();
        score.invoke("set", (Object) null);
        var step = score.invoke("set", (Object) null).getStepUsed()
                .intValue() -  baseStep;
        System.out.println("step = " + step);
        assertEquals(stepCost.replace()*stepCost.replaceBase(), step);

        // null -> non-null
        var ba1 = new byte[0];
        baseStep = score.invoke("emptyBody1", (Object) ba1).getStepUsed()
                .intValue();
        score.invoke("set", (Object) null);
        step = score.invoke("set", (Object) ba1).getStepUsed().intValue()
                - baseStep;
        assertEquals(stepCost.replace()*stepCost.replaceBase() +
                stepCost.defaultSet(), step);

        // non-null -> non-null
        baseStep = score.invoke("emptyBody1", (Object) ba1).getStepUsed()
                .intValue();
        score.invoke("set", (Object) ba1);
        step = score.invoke("set", (Object) ba1).getStepUsed().intValue()
                - baseStep;
        assertEquals(stepCost.replace()*stepCost.replaceBase(), step);

        // non-null -> null
        baseStep = score.invoke("emptyBody0").getStepUsed().intValue();
        score.invoke("set", (Object) ba1);
        step = score.invoke("set", (Object) null).getStepUsed().intValue()
                - baseStep;
        assertEquals(stepCost.defaultDelete(), step);
    }

    @Test
    void testReplaceBase() {
        var ba_0 = new byte[0];
        score.invoke("set", (Object) ba_0);
        var baseStep = score.invoke("emptyBody1", (Object) ba_0)
                .getStepUsed().intValue();
        var step = score.invoke("set", (Object) ba_0).getStepUsed().intValue()
                - baseStep;
        assertEquals(stepCost.replace()*stepCost.replaceBase(),
                step);

        var ba_rb = new byte[stepCost.replaceBase()];
        baseStep = score.invoke("emptyBody1", (Object) ba_rb)
                .getStepUsed().intValue();
        step = score.invoke("set", (Object) ba_rb).getStepUsed().intValue()
                - baseStep;
        assertEquals(stepCost.replace()*stepCost.replaceBase(),
                step);

        var ba_rbPlus1 = new byte[stepCost.replaceBase() + 1];
        baseStep = score.invoke("emptyBody1",
                (Object) ba_rbPlus1).getStepUsed().intValue();
        step = score.invoke("set", (Object) ba_rbPlus1).getStepUsed().intValue()
                - baseStep;
        assertEquals(stepCost.replace()*ba_rbPlus1.length,
                step);
    }

    @Test
    void testGet() {
        var ba = new byte[10];
        var baseStep = score.invoke("emptyBody1", (Object)ba)
                .getStepUsed().intValue();
        score.invoke("set", (Object)ba);
        var step = score.invoke("get").getStepUsed().intValue()
                - baseStep;
        assertEquals(stepCost.defaultGet() + stepCost.get()*ba.length, step);
    }
}
