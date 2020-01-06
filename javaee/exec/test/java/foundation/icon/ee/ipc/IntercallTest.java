package foundation.icon.ee.ipc;

import avm.Address;
import avm.Blockchain;
import foundation.icon.ee.test.GoldenTest;
import foundation.icon.ee.tooling.abi.External;
import foundation.icon.ee.tooling.abi.Optional;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

public class IntercallTest extends GoldenTest {
    public static class ProxyScore {
        private static Address next;

        public static void onInstall(Address addr) {
            next = addr;
        }

        @External
        public static byte mbyte(byte v) {
            var vv = (BigInteger)Blockchain.call(next, "mbyte", v);
            return vv.byteValue();
        }

        @External
        public static short mshort(short v) {
            var vv = (BigInteger)Blockchain.call(next, "mshort", v);
            return vv.shortValue();
        }

        @External
        public static int mint(int v) {
            var vv = (BigInteger)Blockchain.call(next, "mint", v);
            return vv.intValue();
        }

        @External
        public static long mlong(long v) {
            var vv = (BigInteger)Blockchain.call(next, "mlong", v);
            return vv.longValue();
        }

        @External
        public static boolean mboolean(boolean v) {
            return (Boolean)Blockchain.call(next, "mboolean", v);
        }

        @External
        public static char mchar(char v) {
            var vv = (BigInteger)Blockchain.call(next, "mchar", v);
            return (char)vv.intValue();
        }

        @External
        public static BigInteger mBigInteger(@Optional BigInteger v) {
            return (BigInteger)Blockchain.call(next, "mBigInteger", v);
        }

        @External
        public static String mString(@Optional String v) {
            return (String)Blockchain.call(next, "mString", v);
        }

        @External
        public static byte[] mByteArray(@Optional byte[] v) {
            return (byte[])Blockchain.call(next, "mByteArray", (Object) v);
        }

        @External
        public static Address mAddress(@Optional Address v) {
            return (Address)Blockchain.call(next, "mAddress", v);
        }

        @External
        public static void mvoid() {
            Blockchain.call(next, "mvoid");
        }
    }

    @Test
    public void testTypes() {
        var papp = sm.deploy(TypeTest.Score.class);
        var app = sm.deploy(ProxyScore.class, papp.getAddress());
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
        public static void method(Address addr) {
            Blockchain.call(addr, "setValue", 1);
            var res = (BigInteger)Blockchain.call(addr, "getValue");
            Blockchain.require(res.intValue()==1);
            try {
                Blockchain.call(addr, "setValueFail", 2);
            } catch (Exception e) {
                Blockchain.println(e.toString());
            }
            res = (BigInteger)Blockchain.call(addr, "getValue");
            Blockchain.require(res.intValue()==1);
        }
    }

    public static class ScoreFail {
        private static int value = 0;

        @External
        public static void setValue(int v) {
            value = v;
        }

        @External
        public static void setValueFail(int v) {
            value = v;
            Blockchain.revert();
        }

        @External
        public static int getValue() {
            return value;
        }
    }

    @Test
    public void testFail() {
        var app1 = sm.deploy(ScoreA.class);
        var app2 = sm.deploy(ScoreFail.class);
        app1.invoke("method", app2.getAddress());
    }
}
