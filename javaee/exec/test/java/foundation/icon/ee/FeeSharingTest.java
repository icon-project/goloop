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

package foundation.icon.ee;

import foundation.icon.ee.test.GoldenTest;
import org.junit.jupiter.api.Test;
import testcases.FeeSharing;

import java.math.BigInteger;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class FeeSharingTest extends GoldenTest {
    @Test
    void test() {
        var owner = sm.getOrigin();
        var score = sm.mustDeploy(FeeSharing.class);
        assertEquals(BigInteger.ZERO, score.query("getProportion", owner).getRet());
        String value = "No value";
        assertEquals(value, score.query("getValue").getRet());
        value = "Value #1";
        score.invoke("setValue", value);
        assertEquals(value, score.query("getValue").getRet());
        score.invoke("addToWhitelist", owner, 100);
        assertEquals(BigInteger.valueOf(100), score.query("getProportion", owner).getRet());
        value = "Value #2";
        score.invoke("setValue", value);
        assertEquals(value, score.query("getValue").getRet());
    }
}
