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
        sm.deploy(NoConstructor.class);
        var e = assertThrows(TransactionException.class,
                () -> sm.deploy(PackagePrivateConstructor.class));
        assertEquals(Status.IllegalFormat, e.getResult().getStatus());
        e = assertThrows(TransactionException.class,
                () -> sm.deploy(ProtectedConstructor.class));
        assertEquals(Status.IllegalFormat, e.getResult().getStatus());
        e = assertThrows(TransactionException.class,
                () -> sm.deploy(PrivateConstructor.class));
        assertEquals(Status.IllegalFormat, e.getResult().getStatus());
        e = assertThrows(TransactionException.class,
                () -> sm.deploy(Abstract.class));
        assertEquals(Status.IllegalFormat, e.getResult().getStatus());
        e = assertThrows(TransactionException.class,
                () -> sm.deploy(Interface.class));
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
                sm.deploy(new Class<?>[]{ClassAccess.class, Inf.class})
        );
        Assertions.assertDoesNotThrow(() ->
                sm.deploy(new Class<?>[]{ArrayClassAccess.class, Inf.class})
        );
    }
}
