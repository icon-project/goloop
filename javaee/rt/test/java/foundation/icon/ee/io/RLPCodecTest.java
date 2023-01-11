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

import java.math.BigInteger;

public class RLPCodecTest {
    public interface Assertion {
        void assertCodingEquals(String exp, boolean v);

        void assertCodingEquals(String exp, byte v);

        <TH extends Throwable> void assertWriteThrows(Class<TH> exp, byte v);

        void assertCodingEquals(String exp, short v);

        <TH extends Throwable> void assertWriteThrows(Class<TH> exp, short v);

        void assertCodingEquals(String exp, char v);

        void assertCodingEquals(String exp, int v);

        <TH extends Throwable> void assertWriteThrows(Class<TH> exp, int v);

        void assertCodingEquals(String exp, float v);

        void assertCodingEquals(String exp, long v);

        <TH extends Throwable> void assertWriteThrows(Class<TH> exp, long v);

        void assertCodingEquals(String exp, double v);

        void assertCodingEquals(String exp, BigInteger v);

        <TH extends Throwable> void assertWriteThrows(Class<TH> exp, BigInteger v);

        void assertCodingEquals(String exp, String v);

        void assertCodingEquals(String exp, byte[] v);

        void assertListCodingEquals(String exp, byte[] v);
    }

    public static String repeat(String s, int n) {
        var sb = new StringBuilder();
        for (int i = 0; i < n; i++) {
            sb.append(s);
        }
        return sb.toString();
    }

    public static void testRLPSimpleSmall(Assertion a) {
        a.assertCodingEquals("80", false);
        a.assertCodingEquals("01", true);

        a.assertCodingEquals("80", (byte) 0);
        a.assertCodingEquals("01", (byte) 1);
        a.assertCodingEquals("79", (byte) 0x79);
        a.assertWriteThrows(IllegalArgumentException.class, (byte) 0x80);
        a.assertWriteThrows(IllegalArgumentException.class, (byte) 0xff);

        a.assertCodingEquals("80", (short) 0);
        a.assertCodingEquals("01", (short) 1);
        a.assertCodingEquals("79", (short) 0x79);
        a.assertCodingEquals("8180", (short) 0x80);
        a.assertCodingEquals("81ff", (short) 0xff);
        a.assertCodingEquals("820100", (short) 0x100);
        a.assertCodingEquals("827fff", (short) 0x7fff);
        a.assertWriteThrows(IllegalArgumentException.class, (short) 0x8000);
        a.assertWriteThrows(IllegalArgumentException.class, (short) 0xffff);

        a.assertCodingEquals("80", (char) 0);
        a.assertCodingEquals("01", (char) 1);
        a.assertCodingEquals("79", (char) 0x79);
        a.assertCodingEquals("8180", (char) 0x80);
        a.assertCodingEquals("81ff", (char) 0xff);
        a.assertCodingEquals("820100", (char) 0x100);
        a.assertCodingEquals("82ffff", (char) 0xffff);

        a.assertCodingEquals("80", 0);
        a.assertCodingEquals("01", 1);
        a.assertCodingEquals("79", 0x79);
        a.assertCodingEquals("8180", 0x80);
        a.assertCodingEquals("81ff", 0xff);
        a.assertCodingEquals("820100", 0x100);
        a.assertCodingEquals("82ffff", 0xffff);
        a.assertCodingEquals("83010000", 0x010000);
        a.assertCodingEquals("83ffffff", 0xffffff);
        a.assertCodingEquals("8401000000", 0x01000000);
        a.assertCodingEquals("847fffffff", 0x7fffffff);
        a.assertWriteThrows(IllegalArgumentException.class, 0x80000000);
        a.assertWriteThrows(IllegalArgumentException.class, 0xffffffff);

        a.assertCodingEquals("8401020304", Float.intBitsToFloat(0x01020304));

        a.assertCodingEquals("80", 0);
        a.assertCodingEquals("01", 1);
        a.assertCodingEquals("79", 0x79);
        a.assertCodingEquals("8180", 0x80);
        a.assertCodingEquals("81ff", 0xff);
        a.assertCodingEquals("820100", 0x100);
        a.assertCodingEquals("82ffff", 0xffff);
        a.assertCodingEquals("83010000", 0x010000);
        a.assertCodingEquals("83ffffff", 0xffffff);
        a.assertCodingEquals("8401000000", 0x01000000);
        a.assertCodingEquals("84ffffffff", 0xffffffffL);
        a.assertCodingEquals("850100000000", 0x0100000000L);
        a.assertCodingEquals("85ffffffffff", 0xffffffffffL);
        a.assertCodingEquals("86010000000000", 0x010000000000L);
        a.assertCodingEquals("86ffffffffffff", 0xffffffffffffL);
        a.assertCodingEquals("8701000000000000", 0x01000000000000L);
        a.assertCodingEquals("87ffffffffffffff", 0xffffffffffffffL);
        a.assertCodingEquals("880100000000000000", 0x0100000000000000L);
        a.assertCodingEquals("887fffffffffffffff", 0x7fffffffffffffffL);
        a.assertWriteThrows(IllegalArgumentException.class, 0x8000000000000000L);
        a.assertWriteThrows(IllegalArgumentException.class, 0xffffffffffffffffL);

        a.assertCodingEquals("880102030405060708", Double.longBitsToDouble(0x0102030405060708L));

        a.assertCodingEquals("80", BigInteger.valueOf(0));
        a.assertCodingEquals("01", BigInteger.valueOf(1));
        a.assertCodingEquals("8180", BigInteger.valueOf(0x80));
        a.assertCodingEquals("b701" + repeat("00", 54), new BigInteger("01" + repeat("00", 54), 16));
        a.assertWriteThrows(IllegalArgumentException.class, BigInteger.valueOf(-1));

        a.assertCodingEquals("80", "");
        a.assertCodingEquals("40", "@");
        a.assertCodingEquals("824040", "@@");

        a.assertCodingEquals("80", new byte[0]);
        a.assertCodingEquals("00", new byte[1]);
        a.assertCodingEquals("b7" + repeat("00", 55), new byte[55]);
        a.assertCodingEquals("b838" + repeat("00", 56), new byte[56]);
        a.assertCodingEquals("b8ff" + repeat("00", 255), new byte[255]);
        a.assertCodingEquals("b90100" + repeat("00", 256), new byte[256]);
        a.assertCodingEquals("ba010000" + repeat("00", 1 << 16), new byte[1 << 16]);

        a.assertListCodingEquals("c0", null);
        a.assertListCodingEquals("c100", new byte[1]);
        a.assertListCodingEquals("c3820000", new byte[2]);
        a.assertListCodingEquals("f7b6" + repeat("00", 54), new byte[54]);
        a.assertListCodingEquals("f838b7" + repeat("00", 55), new byte[55]);
        a.assertListCodingEquals("f90100b8fe" + repeat("00", 254), new byte[254]);
        a.assertListCodingEquals("fa010004ba010000" + repeat("00", 1 << 16), new byte[1 << 16]);
    }

    public static void testRLPSimple(Assertion a) {
        testRLPSimpleSmall(a);
        a.assertCodingEquals("bb01000000" + repeat("00", 1 << 24), new byte[1 << 24]);
        a.assertListCodingEquals("fb01000005bb01000000" + repeat("00", 1 << 24), new byte[1 << 24]);
    }
}
