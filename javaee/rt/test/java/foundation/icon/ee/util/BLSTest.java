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

package foundation.icon.ee.util;

import foundation.icon.ee.util.bls12381.BLS12381;
import org.aion.avm.core.util.Helpers;
import org.junit.Test;
import org.junit.jupiter.api.Assertions;
import supranational.blst.P1;
import supranational.blst.P2;
import java.math.BigInteger;

public class BLSTest {

    private static byte[] concatBytes(byte[]... args) {
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

    @Test
    public void zeroAggregationIsIdentity() {
        Assertions.assertArrayEquals(BLS12381.identity().compress(), BLS12381.aggregateG1Values(null, new byte[0]));
    }

    @Test
    public void identity() {
        // P1a + (-P1a) = I1
        // P1a + I1 = P1a
        var pk1 = Helpers.hexStringToBytes(
                "a85840694564cd1582f53e30fca43a396214990e5e0b255b8d257931ff0a933a5746b3a9bdd63b9c93ade10a85db0e9b");
        P1 p1a = new P1(pk1);
        P1 p1aNeg = p1a.dup().neg();
        P1 id = p1a.dup().add(p1aNeg);
        System.out.printf("p1a=   %s%n", Helpers.bytesToHexString(p1a.compress()));
        System.out.printf("p1aNeg=%s%n", Helpers.bytesToHexString(p1aNeg.compress()));
        System.out.printf("id=    %s%n", Helpers.bytesToHexString(id.compress()));
        Assertions.assertArrayEquals(p1a.compress(), p1a.dup().add(id).compress());

        // P2a + (-P2a) = I2
        // I1 = I2
        var pk2 = Helpers.hexStringToBytes(
                "b0f6fc69e358da7acefc579b5c87bd5970257a347fc45aa53e73d6c65fe5354ce63f25e27412d301ba7e4661b65175f3");
        P1 p1b = new P1(pk2);
        P1 p1bNeg = p1b.dup().neg();
        P1 id2 = p1b.dup().add(p1bNeg);
        Assertions.assertArrayEquals(id.compress(), id2.compress());

        // I1 + I2 = I1
        P1 id3 = id.dup().add(id2);
        Assertions.assertArrayEquals(id.compress(), id3.compress());

        P1 p1c = new P1();
        Assertions.assertArrayEquals(id.compress(), p1c.compress());
        System.out.printf("id=    %s%n", Helpers.bytesToHexString(p1c.compress()));
    }

    @Test
    public void identityInGroup() {
        Assertions.assertTrue(BLS12381.identity().in_group());
    }

    @Test
    public void pairingCheck() {
        P1 g1 = P1.generator();
        P2 g2 = P2.generator();

        Assertions.assertTrue(BLS12381.pairingCheck(
                concatBytes(
                        g1.serialize(), g2.serialize(),
                        g1.dup().neg().serialize(), g2.serialize(),
                        g1.serialize(), g2.serialize(),
                        g1.serialize(), g2.dup().neg().serialize()),
                false));

        // compressed points
        Assertions.assertTrue(BLS12381.pairingCheck(
                concatBytes(
                        g1.compress(), g2.compress(),
                        g1.dup().neg().compress(), g2.compress(),
                        g1.compress(), g2.compress(),
                        g1.compress(), g2.dup().neg().compress()),
                true));

    }

    @Test
    public void addAndScalarMul() {
        byte[] out;
        P1 g1;
        P2 g2;
        byte[] g1x2b, g1x3b, g2x2b, g2x3b;

        // g1 add and scalarMul tests
        g1 = P1.generator();
        g1x2b = BLS12381.g1ScalarMul(new BigInteger("2").toByteArray(), g1.serialize(), false);
        g1x3b = BLS12381.g1ScalarMul(new BigInteger("3").toByteArray(), g1.serialize(), false);

        out = BLS12381.g1Add(concatBytes(g1.serialize(), g1x2b), false);

        Assertions.assertTrue(new P1(g1x3b).is_equal(new P1(out)), "should be equal");

        // g1 add and scalarMul tests: compressed points
        g1 = P1.generator();
        g1x2b = BLS12381.g1ScalarMul(new BigInteger("2").toByteArray(), g1.compress(), true);
        g1x3b = BLS12381.g1ScalarMul(new BigInteger("3").toByteArray(), g1.compress(), true);

        out = BLS12381.g1Add(concatBytes(g1.compress(), g1x2b), true);

        Assertions.assertTrue(new P1(g1x3b).is_equal(new P1(out)), "should be equal");

        // g2 add and scalarMul tests
        g2 = P2.generator();
        g2x2b = BLS12381.g2ScalarMul(new BigInteger("2").toByteArray(), g2.serialize(), false);
        g2x3b = BLS12381.g2ScalarMul(new BigInteger("3").toByteArray(), g2.serialize(), false);

        out = BLS12381.g2Add(concatBytes(g2.serialize(), g2x2b), false);

        Assertions.assertTrue(new P2(g2x3b).is_equal(new P2(out)), "should be equal");

        // g2 add and scalarMul tests: compressed points
        g2 = P2.generator();
        g2x2b = BLS12381.g2ScalarMul(new BigInteger("2").toByteArray(), g2.compress(), true);
        g2x3b = BLS12381.g2ScalarMul(new BigInteger("3").toByteArray(), g2.compress(), true);

        out = BLS12381.g2Add(concatBytes(g2.compress(), g2x2b), true);

        Assertions.assertTrue(new P2(g2x3b).is_equal(new P2(out)), "should be equal");

    }

}
