package foundation.icon.ee.ipc;

import avm.Address;
import foundation.icon.ee.test.GoldenTest;
import foundation.icon.ee.tooling.abi.External;
import foundation.icon.ee.tooling.abi.Optional;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

public class TypeTest extends GoldenTest {
    public static class Score {
        @External
        public static byte mbyte(byte v) {
            return v;
        }

        @External
        public static short mshort(short v) {
            return v;
        }

        @External
        public static int mint(int v) {
            return v;
        }

        @External
        public static long mlong(long v) {
            return v;
        }

        @External
        public static boolean mboolean(boolean v) {
            return v;
        }

        @External
        public static char mchar(char v) {
            return v;
        }

        /*
        @External
        public static Byte mByte(Byte v) {
            return v;
        }

        @External
        public static Short mShort(Short v) {
            return v;
        }

        @External
        public static Integer mInteger(Integer v) {
            return v;
        }

        @External
        public static Long mLong(Long v) {
            return v;
        }

        @External
        public static Boolean mBoolean(Boolean v) {
            return v;
        }

        @External
        public static Character mCharacter(Character v) {
            return v;
        }
         */

        @External
        public static BigInteger mBigInteger(@Optional BigInteger v) {
            return v;
        }

        @External
        public static String mString(@Optional String v) {
            return v;
        }

        @External
        public static byte[] mByteArray(@Optional byte[] v) {
            return v;
        }

        @External
        public static Address mAddress(@Optional Address v) {
            return v;
        }
    }

    @Test
    public void testTypes() {
        var app = sm.deploy(Score.class);
        app.invoke("mbyte", 0);
        app.invoke("mshort", 0);
        app.invoke("mint", 0);
        app.invoke("mlong", (long)0);
        app.invoke("mboolean", false);
        app.invoke("mchar", 0);
        /*
        app.invoke("mByte", 0);
        app.invoke("mShort", 0);
        app.invoke("mInteger", 0);
        app.invoke("mLong", (long)0);
        app.invoke("mBoolean", 0);
        app.invoke("mCharacter", 0);
         */
        app.invoke("mBigInteger", 0);
        app.invoke("mString", "string");
        app.invoke("mByteArray", new byte[]{0, 1, 2});
        app.invoke("mAddress", sm.newExternalAddress());
        /*
        app.invoke("mByte", (Object)null);
        app.invoke("mShort", (Object)null);
        app.invoke("mInteger", (Object)null);
        app.invoke("mLong", (Object)null);
        app.invoke("mBoolean", (Object)null);
        app.invoke("mCharacter", (Object)null);
         */
        app.invoke("mBigInteger", (Object)null);
        app.invoke("mString", (Object)null);
        app.invoke("mByteArray", (Object)null);
        app.invoke("mAddress", (Object)null);
    }
}
