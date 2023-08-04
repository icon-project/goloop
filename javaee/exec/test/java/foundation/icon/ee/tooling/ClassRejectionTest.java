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

package foundation.icon.ee.tooling;

import foundation.icon.ee.test.SimpleTest;
import foundation.icon.ee.tooling.abi.ABICompilerException;
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.Context;
import score.UserRevertException;
import score.annotation.External;
import score.annotation.Keep;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;

public class ClassRejectionTest extends SimpleTest {

    public static class ClassNotInJCL {
        public ClassNotInJCL() {
            byte[] b = null;
            Objects.requireNonNull(b);
        }
    }

    public static class ClassNotInJCL2 {
        public ClassNotInJCL2() {
            System.out.println("test");
        }
    }

    public static class ClassNotInJCL3 {
        public ClassNotInJCL3() {
            List<String> list = new ArrayList<>();
            list.add("a");
            Context.println(list.toString());
        }
    }

    public static class ClassNotInJCL4 {
        public ClassNotInJCL4() {
            Map<String, String> map = new HashMap<>();
            map.put("a", "test");
            Context.println(map.toString());
        }
    }

    public static class MethodNotSupported {
        public MethodNotSupported() {
            Context.println("2^10=" + BigInteger.TWO.pow(10));
        }
    }

    public static class MethodNotSupported2 {
        public MethodNotSupported2() {
            String a = "test";
            Context.println(String.format("%s", a));
        }
    }

    public static class MethodNotSupported3 {
        public MethodNotSupported3() {
            Arrays.asList("a", "b");
        }
    }

    public static class LambdaPredicate {
        public LambdaPredicate() {
            var list = List.of("a", "b");
            list.removeIf(e -> e.equals("a"));
        }
    }

    @Test
    public void testUnsupported() {
        final Class<?>[] cases = new Class[]{
                ClassNotInJCL.class,
                ClassNotInJCL2.class,
                ClassNotInJCL3.class,
                ClassNotInJCL4.class,
                MethodNotSupported.class,
                MethodNotSupported2.class,
                MethodNotSupported3.class,
                LambdaPredicate.class,
        };
        for (var c : cases) {
            Exception e = Assertions.assertThrows(
                    UnsupportedOperationException.class,
                    () -> makeRelJar(c));
            System.out.println(e.getMessage());
        }
    }

    public static class InvalidFallback {
        @External
        public void fallback() {
        }
    }

    public static class NoReturnInReadOnlyMethod {
        @External(readonly=true)
        public void readProp() {
        }
    }

    @Test
    public void testExternal() {
        final Class<?>[] cases = new Class[]{
                InvalidFallback.class,
                NoReturnInReadOnlyMethod.class,
        };
        for (var c : cases) {
            Exception e = Assertions.assertThrows(
                    ABICompilerException.class,
                    () -> makeRelJar(c));
            System.out.println(e.getMessage());
        }
    }

    public static class UseCustomExceptionForRevert {
        public UseCustomExceptionForRevert() {
            var e = new CustomException(100);
            Context.println(e.getMessage());
            throw e;
        }

        static class CustomException extends UserRevertException {
            private final int code;

            public CustomException(int code) {
                this.code = code;
            }

            @Override
            @Keep
            public int getCode() {
                return this.code;
            }
        }
    }

    @Test
    public void testUserRevertException() {
        Assertions.assertDoesNotThrow(() -> {
            var jar = makeRelJar(UseCustomExceptionForRevert.class);
            var res = sm.tryDeploy(jar);
            Assertions.assertEquals(Status.UserReversionStart + 100, res.getStatus());
        });
    }
}
