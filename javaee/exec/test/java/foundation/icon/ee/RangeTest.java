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

import foundation.icon.ee.test.ContractAddress;
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

public class RangeTest extends SimpleTest {
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

        @External
        public BigInteger subIH(BigInteger a, BigInteger b) {
            return null;
        }
    }

    public static class CharUser {
        @External
        public void take(char value) {
        }

        @External
        public char sum(score.Address adder, char a, char b) {
            return Context.call(char.class, adder, "sumIH", a, b);
        }

        @External
        public char sumWrapper(score.Address adder, char a, char b) {
            return Context.call(Character.class, adder, "sumIH", a, b);
        }

        @External
        public char sub(score.Address adder, char a, char b) {
            return Context.call(char.class, adder, "subIH", a, b);
        }

        @External
        public char subWrapper(score.Address adder, char a, char b) {
            return Context.call(Character.class, adder, "subIH", a, b);
        }
    }

    public static class ByteUser {
        @External
        public void take(byte value) {
        }

        @External
        public byte sum(score.Address adder, byte a, byte b) {
            return Context.call(byte.class, adder, "sumIH", a, b);
        }

        @External
        public byte sumWrapper(score.Address adder, byte a, byte b) {
            return Context.call(Byte.class, adder, "sumIH", a, b);
        }
    }

    public static class ShortUser {
        @External
        public void take(short value) {
        }

        @External
        public short sum(score.Address adder, short a, short b) {
            return Context.call(short.class, adder, "sumIH", a, b);
        }

        @External
        public short sumWrapper(score.Address adder, short a, short b) {
            return Context.call(Short.class, adder, "sumIH", a, b);
        }
    }

    public static class IntUser {
        @External
        public void take(int value) {
        }

        @External
        public int sum(score.Address adder, int a, int b) {
            return Context.call(int.class, adder, "sumIH", a, b);
        }

        @External
        public int sumWrapper(score.Address adder, int a, int b) {
            return Context.call(Integer.class, adder, "sumIH", a, b);
        }
    }

    public static class LongUser {
        @External
        public void take(long value) {
        }

        @External
        public long sum(score.Address adder, long a, long b) {
            return Context.call(long.class, adder, "sumIH", a, b);
        }

        @External
        public long sumWrapper(score.Address adder, long a, long b) {
            return Context.call(Long.class, adder, "sumIH", a, b);
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
        public Result invoke(ServiceManager sm, String code, int flag, Address from, Address to, BigInteger value, BigInteger stepLimit, String method, Object[] params, Map<String, Object> info, byte[] cid, int eid, Object[] codeState) throws IOException {
            if ("sumIH".equals(method)) {
                BigInteger a = (BigInteger) params[0];
                BigInteger b = (BigInteger) params[1];
                return new Result(Status.Success, 10000000, a.add(b));
            } else if ("subIH".equals(method)) {
                BigInteger a = (BigInteger) params[0];
                BigInteger b = (BigInteger) params[1];
                return new Result(Status.Success, 10000000, a.subtract(b));
            }
            return InvokeHandler.defaultHandler().invoke(sm, code, flag,
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
        res = c.tryInvoke("sum", f.getAddress(), max, 1);
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
        res = c.tryInvoke("sum", f.getAddress(), min, 0);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("sum", f.getAddress(), min, -1);
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
    }

    @Test
    public void charValue() {
        var c = sm.mustDeploy(CharUser.class);

        var res = c.tryInvoke("take", Character.MAX_VALUE);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("take", BigInteger.valueOf(Character.MAX_VALUE).add(BigInteger.ONE));
        Assertions.assertEquals(Status.InvalidParameter, res.getStatus());

        res = c.tryInvoke("take", Character.MIN_VALUE);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("take", BigInteger.valueOf(Character.MIN_VALUE).subtract(BigInteger.ONE));
        Assertions.assertEquals(Status.InvalidParameter, res.getStatus());
    }

    @Test
    public void charFromForeignContract() {
        var c = sm.mustDeploy(CharUser.class);
        var f = sm.mustDeploy(ForeignContract.class, new MyInvokeHandler());

        Result res;
        res = c.tryInvoke("sum", f.getAddress(), Character.MAX_VALUE, 0);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("sum", f.getAddress(), Character.MAX_VALUE, 1);
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
        res = c.tryInvoke("sub", f.getAddress(), Character.MIN_VALUE, 0);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("sub", f.getAddress(), Character.MIN_VALUE, 1);
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
        res = c.tryInvoke("sumWrapper", f.getAddress(), Character.MAX_VALUE, 0);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("sumWrapper", f.getAddress(), Character.MAX_VALUE, 1);
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
        res = c.tryInvoke("subWrapper", f.getAddress(), Character.MIN_VALUE, 0);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("subWrapper", f.getAddress(), Character.MIN_VALUE, 1);
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
    }

    private void testForeignContract(ContractAddress c, ContractAddress f,
            String method, long max, long min) {
        Result res;
        res = c.tryInvoke(method, f.getAddress(), max, 0);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke(method, f.getAddress(), max, 1);
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
        res = c.tryInvoke(method, f.getAddress(), min, 0);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke(method, f.getAddress(), min, -1);
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
    }

    @Test
    public void byteValue() {
        var c = sm.mustDeploy(ByteUser.class);

        var res = c.tryInvoke("take", Byte.MAX_VALUE);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("take", BigInteger.valueOf(Byte.MAX_VALUE).add(BigInteger.ONE));
        Assertions.assertEquals(Status.InvalidParameter, res.getStatus());

        res = c.tryInvoke("take", Byte.MIN_VALUE);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("take", BigInteger.valueOf(Byte.MIN_VALUE).subtract(BigInteger.ONE));
        Assertions.assertEquals(Status.InvalidParameter, res.getStatus());
    }

    @Test
    public void byteFromForeignContract() {
        var c = sm.mustDeploy(ByteUser.class);
        var f = sm.mustDeploy(ForeignContract.class, new MyInvokeHandler());

        testForeignContract(c, f, "sum", Byte.MAX_VALUE, Byte.MIN_VALUE);
        testForeignContract(c, f, "sumWrapper", Byte.MAX_VALUE, Byte.MIN_VALUE);
    }

    @Test
    public void shortValue() {
        var c = sm.mustDeploy(ShortUser.class);

        var res = c.tryInvoke("take", Short.MAX_VALUE);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("take", BigInteger.valueOf(Short.MAX_VALUE).add(BigInteger.ONE));
        Assertions.assertEquals(Status.InvalidParameter, res.getStatus());

        res = c.tryInvoke("take", Short.MIN_VALUE);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("take", BigInteger.valueOf(Short.MIN_VALUE).subtract(BigInteger.ONE));
        Assertions.assertEquals(Status.InvalidParameter, res.getStatus());
    }

    @Test
    public void shortFromForeignContract() {
        var c = sm.mustDeploy(ShortUser.class);
        var f = sm.mustDeploy(ForeignContract.class, new MyInvokeHandler());

        testForeignContract(c, f, "sum", Short.MAX_VALUE, Short.MIN_VALUE);
        testForeignContract(c, f, "sumWrapper", Short.MAX_VALUE, Short.MIN_VALUE);
    }

    @Test
    public void intValue() {
        var c = sm.mustDeploy(IntUser.class);

        var res = c.tryInvoke("take", Integer.MAX_VALUE);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("take", BigInteger.valueOf(Integer.MAX_VALUE).add(BigInteger.ONE));
        Assertions.assertEquals(Status.InvalidParameter, res.getStatus());

        res = c.tryInvoke("take", Integer.MIN_VALUE);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("take", BigInteger.valueOf(Integer.MIN_VALUE).subtract(BigInteger.ONE));
        Assertions.assertEquals(Status.InvalidParameter, res.getStatus());
    }

    @Test
    public void intFromForeignContract() {
        var c = sm.mustDeploy(IntUser.class);
        var f = sm.mustDeploy(ForeignContract.class, new MyInvokeHandler());

        testForeignContract(c, f, "sum", Integer.MAX_VALUE, Integer.MIN_VALUE);
        testForeignContract(c, f, "sumWrapper", Integer.MAX_VALUE, Integer.MIN_VALUE);
    }

    @Test
    public void longValue() {
        var c = sm.mustDeploy(LongUser.class);

        var res = c.tryInvoke("take", Long.MAX_VALUE);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("take", BigInteger.valueOf(Long.MAX_VALUE).add(BigInteger.ONE));
        Assertions.assertEquals(Status.InvalidParameter, res.getStatus());

        res = c.tryInvoke("take", Long.MIN_VALUE);
        Assertions.assertEquals(Status.Success, res.getStatus());
        res = c.tryInvoke("take", BigInteger.valueOf(Long.MIN_VALUE).subtract(BigInteger.ONE));
        Assertions.assertEquals(Status.InvalidParameter, res.getStatus());
    }

    @Test
    public void longFromForeignContract() {
        var c = sm.mustDeploy(LongUser.class);
        var f = sm.mustDeploy(ForeignContract.class, new MyInvokeHandler());

        testForeignContract(c, f, "sum", Long.MAX_VALUE, Long.MIN_VALUE);
        testForeignContract(c, f, "sumWrapper", Long.MAX_VALUE, Long.MIN_VALUE);
    }
}
