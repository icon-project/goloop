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

import supranational.blst.BLST_ERROR;
import supranational.blst.P1;
import supranational.blst.P1_Affine;
import supranational.blst.P2_Affine;

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
     * @param prevAgg previous aggregation. null if there is no previous
     *                aggregation.
     * @param values values to be aggregated.
     * @return aggregated value.
     */
    public static byte[] aggregateG1Values(byte[] prevAgg, byte[] values) {
        try {
            P1 res;
            if (prevAgg!=null) {
                res = new P1(prevAgg);
                if (!res.in_group()) {
                    throw new IllegalArgumentException("prevAgg is not in group");
                }
            } else {
                res = I.dup();
            }
            var nValues = values.length / G1_LEN;
            byte[] pk = new byte[G1_LEN];
            for (int i=0; i<nValues; i++) {
                System.arraycopy(values, i*G1_LEN, pk, 0, G1_LEN);
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
}
