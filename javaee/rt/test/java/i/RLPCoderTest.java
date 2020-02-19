package i;

import a.ByteArray;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;
import testutils.TestInstrumentation;
import s.java.lang.String;
import s.java.lang.Byte;
import s.java.lang.Short;
import s.java.lang.Integer;
import s.java.lang.Long;
import s.java.lang.Character;
import s.java.math.BigInteger;
import p.score.Address;

import static org.junit.Assert.*;

public class RLPCoderTest {

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

    static abstract class Case {
        public ByteArray eout;

        Case(byte[] eout) {
            this.eout = new ByteArray(eout);
        }

        abstract void run() throws Exception;
    }

    static class IntCase extends Case {
        public int in;

        IntCase(int in, byte[] eout) {
            super(eout);
            this.in = in;
        }

        void run() throws Exception {
            var rc = new RLPCoder();
            rc.encode(in);
            System.out.println("test in:" + in + " exp:" + eout + " act:" + new ByteArray(rc.toByteArray()));
            assertArrayEquals(eout.getUnderlying(), rc.toByteArray());
        }
    }

    static class ObjectCase extends Case {
        public IObject in;

        ObjectCase(IObject in, byte[] eout) {
            super(eout);
            this.in = in;
        }

        ObjectCase(byte[] in, byte[] eout) {
            super(eout);
            this.in = new ByteArray(in);
        }

        void run() throws Exception {
            var rc = new RLPCoder();
            rc.encode(in);
            System.out.println("test in:" + in + " exp:" + eout + " act:" + new ByteArray(rc.toByteArray()));
            assertArrayEquals(eout.getUnderlying(), rc.toByteArray());
        }
    }


    @Before
    public void setup() throws Exception {
        IInstrumentation.attachedThreadInstrumentation.set(new TestInstrumentation());
    }

    @After
    public void tearDown() throws Exception {
        IInstrumentation.attachedThreadInstrumentation.remove();
    }

    @Test
    public void testSimpleCase() throws Exception {
        Case[] cases = new Case[]{
                new ObjectCase(bytes(), bytes(0x80)),
                new ObjectCase(bytes(0x00), bytes(0x00)),
                new ObjectCase(bytes(0x7f), bytes(0x7f)),
                new ObjectCase(bytes(0x80), bytes(0x81, 0x80)),
                new ObjectCase(bytesL(55 ), bytesL(56, 0xb7)),
                new ObjectCase(bytesL(56 ), bytesL(58, 0xb8, 56)),
                new ObjectCase(bytesL(256 ), bytesL(259, 0xb9, 1, 0)),

                new IntCase(0, bytes(0x00)),
                new IntCase(0x7f, bytes(0x7f)),
                new IntCase(0x80, bytes(0x82, 0x00, 0x80)),
                new IntCase(0x7fff, bytes(0x82, 0x7f, 0xff)),
                new IntCase(0x8000, bytes(0x83, 0x00, 0x80, 0x00)),
                new IntCase( 0x7fffff, bytes(0x83, 0x7f, 0xff, 0xff)),
                new IntCase( 0x800000, bytes(0x84, 0x00, 0x80, 0x00, 0x00)),
                new IntCase( 0x7fffffff, bytes(0x84, 0x7f, 0xff, 0xff, 0xff)),

                new IntCase( 0xffffffff, bytes(0x81, 0xff)),
                new IntCase( 0xffffff80, bytes(0x81, 0x80)),
                new IntCase( 0xffffff7f, bytes(0x82, 0xff, 0x7f)),
                new IntCase( 0xffff8000, bytes(0x82, 0x80, 0x00)),
                new IntCase( 0xffff7fff, bytes(0x83, 0xff, 0x7f, 0xff)),
                new IntCase( 0xff800000, bytes(0x83, 0x80, 0x00, 0x00)),
                new IntCase( 0xff7fffff, bytes(0x84, 0xff, 0x7f, 0xff, 0xff)),
                new IntCase( 0x80000000, bytes(0x84, 0x80, 0x00, 0x00, 0x00)),

                new ObjectCase(new String(""), bytes(0x80)),

                new ObjectCase(Byte.avm_valueOf((byte)0x7f), bytes(0x7f)),
                new ObjectCase(Byte.avm_valueOf((byte)0x80), bytes(0x81, 0x80)),

                new ObjectCase(Short.avm_valueOf((short)0x1), bytes(0x01)),
                new ObjectCase(Short.avm_valueOf((short)0x7fff), bytes(0x82, 0x7f, 0xff)),
                new ObjectCase(Short.avm_valueOf((short)0x8000), bytes(0x82, 0x80, 0x00)),
                new ObjectCase(Short.avm_valueOf((short)0xffff), bytes(0x81, 0xff)),

                new ObjectCase(Integer.avm_valueOf((int)0x1), bytes(0x01)),
                new ObjectCase(Integer.avm_valueOf((int)0x7fffffff), bytes(0x84, 0x7f, 0xff, 0xff, 0xff)),
                new ObjectCase(Integer.avm_valueOf((int)0x80000000), bytes(0x84, 0x80, 0x00, 0x00, 0x00)),
                new ObjectCase(Integer.avm_valueOf((int)0xffffffff), bytes(0x81, 0xff)),

                new ObjectCase(Long.avm_valueOf(0x1), bytes(0x01)),
                new ObjectCase(Long.avm_valueOf(0x7fffffffffL), bytes(0x85, 0x7f, 0xff, 0xff, 0xff, 0xff)),
                new ObjectCase(Long.avm_valueOf(0x8000000000L), bytes(0x86, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00)),
                new ObjectCase(Long.avm_valueOf(-1), bytes(0x81, 0xff)),


                new ObjectCase(Character.avm_valueOf((char)0x7f), bytes(0x7f)),
                new ObjectCase(Character.avm_valueOf((char)0x80), bytes(0x82, 0x00, 0x80)),
                new ObjectCase(Character.avm_valueOf((char)0x7fff), bytes(0x82, 0x7f, 0xff)),
                new ObjectCase(Character.avm_valueOf((char)0x8000), bytes(0x83, 0x00, 0x80, 0x00)),
                new ObjectCase(Character.avm_valueOf((char)0xffff), bytes(0x83, 0x00, 0xff, 0xff)),

                new ObjectCase(new Address(bytesL(21)), bytesL(22, 0x80+21)),

                new ObjectCase(BigInteger.avm_valueOf(0x1), bytes(0x01)),
                new ObjectCase(BigInteger.avm_valueOf(0xff), bytes(0x82, 00, 0xff)),
                new ObjectCase(BigInteger.avm_valueOf(-1), bytes(0x81, 0xff)),
        };
        for (var c : cases)
            c.run();
    }
}
