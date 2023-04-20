/*
 * Copyright 2022 ICON Foundation
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

package foundation.icon.ee.util.bls12381;

import java.util.Arrays;

import supranational.blst.BLST_ERROR;
import supranational.blst.P1;
import supranational.blst.P1_Affine;
import supranational.blst.P2;
import supranational.blst.P2_Affine;
import supranational.blst.PT;
import supranational.blst.Scalar;


public class BLS12381 {
    private static final String dst = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_";
    public static final int G1_LEN = 48;
    public static final int G2_LEN = 96;
    private static final P1 I = new P1();

    public static P1 identity() {
        return I.dup();
    }

    /**
     * Returns aggregation of prevAgg and values.
     *
     * @param prevAgg previous aggregation. null if there is no previous
     *                aggregation.
     * @param values  values to be aggregated.
     * @return aggregated value.
     */
    public static byte[] aggregateG1Values(byte[] prevAgg, byte[] values) {
        try {
            P1 res;
            if (prevAgg != null) {
                res = new P1(prevAgg);
                if (!res.in_group()) {
                    throw new IllegalArgumentException("prevAgg is not in group");
                }
            } else {
                res = I.dup();
            }
            var nValues = values.length / G1_LEN;
            byte[] pk = new byte[G1_LEN];
            for (int i = 0; i < nValues; i++) {
                System.arraycopy(values, i * G1_LEN, pk, 0, G1_LEN);
                var p1a = new P1_Affine(pk);
                if (!p1a.in_group()) {
                    throw new IllegalArgumentException("a value is not in group");
                }
                res.aggregate(p1a);
            }
            return res.compress();
        } catch (Throwable e) {
            if (!(e instanceof IllegalArgumentException)) {
                throw new IllegalArgumentException(e.getMessage());
            }
        }
        // never reach here.
        return null;
    }

    public static boolean verifyG2Signature(byte[] pubKey, byte[] sig, byte[] msg) {
        try {
            var p1a = new P1_Affine(pubKey);
            var p2a = new P2_Affine(sig);
            var err = p2a.core_verify(p1a, true, msg, dst);
            return err == BLST_ERROR.BLST_SUCCESS;
        } catch (Exception e) {
            throw new IllegalArgumentException(e);
        }
    }

    private static byte[] concat(byte[]... args) {
        int length = 0;
        for (int i = 0; i < args.length; i++) {
            length += args[i].length;
        }
        byte[] out = new byte[length];
        int offset = 0;
        for (int i = 0; i < args.length; i++) {
            System.arraycopy(args[i], 0, out, offset, args[i].length);
            offset += args[i].length;
        }
        return out;
    }

    public static byte[] g1Add(byte[] data, boolean compressed) {
        P1 acc = new P1();
        int size = compressed ? G1_LEN : 2 * G1_LEN;
        if (data.length == 0 || data.length % size != 0) {
            throw new IllegalArgumentException("BLS12-381: g1Add: invalid data layout: expected a multiple of " + size
                    + " bytes, got " + data.length);
        }
        byte[] buf = new byte[size];
        for (int i = 0; i < data.length; i += size) {
            System.arraycopy(data, i, buf, 0, size);
            acc = acc.add(new P1(buf));
        }
        return compressed ? acc.compress() : acc.serialize();
    }

    public static byte[] g2Add(byte[] data, boolean compressed) {
        P2 acc = new P2();
        int size = compressed ? G2_LEN : 2 * G2_LEN;
        if (data.length == 0 || data.length % size != 0) {
            throw new IllegalArgumentException("BLS12-381: g2Add: invalid data layout: expected a multiple of " + size
                    + " bytes, got " + data.length);
        }
        for (int i = 0; i < data.length; i += size) {
            byte[] buf = Arrays.copyOfRange(data, i, i + size);
            acc = acc.add(new P2(buf));
        }
        return compressed ? acc.compress() : acc.serialize();
    }

    public static byte[] g1ScalarMul(byte[] scalarBytes, byte[] data, boolean compressed) {
        int size = compressed ? G1_LEN : 2 * G1_LEN;
        Scalar scalar = new Scalar().from_bendian(scalarBytes);
        if (data.length != size) {
            throw new IllegalArgumentException(
                    "BLS12-381: g1ScalarMul: invalid data layout: expected " + size + " bytes, got " + data.length);
        }
        P1 p = new P1(data);
        p = p.mult(scalar);
        return compressed ? p.compress() : p.serialize();
    }

    public static byte[] g2ScalarMul(byte[] scalarBytes, byte[] data, boolean compressed) {
        int size = compressed ? G2_LEN : 2 * G2_LEN;
        Scalar scalar = new Scalar().from_bendian(scalarBytes);
        if (data.length != size) {
            throw new IllegalArgumentException(
                    "BLS12-381: g2ScalarMul: invalid data layout: expected " + size + " bytes, got " + data.length);
        }
        P2 p = new P2(data);
        p = p.mult(scalar);
        return compressed ? p.compress() : p.serialize();
    }

    public static boolean pairingCheck(byte[] data, boolean compressed) {
        int g1Size = compressed ? G1_LEN : 2 * G1_LEN;
        int g2Size = compressed ? G2_LEN : 2 * G2_LEN;
        int size = g1Size + g2Size;

        if (data.length == 0 || data.length % size != 0) {
            throw new IllegalArgumentException("BLS12-381: pairingCheck: invalid data layout: expected a multiple of "
                    + size + " bytes, got " + data.length);
        }

        PT acc = PT.one();

        for (int i = 0; i < data.length; i += size) {
            byte[] p1buf = Arrays.copyOfRange(data, i, i + g1Size);
            byte[] p2buf = Arrays.copyOfRange(data, i + g1Size, i + g1Size + g2Size);
            P1 p1 = new P1(p1buf);
            P2 p2 = new P2(p2buf);
            if (!p1.in_group() || !p2.in_group()) {
                throw new IllegalArgumentException("G1 or G2 point not in subgroup!");
            }
            if (p1.is_inf() || p2.is_inf()) {
                continue;
            }
            acc = acc.mul(new PT(p1, p2));
        }

        return acc.final_exp().is_one();
    }

}
