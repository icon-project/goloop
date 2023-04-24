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
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.Address;
import score.Context;
import score.annotation.External;
import score.annotation.Keep;

import java.io.IOException;
import java.util.List;
import java.util.Map;

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

    public static class ScoreListOfPackagePrivateStructMaker {
        @External
        public List<PackagePrivate> run() {
            return List.of(new PackagePrivate());
        }
    }

    @Test
    void testPackagePrivateStructList() {
        var c = sm.mustDeploy(new Class[] {ScoreListOfPackagePrivateStructMaker.class, PackagePrivate.class});
        var res = c.tryInvoke("run");
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
    }

    public static class ScoreMapMaker {
        @External
        public Map<String, String> run() {
            return Map.of("name", "Jung");
        }
    }

    public static class ScorePackagePrivateStructTaker {
        @External
        public void run(Address addr) {
            var l = Context.call(PackagePrivate.class, addr, "run");
        }
    }

    @Test
    void testPackagePrivateStructFromMap() {
        var maker = sm.mustDeploy(ScoreMapMaker.class);
        var taker = sm.mustDeploy(new Class[]{ScorePackagePrivateStructTaker.class, PackagePrivate.class});
        var res = taker.tryInvoke("run", maker.getAddress());
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
    }

    @Test
    void errorInReadableMethodProperty() throws IOException {
        var jar = readResourceFile("StructTest-packagePrivateStruct.jar");
        var c = sm.mustDeploy(jar);
        var res = c.invoke("createProject", "test1", "token_uri1");
        Assertions.assertEquals(Status.Success, res.getStatus());

        res = c.tryInvoke("getAllProjects");
        Assertions.assertEquals(Status.UnknownFailure, res.getStatus());
    }
}
