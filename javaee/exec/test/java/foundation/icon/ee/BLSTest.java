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

package foundation.icon.ee;

import foundation.icon.ee.test.SimpleTest;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.Context;
import score.annotation.External;

import java.util.Arrays;

public class BLSTest extends SimpleTest {
    public static class Score {
        static private byte[] hexToBytes(String s) {
            int len = s.length();
            byte[] data = new byte[len / 2];
            for (int i = 0; i < len; i += 2) {
                data[i / 2] = (byte) ((Character.digit(s.charAt(i), 16) << 4)
                        + Character.digit(s.charAt(i + 1), 16));
            }
            return data;
        }
        static private final String G1 = "bls12-381-g1";
        static private final String G2 = "bls12-381-g2";
        static private final String pa = "a85840694564cd1582f53e30fca43a396214990e5e0b255b8d257931ff0a933a5746b3a9bdd63b9c93ade10a85db0e9b";
        static private final String pb = "b0f6fc69e358da7acefc579b5c87bd5970257a347fc45aa53e73d6c65fe5354ce63f25e27412d301ba7e4661b65175f3";
        static private final String pa_plus_pb ="ae8831c4f88dfb7853af6b0c4db9fd38becb236dfbe64633c782a2796544fb8e751edcd996b0b19826a0c33fee80805b";
        static private final String pMalformed = pa.substring(0, pa.length()-2);

        static private final String pk = "a931985bb2949bd7bebf453e6ca3b653d4c661d90316e5ec0d844f3c187c2920799c605e76ff64184d0e0f5c1f69e955";
        static private final String pkMalformed = pk.substring(0, pk.length()-2);
        static private final String msg = "6d79206d657373616765";
        static private final String sig = "a9d535044a303502a75c2364570731069f862858a1b0a60ae7c2981b4aa96fa48fe8c4a25d000a2a75b0653c60658dd00ebac42bcef4b9a6fc293dce6e207e10040c909b1f3d2be5ebf55f1865d6b66d72eb8d9379df0b2a737d01de84813af1";

        static private final String sigMalformed = sig.substring(0, sig.length()-2);
        static private final String msg2 = "6d79206d657373616766";

        @External
        public void test() {
            testAggregate0();
            testAggregate1();
            testAggregate2();
            testAggregateMalformed();
            testVerifySignature();
            testVerifySignatureMalformedPK();
            testVerifySignatureMalformedSig();
            testVerifySignatureNotMatch();
        }

        public void testAggregate0() {
            var id = Context.aggregate(G1, null, new byte[0]);
            Context.require(Arrays.equals(
                    id, Context.aggregate(G1, id, id)
            ));
            Context.println("testAggregate0 - OK");
        }

        public void testAggregate1() {
            Context.require(Arrays.equals(
                    hexToBytes(pa),
                    Context.aggregate(G1, hexToBytes(pa), new byte[0])
            ));
            Context.require(Arrays.equals(
                    hexToBytes(pa),
                    Context.aggregate(G1, null, hexToBytes(pa))
            ));
            Context.println("testAggregate1 - OK");
        }

        public void testAggregate2() {
            Context.require(Arrays.equals(
                    hexToBytes(pa_plus_pb),
                    Context.aggregate(G1, null, hexToBytes(pa+pb))
            ));
            var id = Context.aggregate(G1, null, new byte[0]);
            Context.require(Arrays.equals(
                    hexToBytes(pa_plus_pb),
                    Context.aggregate(G1, id, hexToBytes(pa+pb))
            ));
            Context.require(Arrays.equals(
                    hexToBytes(pa_plus_pb),
                    Context.aggregate(G1, hexToBytes(pa), hexToBytes(pb))
            ));
            Context.println("testAggregate2 - OK");
        }

        public void testAggregateMalformed() {
            try {
                Context.aggregate(G1, hexToBytes(pMalformed), hexToBytes(pa));
                Context.require(false, "shall not reach here");
            } catch (IllegalArgumentException e) {
            }
            Context.println("testAggregateMalformed - OK");
        }

        public void testVerifySignature() {
            Context.require(Context.verifySignature(G2, hexToBytes(msg), hexToBytes(sig), hexToBytes(pk)));
            Context.println("testVerifySignature - OK");
        }

        public void testVerifySignatureMalformedPK() {
            try {
                Context.verifySignature(G2, hexToBytes(msg), hexToBytes(sig), hexToBytes(pkMalformed));
                Context.require(false, "shall not reach here");
            } catch (IllegalArgumentException e) {
            }
            Context.println("testVerifySignatureMalformedPK - OK");
        }

        public void testVerifySignatureMalformedSig() {
            try {
                Context.verifySignature(G2, hexToBytes(msg), hexToBytes(sigMalformed), hexToBytes(pk));
                Context.require(false, "shall not reach here");
            } catch (IllegalArgumentException e) {
            }
            Context.println("testVerifySignatureMalformedSig - OK");
        }

        public void testVerifySignatureNotMatch() {
            Context.require(!Context.verifySignature(G2, hexToBytes(msg2), hexToBytes(sig), hexToBytes(pk)));
            Context.println("testVerifySignatureNotMatch - OK");
        }
    }

    @Test
    public void run() {
        var c = sm.mustDeploy(Score.class);
        Assertions.assertEquals(0, c.invoke("test").getStatus());
    }
}
