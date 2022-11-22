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
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.math.BigInteger;
import java.util.function.Function;

import score.Context;
import score.RevertedException;
import score.annotation.External;

public class LambdaExceptionTest extends SimpleTest {
    public static class RunnableScore {
        @External
        public void run() {
            Runnable f = () -> {
                throw new IllegalArgumentException();
            };
            var expected = false;
            try {
                f.run();
                Context.require(false, "no exception");
            } catch (IllegalArgumentException e) {
                expected = true;
            }
            Context.require(expected, "not an IllegalArgumentException");
        }
    }

    @Test
    void runnable() {
        var c = sm.mustDeploy(RunnableScore.class);
        c.invoke("run");
    }

    public static class NestedRunnableScore {
        @External
        public void run() {
            Runnable f = () -> {
                Runnable g = () -> {
                    throw new IllegalArgumentException();
                };
                g.run();
            };
            var expected = false;
            try {
                f.run();
                Context.require(false, "no exception");
            } catch (IllegalArgumentException e) {
                expected = true;
            }
            Context.require(expected, "not an IllegalArgumentException");
        }
    }

    @Test
    void nestedRunnable() {
        var c = sm.mustDeploy(NestedRunnableScore.class);
        c.invoke("run");
    }

    public static class FunctionScore {
        @External
        public void run() {
            Function<BigInteger, BigInteger> f = (a) -> {
                throw new IllegalArgumentException();
            };
            var expected = false;
            try {
                f.apply(BigInteger.ONE);
                Context.require(false, "no exception");
            } catch (IllegalArgumentException e) {
                expected = true;
            }
            Context.require(expected, "not an IllegalArgumentException");
        }
    }

    @Test
    void function() {
        var c = sm.mustDeploy(FunctionScore.class);
        c.invoke("run");
    }

    public static class NestedFunctionScore {
        @External
        public void run() {
            Function<BigInteger, BigInteger> f = (a) -> {
                Function<BigInteger, BigInteger> g = (b) -> {
                    throw new IllegalArgumentException();
                };
                return g.apply(a);
            };
            var expected = false;
            try {
                f.apply(BigInteger.ONE);
                Context.require(false, "no exception");
            } catch (IllegalArgumentException e) {
                expected = true;
            }
            Context.require(expected, "not an IllegalArgumentException");
        }
    }

    @Test
    void nestedFunction() {
        var c = sm.mustDeploy(FunctionScore.class);
        c.invoke("run");
    }

    public static class AvmExceptionScore {
        @External
        public void revert() {
            Runnable f = () -> {
                try {
                    // throws AvmException
                    Context.require(false, "AvmException by force");
                } catch (Throwable t) {
                    // cannot catch
                    Context.require(false, "shall not caught AvmException");
                }
            };
            f.run();
            Context.require(false, "no exception");
        }

        @External
        public void run() {
            var expected = false;
            try {
                Context.call(Context.getAddress(), "revert");
                Context.require(false, "no exception");
            } catch (RevertedException e) {
                expected = true;
            }
            Context.require(expected, "not a RevertedException");
        }
    }

    @Test
    void avmException() {
        var c = sm.mustDeploy(AvmExceptionScore.class);
        c.invoke("run");
    }
}
