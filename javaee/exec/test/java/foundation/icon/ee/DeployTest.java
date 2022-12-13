package foundation.icon.ee;

import foundation.icon.ee.test.SimpleTest;
import foundation.icon.ee.test.TransactionException;
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.Context;
import score.annotation.External;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

public class DeployTest extends SimpleTest {
    public static class NoConstructor {
    }

    public static class PackagePrivateConstructor {
        PackagePrivateConstructor() {
        }
    }

    public static class ProtectedConstructor {
        protected ProtectedConstructor() {
        }
    }

    public static class PrivateConstructor {
        private PrivateConstructor() {
        }
    }

    public static abstract class Abstract {
    }

    public interface Interface {
    }

    @Test
    public void test() {
        sm.mustDeploy(NoConstructor.class);
        var e = assertThrows(TransactionException.class,
                () -> sm.mustDeploy(PackagePrivateConstructor.class));
        assertEquals(Status.IllegalFormat, e.getResult().getStatus());
        e = assertThrows(TransactionException.class,
                () -> sm.mustDeploy(ProtectedConstructor.class));
        assertEquals(Status.IllegalFormat, e.getResult().getStatus());
        e = assertThrows(TransactionException.class,
                () -> sm.mustDeploy(PrivateConstructor.class));
        assertEquals(Status.IllegalFormat, e.getResult().getStatus());
        e = assertThrows(TransactionException.class,
                () -> sm.mustDeploy(Abstract.class));
        assertEquals(Status.IllegalFormat, e.getResult().getStatus());
        e = assertThrows(TransactionException.class,
                () -> sm.mustDeploy(Interface.class));
        assertEquals(Status.IllegalFormat, e.getResult().getStatus());
    }

    public interface Inf {
        void run();
    }

    public static class ClassAccess {
        @External
        public void run() {
            Context.newVarDB("vdb", Inf.class);
        }
    }

    public static class ArrayClassAccess {
        @External
        public void run() {
            Context.newVarDB("vdb", Inf[].class);
        }
    }

    @Test
    void testClassAccess() {
        Assertions.assertDoesNotThrow(() ->
                sm.mustDeploy(new Class<?>[]{ClassAccess.class, Inf.class})
        );
        Assertions.assertDoesNotThrow(() ->
                sm.mustDeploy(new Class<?>[]{ArrayClassAccess.class, Inf.class})
        );
    }

    public static class ExceptionInConstructor {
        public ExceptionInConstructor() {
            try {
                throw new RuntimeException();
            } catch (RuntimeException e) {
                // ignore
            }
        }
    }

    @Test
    void testExceptionInConstructor() {
        var res = sm.tryDeploy(ExceptionInConstructor.class);
        Assertions.assertEquals(Status.Success, res.getStatus());
    }
}
