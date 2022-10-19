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

import java.util.Arrays;

public class BLSTest {
    private static byte[] hexToBytes(String hex) {
        return Helpers.hexStringToBytes(hex);
    }

    @Test
    public void zeroAggregationIsIdentity() {
        Assertions.assertArrayEquals(BLS12381.identity().compress(), BLS12381.aggregateG1Values(null, new byte[0]));
    }

    @Test
    public void identity() {
        // P1a + (-P1a) = I1
        // P1a + I1 = P1a
        var pk1 = Helpers.hexStringToBytes("a85840694564cd1582f53e30fca43a396214990e5e0b255b8d257931ff0a933a5746b3a9bdd63b9c93ade10a85db0e9b");
        P1 p1a = new P1(pk1);
        P1 p1aNeg = p1a.dup().neg();
        P1 id = p1a.dup().add(p1aNeg);
        System.out.printf("p1a=   %s%n", Helpers.bytesToHexString(p1a.compress()));
        System.out.printf("p1aNeg=%s%n", Helpers.bytesToHexString(p1aNeg.compress()));
        System.out.printf("id=    %s%n", Helpers.bytesToHexString(id.compress()));
        Assertions.assertArrayEquals(p1a.compress(), p1a.dup().add(id).compress());

        // P2a + (-P2a) = I2
        // I1 = I2
        var pk2 = Helpers.hexStringToBytes("b0f6fc69e358da7acefc579b5c87bd5970257a347fc45aa53e73d6c65fe5354ce63f25e27412d301ba7e4661b65175f3");
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
}
