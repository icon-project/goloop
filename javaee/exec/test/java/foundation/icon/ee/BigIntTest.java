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

package foundation.icon.ee;

import foundation.icon.ee.test.SimpleTest;
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.annotation.External;

import java.math.BigInteger;
import java.util.Arrays;

public class BigIntTest extends SimpleTest {
    public static class BigIntUser {
        @External
        public void take(BigInteger value) {
        }

        @External
        public BigInteger make(String value, int radix) {
            return new BigInteger(value, radix);
        }
    }

    @Test
    public void bigInt() {
        var c = sm.mustDeploy(BigIntUser.class);

        // 2^512-1
        var max = BigInteger.valueOf(2).pow(512).subtract(BigInteger.ONE);
        var res = c.tryInvoke("make", max.toString(16), 16);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("take", max);
        Assertions.assertEquals(Status.Success, res.getStatus());

        var aboveMax = max.add(BigInteger.ONE);
        res = c.tryInvoke("make", aboveMax.toString(16), 16);
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
        res = c.tryInvoke("take", aboveMax);
        Assertions.assertEquals(Status.InvalidParameter, res.getStatus());

        // -(2^512 - 1)
        var min = max.negate();
        res = c.tryInvoke("make", min.toString(16), 16);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("take", min);
        Assertions.assertEquals(Status.Success, res.getStatus());

        var belowMin = min.subtract(BigInteger.ONE);
        res = c.tryInvoke("make", belowMin.toString(16), 16);
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
        res = c.tryInvoke("take", belowMin);
        Assertions.assertEquals(Status.InvalidParameter, res.getStatus());
    }
}
