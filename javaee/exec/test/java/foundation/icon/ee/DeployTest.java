package foundation.icon.ee;

import foundation.icon.ee.test.GoldenTest;
import foundation.icon.ee.test.TransactionException;
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

public class DeployTest extends GoldenTest {
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
        assertEquals(e.getResult().getStatus(), Status.IllegalFormat);
        e = assertThrows(TransactionException.class,
                () -> sm.deploy(ProtectedConstructor.class));
        assertEquals(e.getResult().getStatus(), Status.IllegalFormat);
        e = assertThrows(TransactionException.class,
                () -> sm.deploy(PrivateConstructor.class));
        assertEquals(e.getResult().getStatus(), Status.IllegalFormat);
        e = assertThrows(TransactionException.class,
                () -> sm.deploy(Abstract.class));
        assertEquals(e.getResult().getStatus(), Status.IllegalFormat);
        e = assertThrows(TransactionException.class,
                () -> sm.deploy(Interface.class));
        assertEquals(e.getResult().getStatus(), Status.IllegalFormat);
    }
}
