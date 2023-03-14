/*
 * Copyright 2023 ICON Foundation
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

import foundation.icon.ee.test.NoDebugTest;
import foundation.icon.ee.tooling.abi.ABICompilerException;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.annotation.External;
import score.annotation.Keep;

public class StructTest extends NoDebugTest {
    static class PackagePrivate {
        private String name;

        @Keep
        public PackagePrivate() {
        }

        @Keep
        public void setName(String name) {
            this.name = name;
        }

        @Keep
        public String getName() {
            return name;
        }
    }

    public static class ScorePackagePrivateStructMaker {
        @External
        public PackagePrivate run() {
            return new PackagePrivate();
        }
    }

    @Test
    void testPackagePrivateStruct() {
        Assertions.assertThrows(ABICompilerException.class, () -> {
            makeRelJar(ScorePackagePrivateStructMaker.class, PackagePrivate.class);
        });
    }
}
