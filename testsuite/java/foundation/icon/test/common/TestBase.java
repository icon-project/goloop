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

package foundation.icon.test.common;

import foundation.icon.icx.data.TransactionResult;
import org.opentest4j.AssertionFailedError;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.fail;

public class TestBase {
    protected static void assertSuccess(TransactionResult result) {
        assertStatus(Constants.STATUS_SUCCESS, result);
    }

    protected static void assertFailure(TransactionResult result) {
        assertStatus(Constants.STATUS_FAIL, result);
        LOG.info("Expected " + result.getFailure());
    }

    private static void assertStatus(BigInteger status, TransactionResult result) {
        try {
            assertEquals(status, result.getStatus());
        } catch (AssertionFailedError e) {
            LOG.info("Assertion Failed: result=" + result);
            fail(e.getMessage());
        }
    }
}
