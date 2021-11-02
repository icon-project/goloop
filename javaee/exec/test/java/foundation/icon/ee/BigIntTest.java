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

import foundation.icon.ee.test.InvokeHandler;
import foundation.icon.ee.test.ServiceManager;
import foundation.icon.ee.test.SimpleTest;
import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Result;
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.Context;
import score.annotation.External;

import java.io.IOException;
import java.math.BigInteger;
import java.util.Map;

public class BigIntTest extends SimpleTest {
    public static class BigIntUser {
        @External
        public void take(BigInteger value) {
        }

        @External
        public BigInteger make(String value, int radix) {
            return new BigInteger(value, radix);
        }

        @External
        public BigInteger sum(score.Address adder, BigInteger a, BigInteger b) {
            return Context.call(BigInteger.class, adder, "sumIH", a, b);
        }
    }

    public static class ForeignContract {
        // placeholder method for InvokeHandler
        // add two big integers without limit like python
        @External
        public BigInteger sumIH(BigInteger a, BigInteger b) {
            return null;
        }
    }

    // 2^512-1
    static final BigInteger max = BigInteger.valueOf(2).pow(512)
            .subtract(BigInteger.ONE);
    static final BigInteger aboveMax = max.add(BigInteger.ONE);
    // -(2^512 - 1)
    static final BigInteger min = max.negate();
    static final BigInteger belowMin = min.subtract(BigInteger.ONE);

    static class MyInvokeHandler implements InvokeHandler {
        @Override
        public Result invoke(ServiceManager sm, String code, boolean isReadOnly, Address from, Address to, BigInteger value, BigInteger stepLimit, String method, Object[] params, Map<String, Object> info, byte[] cid, int eid, Object[] codeState) throws IOException {
            if ("sumIH".equals(method)) {
                BigInteger a = (BigInteger) params[0];
                BigInteger b = (BigInteger) params[1];
                return new Result(Status.Success, 10000000,
                        a.add(b));
            }
            return InvokeHandler.defaultHandler().invoke(sm, code, isReadOnly,
                    from, to, value, stepLimit, method, params, info, cid, eid,
                    codeState);
        }
    }

    @Test
    public void bigInt() {
        var c = sm.mustDeploy(BigIntUser.class);

        var res = c.tryInvoke("make", max.toString(16), 16);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("take", max);
        Assertions.assertEquals(Status.Success, res.getStatus());

        res = c.tryInvoke("make", aboveMax.toString(16), 16);
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
        res = c.tryInvoke("take", aboveMax);
        Assertions.assertEquals(Status.InvalidParameter, res.getStatus());

        res = c.tryInvoke("make", min.toString(16), 16);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("take", min);
        Assertions.assertEquals(Status.Success, res.getStatus());

        res = c.tryInvoke("make", belowMin.toString(16), 16);
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
        res = c.tryInvoke("take", belowMin);
        Assertions.assertEquals(Status.InvalidParameter, res.getStatus());
    }

    @Test
    public void bigIntFromForeignContract() {
        var c = sm.mustDeploy(BigIntUser.class);
        var f = sm.mustDeploy(ForeignContract.class, new MyInvokeHandler());

        var res = c.tryInvoke("sum", f.getAddress(), max, BigInteger.ZERO);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("sum", f.getAddress(), max, BigInteger.ONE);
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
        res = c.tryInvoke("sum", f.getAddress(), min, BigInteger.ZERO);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("sum", f.getAddress(), min, BigInteger.ONE.negate());
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
    }
}
