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
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.util.function.BiConsumer;
import java.util.function.Function;
import java.util.function.Supplier;

public class RLPDataReaderWriterTest {
    static String hex(byte[] ba) {
        return Strings.hexFromBytes(ba);
    }

    static <T, DW extends DataWriter, DR extends DataReader> void test(
            String exp,
            T v,
            Supplier<DW> dwf,
            BiConsumer<DW, T> wfn,
            Function<byte[], DR> drf,
            Function<DR, T> rfn
    ) {
        var dw = dwf.get();
        wfn.accept(dw, v);
        var ba = dw.toByteArray();
        Assertions.assertEquals(exp, hex(ba));
        var dr = drf.apply(ba);
        Assertions.assertEquals(v, rfn.apply(dr));
    }

    static <DW extends DataWriter, DR extends DataReader> void testByteArray(
            String exp,
            byte[] v,
            Supplier<DW> dwf,
            BiConsumer<DW, byte[]> wfn,
            Function<byte[], DR> drf,
            Function<DR, byte[]> rfn
    ) {
        var dw = dwf.get();
        wfn.accept(dw, v);
        var ba = dw.toByteArray();
        Assertions.assertEquals(exp, hex(ba));
        var dr = drf.apply(ba);
        Assertions.assertArrayEquals(v, rfn.apply(dr));
    }

    static <DW extends DataWriter, DR extends DataReader> void testListOfByteArray(
            String exp,
            byte[] v,
            Supplier<DW> dwf,
            BiConsumer<DW, byte[]> wfn,
            Function<byte[], DR> drf,
            Function<DR, byte[]> rfn
    ) {
        var dw = dwf.get();
        if (v == null) {
            dw.writeListHeader(0);
            dw.writeFooter();
        } else {
            dw.writeListHeader(1);
            wfn.accept(dw, v);
            dw.writeFooter();
        }
        var ba = dw.toByteArray();
        Assertions.assertEquals(exp, hex(ba));
        var dr = drf.apply(ba);
        dr.readListHeader();
        if (v != null) {
            Assertions.assertArrayEquals(v, rfn.apply(dr));
        }
        dr.readFooter();
    }

    static <T, TH extends Throwable, DW extends DataWriter> void testThrows(
            Class<TH> exp,
            T v,
            Supplier<DW> dwf,
            BiConsumer<DW, T> wfn
    ) {
        Assertions.assertThrows(exp, () -> {
            var dw = dwf.get();
            wfn.accept(dw, v);
        });
    }

    public static class RLPAssertion implements RLPCodecTest.Assertion {
        public void assertCodingEquals(String exp, boolean v) {
            test(exp, v, RLPDataWriter::new, RLPDataWriter::write,
                    RLPDataReader::new, RLPDataReader::readBoolean);
        }

        public void assertCodingEquals(String exp, byte v) {
            test(exp, v, RLPDataWriter::new, RLPDataWriter::write,
                    RLPDataReader::new, RLPDataReader::readByte);
        }

        public <TH extends Throwable> void assertWriteThrows(Class<TH> exp, byte v) {
            testThrows(exp, v, RLPDataWriter::new, RLPDataWriter::write);
        }

        public void assertCodingEquals(String exp, short v) {
            test(exp, v, RLPDataWriter::new, RLPDataWriter::write,
                    RLPDataReader::new, RLPDataReader::readShort);
        }

        public <TH extends Throwable> void assertWriteThrows(Class<TH> exp, short v) {
            testThrows(exp, v, RLPDataWriter::new, RLPDataWriter::write);
        }

        public void assertCodingEquals(String exp, char v) {
            test(exp, v, RLPDataWriter::new, RLPDataWriter::write,
                    RLPDataReader::new, RLPDataReader::readChar);
        }

        public void assertCodingEquals(String exp, int v) {
            test(exp, v, RLPDataWriter::new, RLPDataWriter::write,
                    RLPDataReader::new, RLPDataReader::readInt);
        }

        public <TH extends Throwable> void assertWriteThrows(Class<TH> exp, int v) {
            testThrows(exp, v, RLPDataWriter::new, RLPDataWriter::write);
        }

        public void assertCodingEquals(String exp, float v) {
            test(exp, v, RLPDataWriter::new, RLPDataWriter::write,
                    RLPDataReader::new, RLPDataReader::readFloat);
        }

        public void assertCodingEquals(String exp, long v) {
            test(exp, v, RLPDataWriter::new, RLPDataWriter::write,
                    RLPDataReader::new, RLPDataReader::readLong);
        }

        public <TH extends Throwable> void assertWriteThrows(Class<TH> exp, long v) {
            testThrows(exp, v, RLPDataWriter::new, RLPDataWriter::write);
        }

