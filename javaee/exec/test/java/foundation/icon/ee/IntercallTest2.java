package foundation.icon.ee;

import foundation.icon.ee.test.SimpleTest;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.Address;
import score.Context;
import score.annotation.External;
import score.annotation.Payable;

import java.math.BigInteger;

public class IntercallTest2 extends SimpleTest {
    public static class BadParam {
        @External
        public void run() {
            try {
                Context.call(Context.getAddress(), "test", new Object());
            } catch (Exception ignored) {
            }
        }
    }

    @Test
    void testBadParam() {
        var c = sm.mustDeploy(BadParam.class);
        var res = c.invoke("run");
        Assertions.assertEquals(0, res.getStatus());
    }

    public static class Callee {
        @Payable
        @External
        public BigInteger mBigInteger(BigInteger v) {
            return v;
        }

        @Payable
        @External
        public Address mAddress(Address v) {
            return v;
        }
    }

    public static class Caller {
        private final Address addr;

        public Caller(Address addr) {
            this.addr = addr;
        }

        @Payable
        @External
        public void run(BigInteger v) {
            BigInteger v1 = (BigInteger) Context.call(addr, "mBigInteger", v);
            Context.require(v.equals(v1));

            BigInteger v2 = (BigInteger) Context.call(Context.getValue(), addr, "mBigInteger", v);
            Context.require(v.equals(v2));

            BigInteger v3 = Context.call(BigInteger.class, addr, "mBigInteger", v);
            Context.require(v.equals(v3));

            BigInteger v4 = Context.call(BigInteger.class, Context.getValue(), addr, "mBigInteger", v);
            Context.require(v.equals(v4));
        }

        @Payable
        @External
        public void run2(Address v) {
            Address v1 = (Address) Context.call(addr, "mAddress", v);
            Context.require(v.equals(v1));

            Address v2 = (Address) Context.call(Context.getValue(), addr, "mAddress", v);
            Context.require(v.equals(v2));

            Address v3 = Context.call(Address.class, addr, "mAddress", v);
            Context.require(v.equals(v3));

            Address v4 = Context.call(Address.class, Context.getValue(), addr, "mAddress", v);
            Context.require(v.equals(v4));
        }
    }

    @Test
    void testIntercall() {
        var callee = sm.mustDeploy(Callee.class);
        var caller = sm.mustDeploy(Caller.class, callee.getAddress());
        var res = caller.invoke(BigInteger.TEN, sm.getStepLimit(), "run", BigInteger.valueOf(101));
        Assertions.assertEquals(0, res.getStatus());
        var res2 = caller.invoke(BigInteger.TEN, sm.getStepLimit(), "run2", caller.getAddress());
        Assertions.assertEquals(0, res2.getStatus());
    }
}
