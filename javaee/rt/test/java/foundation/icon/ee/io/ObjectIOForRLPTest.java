package foundation.icon.ee.io;

import a.ByteArray;
import foundation.icon.ee.util.Strings;
import i.IInstrumentation;
import i.IObject;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import p.score.Address;
import pi.ObjectReaderImpl;
import pi.ObjectWriterImpl;
import testutils.TestInstrumentation;

import static org.junit.jupiter.api.Assertions.*;

public class ObjectIOForRLPTest {
    static byte[] bytes(String s) {
        return bytes(0, s);
    }

    static byte[] bytes(int l, String s) {
        s = s.replaceAll("\\s+","");
        int len = s.length();
        byte[] data = new byte[Math.max(len / 2, l)];
        for (int i = 0; i < len; i += 2) {
            data[i / 2] = (byte) ((Character.digit(s.charAt(i), 16) << 4)
                    + Character.digit(s.charAt(i+1), 16));
        }
        return data;
    }

    static byte[] bytes(int... elem) {
        var teout = new byte[elem.length];
        for (int i = 0; i < teout.length; i++) {
            teout[i] = (byte) elem[i];
        }
        return teout;
    }

    static byte[] bytesL(int len, int... elem) {
        var teout = new byte[len];
        for (int i = 0; i < elem.length; i++) {
            teout[i] = (byte) elem[i];
        }
        return teout;
    }

    private static String hex(byte[] ba) {
        return Strings.hexFromBytes(ba, " ");
    }

    static void assertAvmObjectEquals(IObject a, IObject b) {
        boolean eq;
        if (a instanceof a.ByteArray) {
            eq = s.java.util.Arrays.avm_equals((ByteArray)a, (ByteArray)b);
        } else {
            eq = a.avm_equals(b);
        }
        assertTrue(eq, "not equal");
    }

    static void testObject(IObject data, byte[] enc) {
        var w = new ObjectWriterImpl(new RLPNDataWriter());
        var r = new ObjectReaderImpl(new RLPNDataReader(enc));
        w.avm_write(data);
        IObject adata = r._read(data.getClass());
        var aenc = w.toByteArray();
        System.out.format("enc:%s => %s dec:%s => %s %n", data.avm_toString(), hex(aenc), hex(enc), adata.avm_toString());
        assertArrayEquals(enc, aenc);
        assertAvmObjectEquals(data, adata);
    }

    static void testBytes(byte[] data, byte[] enc) {
        testObject(new ByteArray(data), enc);
    }

    static void testList(IObject[] data, byte[] enc) {
        var w = new ObjectWriterImpl(new RLPNDataWriter());
        var r = new ObjectReaderImpl(new RLPNDataReader(enc));
        w.avm_beginList(data.length);
        for (IObject e : data)
            w.avm_write(e);
        w.avm_end();
        IObject[] adata = new IObject[data.length];
        r.avm_beginList();
        int i = 0;
        while (r.avm_hasNext()) {
            adata[i] = r._read(data[0].getClass());
            assertAvmObjectEquals(data[i], adata[i]);
            i++;
        }
        r.avm_end();
        var aenc = w.toByteArray();
        assertArrayEquals(enc, aenc);
    }

    @BeforeEach
    public void setup() throws Exception {
        IInstrumentation.attachedThreadInstrumentation.set(new TestInstrumentation());
    }

    @AfterEach
    public void tearDown() throws Exception {
        IInstrumentation.attachedThreadInstrumentation.remove();
    }

    @Test
    public void testSimpleCase() throws Exception {
        testBytes(bytes(), bytes(0x80));
        testBytes(bytes(0x00), bytes(0x00));
        testBytes(bytes(0x7f), bytes(0x7f));
        testBytes(bytes(0x80), bytes(0x81, 0x80));
        testBytes(bytesL(55 ), bytesL(56, 0xb7));
        testBytes(bytesL(56 ), bytesL(58, 0xb8, 56));
        testBytes(bytesL(256 ), bytesL(259, 0xb9, 1, 0));

        testObject(s.java.lang.Boolean.avm_FALSE, bytes("00"));
        testObject(s.java.lang.Boolean.avm_TRUE, bytes("01"));

        testObject(s.java.lang.Byte.avm_valueOf((byte)0x7f), bytes(0x7f));
        testObject(s.java.lang.Byte.avm_valueOf((byte)0x80), bytes(0x81, 0x80));

        testObject(s.java.lang.Short.avm_valueOf((short)0x1), bytes(0x01));
        testObject(s.java.lang.Short.avm_valueOf((short)0x7fff), bytes(0x82, 0x7f, 0xff));
        testObject(s.java.lang.Short.avm_valueOf((short)0x8000), bytes(0x82, 0x80, 0x00));
        testObject(s.java.lang.Short.avm_valueOf((short)0xffff), bytes(0x81, 0xff));

        testObject(s.java.lang.Character.avm_valueOf((char)0x7f), bytes(0x7f));
        testObject(s.java.lang.Character.avm_valueOf((char)0x80), bytes(0x82, 0x00, 0x80));
        testObject(s.java.lang.Character.avm_valueOf((char)0x7fff), bytes(0x82, 0x7f, 0xff));
        testObject(s.java.lang.Character.avm_valueOf((char)0x8000), bytes(0x83, 0x00, 0x80, 0x00));
        testObject(s.java.lang.Character.avm_valueOf((char)0xffff), bytes(0x83, 0x00, 0xff, 0xff));

        testObject(s.java.lang.Integer.avm_valueOf((int)0x1), bytes(0x01));
        testObject(s.java.lang.Integer.avm_valueOf((int)0x7fffffff), bytes(0x84, 0x7f, 0xff, 0xff, 0xff));
        testObject(s.java.lang.Integer.avm_valueOf((int)0x80000000), bytes(0x84, 0x80, 0x00, 0x00, 0x00));
        testObject(s.java.lang.Integer.avm_valueOf((int)0xffffffff), bytes(0x81, 0xff));

        testObject(s.java.lang.Float.avm_valueOf(Float.intBitsToFloat(0x01020304)), bytes(0x84, 1, 2, 3, 4));

        testObject(s.java.lang.Long.avm_valueOf(0x1), bytes(0x01));
        testObject(s.java.lang.Long.avm_valueOf(0x7fffffffffL), bytes(0x85, 0x7f, 0xff, 0xff, 0xff, 0xff));
        testObject(s.java.lang.Long.avm_valueOf(0x8000000000L), bytes(0x86, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00));
        testObject(s.java.lang.Long.avm_valueOf(-1), bytes(0x81, 0xff));

        testObject(new Address(bytesL(21)), bytesL(22, 0x80+21));

        testObject(s.java.math.BigInteger.avm_valueOf(0x1), bytes(0x01));
        testObject(s.java.math.BigInteger.avm_valueOf(0xff), bytes(0x82, 00, 0xff));
        testObject(s.java.math.BigInteger.avm_valueOf(-1), bytes(0x81, 0xff));
        testList(new IObject[]{
                s.java.math.BigInteger.avm_valueOf(1),
                s.java.math.BigInteger.avm_valueOf(2),
                s.java.math.BigInteger.avm_valueOf(3),
        }, bytes(0xc3, 1, 2, 3));
        testList(new IObject[] {
                new ByteArray(bytesL(56, 0))
        }, bytesL(60, 0xf8, 58, 0xb8, 56));
    }
}
