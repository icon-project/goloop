package foundation.icon.ee;

import foundation.icon.ee.test.ContractAddress;
import foundation.icon.ee.test.SimpleTest;
import foundation.icon.ee.test.TransactionException;
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import score.Address;
import score.Context;
import score.annotation.EventLog;
import score.annotation.External;

import java.math.BigInteger;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

public class ReadOnlyTest extends SimpleTest {
    public static class Score {
        private int intVal;
        private String sVal = "string";

        @External(readonly=true)
        public int setDB() {
            var db = Context.newVarDB("varDB", String.class);
            db.set(sVal);
            return 0;
        }

        @External(readonly=true)
        public int logEvent() {
            Log();
            return 0;
        }

        @EventLog
        private void Log() {
        }

        @External(readonly=true)
        public int changeIntVal() {
            intVal = 10;
            return 0;
        }

        @External(readonly=true)
        public int changeSVal() {
            sVal = "string2";
            return 0;
        }

        @External(readonly=true)
        public int createTempObject() {
            Object obj = new Object();
            return obj.hashCode();
        }
    }

    public static class ProxyScore {
        Address real;

        public ProxyScore(Address real) {
            this.real = real;
        }

        @External
        public int setDB() {
            return ((BigInteger) Context.call(real, "setDB")).intValue();
        }

        @External
        public int logEvent() {
            return ((BigInteger) Context.call(real, "logEvent")).intValue();
        }

        @External
        public int changeIntVal() {
            return ((BigInteger) Context.call(real, "changeIntVal")).intValue();
        }

        @External
        public int changeSVal() {
            return ((BigInteger) Context.call(real, "changeSVal")).intValue();
        }

        @External
        public int createTempObject() {
            return ((BigInteger) Context.call(real, "createTempObject")).intValue();
        }
    }

    private ContractAddress score;
    private TransactionException e;

    @Nested
    class Direct {
        @BeforeEach
        void setUp() {
            score = sm.mustDeploy(Score.class);
        }

        @Test
        void setFailsInReadOnly() {
            e = assertThrows(TransactionException.class,
                    () -> score.query("setDB"));
            assertEquals(Status.UnknownFailure, e.getResult().getStatus());
        }

        @Test
        void logEventFailsInReadOnly() {
            e = assertThrows(TransactionException.class,
                    () -> score.query("logEvent"));
            assertEquals(Status.UnknownFailure, e.getResult().getStatus());
        }

        @Test
        void changeIntValFailsInReadOnly() {
            e = assertThrows(TransactionException.class,
                    () -> score.query("changeIntVal"));
            assertEquals(Status.AccessDenied, e.getResult().getStatus());
        }

        @Test
        void changeSValFailsInReadOnly() {
            e = assertThrows(TransactionException.class,
                    () -> score.query("changeSVal"));
            assertEquals(Status.AccessDenied, e.getResult().getStatus());
        }

        @Test
        void creatingTempObjectSucceeds() {
            score.invoke("createTempObject");
            score.query("createTempObject");
        }
    }

    @Nested
    class Indirect {
        @BeforeEach
        void setUp() {
            var real = sm.mustDeploy(Score.class);
            score = sm.mustDeploy(ProxyScore.class, real.getAddress());
        }

        @Test
        void setFailsInReadOnly() {
            e = assertThrows(TransactionException.class,
                    () -> score.invoke("setDB"));
            assertEquals(Status.UnknownFailure, e.getResult().getStatus());
        }

        @Test
        void logEventFailsInReadOnly() {
            e = assertThrows(TransactionException.class,
                    () -> score.invoke("logEvent"));
            assertEquals(Status.UnknownFailure, e.getResult().getStatus());
        }

        @Test
        void changeIntValFailsInReadOnly() {
            e = assertThrows(TransactionException.class,
                    () -> score.invoke("changeIntVal"));
            assertEquals(Status.UnknownFailure, e.getResult().getStatus());
        }

        @Test
        void changeSValFailsInReadOnly() {
            e = assertThrows(TransactionException.class,
                    () -> score.invoke("changeSVal"));
            assertEquals(Status.UnknownFailure, e.getResult().getStatus());
        }

        @Test
        void creatingTempObjectSucceeds() {
            score.invoke("createTempObject");
        }
    }
}
