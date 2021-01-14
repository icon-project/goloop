package foundation.icon.ee;

import foundation.icon.ee.test.GoldenTest;
import org.junit.jupiter.api.Test;
import score.Address;
import score.Context;
import score.annotation.External;
import score.annotation.Optional;

import java.math.BigInteger;

public class IntercallTest extends GoldenTest {
    public static class ProxyScore {
        private Address next;

        public ProxyScore(Address addr) {
            next = addr;
        }

        @External
        public byte mbyte(byte v) {
            var vv = (BigInteger) Context.call(next, "mbyte", v);
            return vv.byteValue();
        }

        @External
        public short mshort(short v) {
            var vv = (BigInteger) Context.call(next, "mshort", v);
            return vv.shortValue();
        }

        @External
        public int mint(int v) {
            var vv = (BigInteger) Context.call(next, "mint", v);
            return vv.intValue();
        }

        @External
        public long mlong(long v) {
            var vv = (BigInteger) Context.call(next, "mlong", v);
            return vv.longValue();
        }

        @External
        public boolean mboolean(boolean v) {
            return (Boolean) Context.call(next, "mboolean", v);
        }

        @External
        public char mchar(char v) {
            var vv = (BigInteger) Context.call(next, "mchar", v);
            return (char)vv.intValue();
        }

        @External
        public BigInteger mBigInteger(@Optional BigInteger v) {
            return (BigInteger) Context.call(next, "mBigInteger", v);
        }

        @External
        public String mString(@Optional String v) {
            return (String) Context.call(next, "mString", v);
        }

        @External
        public byte[] mByteArray(@Optional byte[] v) {
            return (byte[]) Context.call(next, "mByteArray", (Object) v);
        }

        @External
        public Address mAddress(@Optional Address v) {
            return (Address) Context.call(next, "mAddress", v);
        }

        @External
        public void mvoid() {
            Context.call(next, "mvoid");
        }
    }

    @Test
    public void testTypes() {
        var papp = sm.mustDeploy(TypeTest.Score.class);
        var app = sm.mustDeploy(ProxyScore.class, papp.getAddress());
        app.invoke("mbyte", 0);
        app.invoke("mshort", 0);
        app.invoke("mint", 0);
        app.invoke("mlong", (long)0);
        app.invoke("mboolean", false);
        app.invoke("mchar", 0);
        app.invoke("mBigInteger", 0);
        app.invoke("mString", "string");
        app.invoke("mByteArray", (Object) new byte[]{0, 1, 2});
        app.invoke("mAddress", sm.newExternalAddress());
        app.invoke("mBigInteger", (Object)null);
        app.invoke("mString", (Object)null);
        app.invoke("mByteArray", (Object)null);
        app.invoke("mAddress", (Object)null);
        app.invoke("mvoid");
    }

    public static class ScoreA {
        @External
        public void method(Address addr) {
            Context.call(addr, "setValue", 1);
            var res = (BigInteger) Context.call(addr, "getValue");
            Context.require(res.intValue()==1);
            try {
                Context.call(addr, "setValueFail", 2);
            } catch (Exception e) {
                Context.println(e.toString());
            }
            res = (BigInteger) Context.call(addr, "getValue");
            Context.require(res.intValue()==1);
        }
    }

    public static class ScoreFail {
        private int value = 0;

        @External
        public void setValue(int v) {
            value = v;
        }

        @External
        public void setValueFail(int v) {
            value = v;
            Context.revert();
        }

        @External
        public int getValue() {
            return value;
        }
    }

    @Test
    public void testFail() {
        var app1 = sm.mustDeploy(ScoreA.class);
        var app2 = sm.mustDeploy(ScoreFail.class);
        app1.invoke("method", app2.getAddress());
    }
}
