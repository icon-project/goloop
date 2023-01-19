/*
 * Copyright 2023 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package foundation.icon.ee.io;

import foundation.icon.ee.util.Strings;
import i.IInstrumentation;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import pi.ObjectReaderImpl;
import pi.ObjectWriterImpl;
import score.Address;
import testutils.TestInstrumentation;

import java.math.BigInteger;
import java.util.function.BiConsumer;
import java.util.function.Function;
import java.util.function.Supplier;

public class RLPObjectReaderWriterTest {
    @BeforeEach
    public void setup() throws Exception {
        IInstrumentation.attachedThreadInstrumentation.set(new TestInstrumentation());
    }

    @AfterEach
    public void tearDown() {
        IInstrumentation.attachedThreadInstrumentation.remove();
    }

    static String hex(byte[] ba) {
        return Strings.hexFromBytes(ba);
    }

    static <T, DW extends DataWriter, DR extends DataReader> void test(
            String exp,
            T v,
            Supplier<DW> dwf,
            BiConsumer<ObjectWriterImpl, T> wfn,
            Function<byte[], DR> drf,
            Function<ObjectReaderImpl, T> rfn
    ) {
        var ow = new ObjectWriterImpl(dwf.get());
        wfn.accept(ow, v);
        var ba = ow.avm_toByteArray();
        Assertions.assertEquals(exp, hex(ba.getUnderlying()));
        var or = new ObjectReaderImpl(drf.apply(ba.getUnderlying()));
        Assertions.assertEquals(v, rfn.apply(or));
    }

    static <T, TH extends Throwable, DW extends DataWriter> void testThrows(
            Class<TH> exp,
            T v,
            Supplier<DW> dwf,
            BiConsumer<ObjectWriterImpl, T> wfn
    ) {
        Assertions.assertThrows(exp, () -> {
            var ow = new ObjectWriterImpl(dwf.get());
            wfn.accept(ow, v);
        });
    }

    public static class RLPAssertion implements RLPCodecTest.Assertion {
        public void assertCodingEquals(String exp, boolean v) {
            test(exp, v, RLPDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPDataReader::new, ObjectReaderImpl::avm_readBoolean);
        }

        public void assertCodingEquals(String exp, byte v) {
            test(exp, v, RLPDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPDataReader::new, ObjectReaderImpl::avm_readByte);
        }

        public<TH extends Throwable> void assertWriteThrows(Class<TH> exp, byte v) {
            testThrows(exp, v, RLPDataWriter::new, ObjectWriterImpl::avm_write);
        }

        public void assertCodingEquals(String exp, short v) {
            test(exp, v, RLPDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPDataReader::new, ObjectReaderImpl::avm_readShort);
        }

        public<TH extends Throwable> void assertWriteThrows(Class<TH> exp, short v) {
            testThrows(exp, v, RLPDataWriter::new, ObjectWriterImpl::avm_write);
        }

        public void assertCodingEquals(String exp, char v) {
            test(exp, v, RLPDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPDataReader::new, ObjectReaderImpl::avm_readChar);
        }

        public void assertCodingEquals(String exp, int v) {
            test(exp, v, RLPDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPDataReader::new, ObjectReaderImpl::avm_readInt);
        }

        public<TH extends Throwable> void assertWriteThrows(Class<TH> exp, int v) {
            testThrows(exp, v, RLPDataWriter::new, ObjectWriterImpl::avm_write);
        }

        public void assertCodingEquals(String exp, float v) {
            test(exp, v, RLPDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPDataReader::new, ObjectReaderImpl::avm_readFloat);
        }

        public void assertCodingEquals(String exp, long v) {
            test(exp, v, RLPDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPDataReader::new, ObjectReaderImpl::avm_readLong);
        }

        public<TH extends Throwable> void assertWriteThrows(Class<TH> exp, long v) {
            testThrows(exp, v, RLPDataWriter::new, ObjectWriterImpl::avm_write);
        }

        public void assertCodingEquals(String exp, double v) {
            test(exp, v, RLPDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPDataReader::new, ObjectReaderImpl::avm_readDouble);
        }

        public void assertCodingEquals(String exp, BigInteger v) {
            var ow = new ObjectWriterImpl(new RLPDataWriter());
            ow.avm_write(new s.java.math.BigInteger(v));
            var ba = ow.avm_toByteArray();
            Assertions.assertEquals(exp, hex(ba.getUnderlying()));
            var or = new ObjectReaderImpl(new RLPDataReader(ba.getUnderlying()));
            Assertions.assertEquals(v, or.avm_readBigInteger().getUnderlying());
        }

        public<TH extends Throwable> void assertWriteThrows(Class<TH> exp, BigInteger v) {
            Assertions.assertThrows(exp, () -> {
                var ow = new ObjectWriterImpl(new RLPDataWriter());
                ow.avm_write(new s.java.math.BigInteger(v));
            });
        }

        public void assertCodingEquals(String exp, String v) {
            var ow = new ObjectWriterImpl(new RLPDataWriter());
            ow.avm_write(new s.java.lang.String(v));
            var ba = ow.avm_toByteArray();
            Assertions.assertEquals(exp, hex(ba.getUnderlying()));
            var or = new ObjectReaderImpl(new RLPDataReader(ba.getUnderlying()));
            Assertions.assertEquals(v, or.avm_readString().getUnderlying());
        }

        public void assertCodingEquals(String exp, byte[] v) {
            var ow = new ObjectWriterImpl(new RLPDataWriter());
            ow.avm_write(new a.ByteArray(v));
            var ba = ow.avm_toByteArray();
            Assertions.assertEquals(exp, hex(ba.getUnderlying()));
            var or = new ObjectReaderImpl(new RLPDataReader(ba.getUnderlying()));
            Assertions.assertArrayEquals(v, or.avm_readByteArray().getUnderlying());
        }

        public void assertListCodingEquals(String exp, byte[] v) {
            var ow = new ObjectWriterImpl(new RLPDataWriter());
            if (v==null) {
                ow.avm_beginList(0);
                ow.avm_end();
            } else {
                ow.avm_beginList(1);
                ow.avm_write(new a.ByteArray(v));
                ow.avm_end();
            }
            var ba = ow.avm_toByteArray();
            Assertions.assertEquals(exp, hex(ba.getUnderlying()));
            var or = new ObjectReaderImpl(new RLPDataReader(ba.getUnderlying()));
            or.avm_beginList();
            if (v!=null) {
                Assertions.assertArrayEquals(v, or.avm_readByteArray().getUnderlying());
            }
            or.avm_end();
        }

    }

    public static class RLPNAssertion implements RLPNCodecTest.Assertion {
        public void assertCodingEquals(String exp, boolean v) {
            test(exp, v, RLPNDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPNDataReader::new, ObjectReaderImpl::avm_readBoolean);
        }

        public void assertCodingEquals(String exp, byte v) {
            test(exp, v, RLPNDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPNDataReader::new, ObjectReaderImpl::avm_readByte);
        }

        public void assertCodingEquals(String exp, short v) {
            test(exp, v, RLPNDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPNDataReader::new, ObjectReaderImpl::avm_readShort);
        }

        public void assertCodingEquals(String exp, char v) {
            test(exp, v, RLPNDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPNDataReader::new, ObjectReaderImpl::avm_readChar);
        }

        public void assertCodingEquals(String exp, int v) {
            test(exp, v, RLPNDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPNDataReader::new, ObjectReaderImpl::avm_readInt);
        }

        public void assertCodingEquals(String exp, float v) {
            test(exp, v, RLPNDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPNDataReader::new, ObjectReaderImpl::avm_readFloat);
        }

        public void assertCodingEquals(String exp, long v) {
            test(exp, v, RLPNDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPNDataReader::new, ObjectReaderImpl::avm_readLong);
        }

        public void assertCodingEquals(String exp, double v) {
            test(exp, v, RLPNDataWriter::new, ObjectWriterImpl::avm_write,
                    RLPNDataReader::new, ObjectReaderImpl::avm_readDouble);
        }

        public void assertCodingEquals(String exp, BigInteger v) {
            var ow = new ObjectWriterImpl(new RLPNDataWriter());
            ow.avm_write(new s.java.math.BigInteger(v));
            var ba = ow.avm_toByteArray();
            Assertions.assertEquals(exp, hex(ba.getUnderlying()));
            var or = new ObjectReaderImpl(new RLPNDataReader(ba.getUnderlying()));
            Assertions.assertEquals(v, or.avm_readBigInteger().getUnderlying());
        }

        public void assertCodingEquals(String exp, String v) {
            var ow = new ObjectWriterImpl(new RLPNDataWriter());
            ow.avm_write(new s.java.lang.String(v));
            var ba = ow.avm_toByteArray();
            Assertions.assertEquals(exp, hex(ba.getUnderlying()));
            var or = new ObjectReaderImpl(new RLPNDataReader(ba.getUnderlying()));
            Assertions.assertEquals(v, or.avm_readString().getUnderlying());
        }

        public void assertCodingEquals(String exp, byte[] v) {
            var ow = new ObjectWriterImpl(new RLPNDataWriter());
            ow.avm_write(new a.ByteArray(v));
            var ba = ow.avm_toByteArray();
            Assertions.assertEquals(exp, hex(ba.getUnderlying()));
            var or = new ObjectReaderImpl(new RLPNDataReader(ba.getUnderlying()));
            Assertions.assertArrayEquals(v, or.avm_readByteArray().getUnderlying());
        }

        public void assertListCodingEquals(String exp, byte[] v) {
            var ow = new ObjectWriterImpl(new RLPNDataWriter());
            if (v==null) {
                ow.avm_beginList(0);
                ow.avm_end();
            } else {
                ow.avm_beginList(1);
                ow.avm_write(new a.ByteArray(v));
                ow.avm_end();
            }
            var ba = ow.avm_toByteArray();
            Assertions.assertEquals(exp, hex(ba.getUnderlying()));
            var or = new ObjectReaderImpl(new RLPNDataReader(ba.getUnderlying()));
            or.avm_beginList();
            if (v!=null) {
                Assertions.assertArrayEquals(v, or.avm_readByteArray().getUnderlying());
            }
            or.avm_end();
        }

    }

    static void testCodingEquals(String exp, p.score.Address v, Supplier<DataWriter> dwf, Function<byte[], DataReader> drf) {
        var ow = new ObjectWriterImpl(dwf.get());
        ow.avm_write(v);
        var ba = ow.avm_toByteArray();
        Assertions.assertEquals(exp, hex(ba.getUnderlying()));
        var or = new ObjectReaderImpl(drf.apply(ba.getUnderlying()));
        Assertions.assertEquals(v, or.avm_readAddress());
    }

    @Test
    void testRLPSimple() {
        RLPCodecTest.testRLPSimple(new RLPAssertion());
    }

    @Test
    void testRLPSimple2() {
        testCodingEquals("95" + "00".repeat(21), new p.score.Address(new byte[21]), RLPDataWriter::new, RLPDataReader::new);
    }

    @Test
    void testRLPNSimple() {
        RLPNCodecTest.testRLPNSimple(new RLPNAssertion());
    }

    @Test
    void testRLPNSimple2() {
        testCodingEquals("95" + "00".repeat(21), new p.score.Address(new byte[21]), RLPNDataWriter::new, RLPNDataReader::new);
    }
}
