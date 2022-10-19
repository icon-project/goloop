/*
 * Copyright 2021 ICON Foundation
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

import org.junit.Test;
import org.junit.jupiter.api.Assertions;

import java.math.BigInteger;
import java.nio.charset.StandardCharsets;

public class HashTest {
    public byte[] hexToBytes(String hexString) {
        byte[] byteArray = new BigInteger(hexString, 16)
                .toByteArray();
        if (byteArray[0] == 0) {
            byte[] output = new byte[byteArray.length - 1];
            System.arraycopy(
                    byteArray, 1, output,
                    0, output.length);
            return output;
        }
        return byteArray;
    }

    void testHash(String alg, String msg, String exp) {
        var m = msg.getBytes(StandardCharsets.UTF_8);
        var e = hexToBytes(exp);
        var o = Crypto.hash(alg, m);
        Assertions.assertArrayEquals(e, o, "for message: " + Strings.hexFromBytes(m));
    }

    @Test
    public void testHashes() {
        testHash("xxhash-128", "test\n", "3ca61fda08187f2d0012a945c6197364");
        testHash("blake2b-128", "test\n", "21ebd7636fdde0f4929e0ed3c0beaf55");
        testHash("blake2b-256", "test\n", "579da00778a5b4567c94630399203935f7d84bb2c457e56537e36a56ff490a4a");
    }
}