        public void assertCodingEquals(String exp, double v) {
            test(exp, v, RLPDataWriter::new, RLPDataWriter::write,
                    RLPDataReader::new, RLPDataReader::readDouble);
        }

        public void assertCodingEquals(String exp, BigInteger v) {
            test(exp, v, RLPDataWriter::new, RLPDataWriter::write,
                    RLPDataReader::new, RLPDataReader::readBigInteger);
        }

        public <TH extends Throwable> void assertWriteThrows(Class<TH> exp, BigInteger v) {
            testThrows(exp, v, RLPDataWriter::new, RLPDataWriter::write);
        }

        public void assertCodingEquals(String exp, String v) {
            test(exp, v, RLPDataWriter::new, RLPDataWriter::write,
                    RLPDataReader::new, RLPDataReader::readString);
        }

        public void assertCodingEquals(String exp, byte[] v) {
            testByteArray(exp, v, RLPDataWriter::new, RLPDataWriter::write,
                    RLPDataReader::new, RLPDataReader::readByteArray);
        }

        public void assertListCodingEquals(String exp, byte[] v) {
            testListOfByteArray(exp, v, RLPDataWriter::new, RLPDataWriter::write,
                    RLPDataReader::new, RLPDataReader::readByteArray);
        }

    }

    public static class RLPNAssertion implements RLPNCodecTest.Assertion {
        public void assertCodingEquals(String exp, boolean v) {
            test(exp, v, RLPNDataWriter::new, RLPNDataWriter::write,
                    RLPNDataReader::new, RLPNDataReader::readBoolean);
        }

        public void assertCodingEquals(String exp, byte v) {
            test(exp, v, RLPNDataWriter::new, RLPNDataWriter::write,
                    RLPNDataReader::new, RLPNDataReader::readByte);
        }

        public void assertCodingEquals(String exp, short v) {
            test(exp, v, RLPNDataWriter::new, RLPNDataWriter::write,
                    RLPNDataReader::new, RLPNDataReader::readShort);
        }

        public void assertCodingEquals(String exp, char v) {
            test(exp, v, RLPNDataWriter::new, RLPNDataWriter::write,
                    RLPNDataReader::new, RLPNDataReader::readChar);
        }

        public void assertCodingEquals(String exp, int v) {
            test(exp, v, RLPNDataWriter::new, RLPNDataWriter::write,
                    RLPNDataReader::new, RLPNDataReader::readInt);
        }

        public void assertCodingEquals(String exp, float v) {
            test(exp, v, RLPNDataWriter::new, RLPNDataWriter::write,
                    RLPNDataReader::new, RLPNDataReader::readFloat);
        }

        public void assertCodingEquals(String exp, long v) {
            test(exp, v, RLPNDataWriter::new, RLPNDataWriter::write,
                    RLPNDataReader::new, RLPNDataReader::readLong);
        }

        public void assertCodingEquals(String exp, double v) {
            test(exp, v, RLPNDataWriter::new, RLPNDataWriter::write,
                    RLPNDataReader::new, RLPNDataReader::readDouble);
        }

        public void assertCodingEquals(String exp, BigInteger v) {
            test(exp, v, RLPNDataWriter::new, RLPNDataWriter::write,
                    RLPNDataReader::new, RLPNDataReader::readBigInteger);
        }

        public void assertCodingEquals(String exp, String v) {
            test(exp, v, RLPNDataWriter::new, RLPNDataWriter::write,
                    RLPNDataReader::new, RLPNDataReader::readString);
        }

        public void assertCodingEquals(String exp, byte[] v) {
            testByteArray(exp, v, RLPNDataWriter::new, RLPNDataWriter::write,
                    RLPNDataReader::new, RLPNDataReader::readByteArray);
        }

        public void assertListCodingEquals(String exp, byte[] v) {
            testListOfByteArray(exp, v, RLPNDataWriter::new, RLPNDataWriter::write,
                    RLPNDataReader::new, RLPNDataReader::readByteArray);
        }
    }

    @Test
    void testRLPNSimple() {
        RLPNCodecTest.testRLPNSimple(new RLPNAssertion());
    }

    @Test
    void testRLPNNullity() {
        var dw = new RLPNDataWriter();
        dw.writeNullity(true);
        var ba = dw.toByteArray();
        Assertions.assertEquals("f800", hex(ba));
        var dr = new RLPNDataReader(ba);
        Assertions.assertEquals(true, dr.readNullity());
    }

    @Test
    void testRLPSimple() {
        RLPCodecTest.testRLPSimple(new RLPAssertion());
    }

    @Test
    void testRLPNullity() {
        Assertions.assertThrows(UnsupportedOperationException.class, () -> {
            var dw = new RLPDataWriter();
            dw.writeNullity(true);
        });
    }
}
