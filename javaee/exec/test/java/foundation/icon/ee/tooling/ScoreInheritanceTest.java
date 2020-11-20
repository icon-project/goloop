/*
 * Copyright 2020 ICON Foundation
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
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.annotation.External;

public class ScoreInheritanceTest extends SimpleTest {
    public static class Super {
        @External
        public void f() {
        }
    }

    public static class Sub extends Super {
        @External
        public void f() {
        }
    }

    public static class Sub2 extends Super {
        @External
        public void f(int v) {
        }
    }

    @Test
    public void externalMethodWithDifferentSignatureAndSameName() {
        Assertions.assertDoesNotThrow(
                () -> makeRelJar(Sub.class, Super.class)
        );
        Assertions.assertThrows(
                ABICompilerException.class,
                () -> makeRelJar(Sub2.class, Super.class)
        );
    }
}
