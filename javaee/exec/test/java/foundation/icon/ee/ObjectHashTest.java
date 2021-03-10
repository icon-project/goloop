package foundation.icon.ee;

import foundation.icon.ee.test.ContractAddress;
import foundation.icon.ee.test.SimpleTest;
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import score.Address;
import score.Context;
import score.RevertedException;
import score.annotation.External;

import java.math.BigInteger;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class ObjectHashTest extends SimpleTest {
    public static class A {
        static final int TEMP_OBJS = 1;
        private Address next;
        private int value = 0;

        @External
        public void setNext(Address addr) {
            next = addr;
        }

        private void createTempObjs() {
            for (int i=0; i<TEMP_OBJS; i++) {
                var ba = new Object();
            }
        }

        @External
        public int getNextHash() {
            return new Object().hashCode();
        }

        @External
        public void doNotChangeGraph() {
            createTempObjs();
        }

        @External
        public void changeGraph() {
            createTempObjs();
            value++;
        }

        @External
        public void changeGraphAndFail() {
            createTempObjs();
            value++;
            Context.revert();
        }

        @External
        public void callDoNotChangeGraph() {
            var args = new Object[] {"doNotChangeGraph"};
            var ba = new Object();
            Context.call(next, "call", args);
            var ba2 = new Object();
            Context.require(ba.hashCode() + 1 == ba2.hashCode());
        }

        @External
        public void callChangeGraph() {
            var args = new Object[] {"changeGraph"};
            var ba = new Object();
            Context.call(next, "call", args);
            var ba2 = new Object();
            Context.require(ba.hashCode() + TEMP_OBJS + 1 ==
                    ba2.hashCode());
        }

        @External
        public void callChangeGraphAndFail() {
            var args = new Object[] {"changeGraphAndFail"};
            var ba = new Object();
            try {
                Context.call(next, "call", args);
            } catch (RevertedException e) {
                Context.require(ba.hashCode() + 1 == e.hashCode());
            }
        }
    }

    public static class B {
        private final Address next;

        public B(Address next) {
            this.next = next;
        }

        @External
        public void call(String method) {
            Context.call(next, method);
        }
    }

    private ContractAddress score;

    @BeforeEach
    public void setUp() {
        super.setUp();
        score = sm.mustDeploy(A.class);
        var score2 = sm.mustDeploy(B.class, score.getAddress());
        score.invoke("setNext", score2.getAddress());
    }

    @Test
    void getNextHashDoesNotChangeNextHash() {
        var res = score.invoke("getNextHash");
        var res2 = score.invoke("getNextHash");
        assertEquals(res.getRet(), res2.getRet());
    }

    @Test
    void nextHashDoesNotChangeIfGraphDoseNotChange() {
        var res = score.invoke("getNextHash");
        score.invoke("doNotChangeGraph");
        var res2 = score.invoke("getNextHash");
        assertEquals(res.getRet(), res2.getRet());
    }

    @Test
    void nextHashDoesNotChangeIfGraphDoseNotChangeInScore() {
        score.invoke("callDoNotChangeGraph");
    }

    @Test
    void nextHashChangesIfGraphChanges() {
        var res = score.invoke("getNextHash");
        score.invoke("changeGraph");
        var res2 = score.invoke("getNextHash");
        var expected = ((BigInteger)res.getRet()).add(
                BigInteger.valueOf(A.TEMP_OBJS)
        );
        assertEquals(expected, res2.getRet());
    }

    @Test
    void nextHashChangesIfGraphChangesInScore() {
        score.invoke("callChangeGraph");
    }

    @Test
    void nextHashDoesNotChangeIfFail() {
        var res = score.invoke("getNextHash");
        var r = score.tryInvoke("changeGraphAndFail");
        assertEquals(Status.UserReversionStart, r.getStatus());
        var res2 = score.invoke("getNextHash");
        assertEquals(res.getRet(), res2.getRet());
    }

    @Test
    void nextHashDoesNotChangeIfFailInScore() {
        score.invoke("callChangeGraphAndFail");
    }
}
