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
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import testcases.BLSTestScore;

public class BLSTest extends SimpleTest {
    @Test
    public void run() {
        var c = sm.mustDeploy(BLSTestScore.class);
        Assertions.assertEquals(0, c.invoke("test").getStatus());
    }
}
