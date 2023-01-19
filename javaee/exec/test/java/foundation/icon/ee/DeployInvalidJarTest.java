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

import foundation.icon.ee.test.ContractAddress;
import foundation.icon.ee.test.Jars;
import foundation.icon.ee.test.SimpleTest;
import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.Context;
import score.annotation.External;

public class DeployInvalidJarTest extends SimpleTest {
    public static class Score {
        @External
        public void run() {
            Context.println("run()");
        }
    }

    @Test
    void packageError() {
        var jar = Jars.make(Score.class);
        for (int i=0; i<jar.length/2; i++) {
            jar[jar.length-i-1] = (byte)~jar[jar.length-i-1];
        }
        var res = sm.tryDeploy(jar);
        if (res.getStatus() == Status.Success) {
            var address = (Address) res.getRet();
            var c = new ContractAddress(sm, address);
            c.invoke("run");
        }
        Assertions.assertEquals(Status.PackageError, res.getStatus());
    }

    @Test
    void illegalFormat() {
        var jar = Jars.make(Score.class);
        jar[2] = (byte)~jar[2];
        jar[3] = (byte)~jar[3];
        var res = sm.tryDeploy(jar);
        if (res.getStatus() == Status.Success) {
            var address = (Address) res.getRet();
            var c = new ContractAddress(sm, address);
            c.invoke("run");
        }
        Assertions.assertEquals(Status.IllegalFormat, res.getStatus());
    }
}
