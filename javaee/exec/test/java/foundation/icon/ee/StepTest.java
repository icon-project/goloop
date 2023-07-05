package foundation.icon.ee;

import foundation.icon.ee.test.ContractAddress;
import foundation.icon.ee.test.SimpleTest;
import foundation.icon.ee.types.StepCost;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import score.Context;
import score.VarDB;
import score.annotation.External;

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

        @External
        public void hash(byte[] v) {
            Context.hash("sha3-256", v);
        }
    }

    private ContractAddress score;
    private StepCost stepCost;
    private int hashCost;

    @BeforeEach
    public void setUp() {
        super.setUp();
        sm.enableClassMetering(false);
        score = sm.mustDeploy(Score.class);
        stepCost = sm.getStepCost();
        var storageKey = new byte[]{2, (byte)0x85, 'v', 'a', 'r', 'D', 'B'};
        // call, read OG, create storageKey object
        var baseCost = score.invoke("emptyBody1", (Object)storageKey)
                .getStepUsed().intValue();
        hashCost = score.invoke("hash", (Object) storageKey)
                .getStepUsed().intValue() - baseCost;
    }

    @Test
    void testSetCases() {
        // null -> null
        var baseStep = score.invoke("emptyBody0").getStepUsed().intValue();
        score.invoke("set", (Object) null);
        var step = score.invoke("set", (Object) null).getStepUsed()
                .intValue() -  baseStep;
        System.out.println("step = " + step);
        assertEquals(stepCost.replaceBase() + hashCost, step);

        // null -> non-null (length 0)
        var ba0 = new byte[0];
        baseStep = score.invoke("emptyBody1", (Object) ba0).getStepUsed()
                .intValue();
        score.invoke("set", (Object) null);
        step = score.invoke("set", (Object) ba0).getStepUsed().intValue()
                - baseStep;
        assertEquals(stepCost.setBase() + hashCost, step);

        // null -> non-null (length 10)
        var ba10 = new byte[10];
        baseStep = score.invoke("emptyBody1", (Object) ba10).getStepUsed()
                .intValue();
        score.invoke("set", (Object) null);
        step = score.invoke("set", (Object) ba10).getStepUsed().intValue()
                - baseStep;
        assertEquals(stepCost.setBase() + ba10.length * stepCost.set()
                + hashCost, step);

        // non-null -> null
        baseStep = score.invoke("emptyBody0").getStepUsed().intValue();
        step = score.invoke("set", (Object) null).getStepUsed().intValue()
                - baseStep;
        assertEquals(stepCost.deleteBase() + ba10.length * stepCost.delete()
                + hashCost, step);
    }

    @Test
    void testReplaceBase() {
        var ba1 = new byte[1];
        score.invoke("set", (Object) ba1);
        var baseStep = score.invoke("emptyBody1", (Object) ba1)
                .getStepUsed().intValue();
        var step = score.invoke("set", (Object) ba1).getStepUsed().intValue()
                - baseStep;
        assertEquals(stepCost.replaceBase()
                + ba1.length * (stepCost.set() + stepCost.delete())
                + hashCost, step);

        var ba_rb = new byte[(int) stepCost.replaceBase()];
        baseStep = score.invoke("emptyBody1", (Object) ba_rb)
                .getStepUsed().intValue();
        step = score.invoke("set", (Object) ba_rb).getStepUsed().intValue()
                - baseStep;
        assertEquals(stepCost.replaceBase()
                + ba_rb.length * stepCost.set()
                + ba1.length * stepCost.delete()
                + hashCost, step);

        var ba_rbPlus1 = new byte[(int) stepCost.replaceBase() + 1];
        baseStep = score.invoke("emptyBody1",
                (Object) ba_rbPlus1).getStepUsed().intValue();
        step = score.invoke("set", (Object) ba_rbPlus1).getStepUsed().intValue()
                - baseStep;
        assertEquals(stepCost.replaceBase()
                + ba_rbPlus1.length * stepCost.set()
                + ba_rb.length * stepCost.delete()
                + hashCost, step);
    }

    @Test
    void testGet() {
        var ba = new byte[10];
        var baseStep = score.invoke("emptyBody1", (Object)ba)
                .getStepUsed().intValue();

        score.invoke("set", (Object)ba);
        var step = score.invoke("get").getStepUsed().intValue()
                - baseStep;
        assertEquals(stepCost.getBase() + stepCost.get()*ba.length +
                hashCost, step);
    }
}
