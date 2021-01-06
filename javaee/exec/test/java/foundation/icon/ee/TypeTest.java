package foundation.icon.ee;

import foundation.icon.ee.test.GoldenTest;
import org.junit.jupiter.api.Test;
import score.Address;
import score.annotation.External;
import score.annotation.Optional;

import java.math.BigInteger;

public class TypeTest extends GoldenTest {
    public static class Score {
        @External
        public byte mbyte(byte v) {
            return v;
        }

        @External
        public short mshort(short v) {
            return v;
        }

        @External
        public int mint(int v) {
            return v;
        }

        @External
        public long mlong(long v) {
            return v;
        }

        @External
        public boolean mboolean(boolean v) {
            return v;
        }

        @External
        public char mchar(char v) {
            return v;
        }

        @External
        public BigInteger mBigInteger(@Optional BigInteger v) {
            return v;
        }

        @External
        public String mString(@Optional String v) {
            return v;
        }

        @External
        public byte[] mByteArray(@Optional byte[] v) {
            return v;
        }

        @External
        public Address mAddress(@Optional Address v) {
            return v;
        }

        @External
        public void mvoid() {
        }
    }

    @Test
    public void testTypes() {
        var app = sm.mustDeploy(Score.class);
        app.invoke("mbyte", 0);
        app.invoke("mshort", 0);
        app.invoke("mint", 0);
        app.invoke("mlong", (long)0);
        app.invoke("mboolean", false);
        app.invoke("mchar", 0);
        app.invoke("mBigInteger", 0);
        app.invoke("mString", "string");
        app.invoke("mByteArray", new byte[]{0, 1, 2});
        app.invoke("mAddress", sm.newExternalAddress());
        app.invoke("mBigInteger", (Object)null);
        app.invoke("mString", (Object)null);
        app.invoke("mByteArray", (Object)null);
        app.invoke("mAddress", (Object)null);
        app.invoke("mvoid");
    }
}
