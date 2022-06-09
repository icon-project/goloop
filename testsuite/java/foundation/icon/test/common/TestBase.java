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

import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import org.opentest4j.AssertionFailedError;

import java.io.IOException;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.List;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.fail;

public class TestBase {
    protected static final BigInteger ICX = BigInteger.TEN.pow(18);

    protected static void assertSuccess(TransactionResult result) {
        assertStatus(Constants.STATUS_SUCCESS, result);
    }

    protected static void assertFailure(TransactionResult result) {
        assertStatus(Constants.STATUS_FAILURE, result);
        LOG.info("Expected " + result.getFailure());
    }

    protected static void assertFailureCode(int code, TransactionResult result) {
        assertStatus(Constants.STATUS_FAILURE, result);
        assertEquals(code, result.getFailure().getCode().intValue());
        LOG.info("Expected " + result.getFailure());
    }

    protected static void assertStatus(BigInteger status, TransactionResult result) {
        try {
            assertEquals(status, result.getStatus());
        } catch (AssertionFailedError e) {
            LOG.info("Assertion Failed: result=" + result);
            fail(e.getMessage());
        }
    }

    protected static void transferAndCheckResult(TransactionHandler txHandler, Address to, BigInteger amount)
            throws IOException, ResultTimeoutException {
        Bytes txHash = txHandler.transfer(to, amount);
        assertSuccess(txHandler.getResult(txHash));
    }

    protected static void transferAndCheckResult(TransactionHandler txHandler, Address[] addresses, BigInteger amount)
            throws IOException, ResultTimeoutException {
        List<Bytes> hashes = new ArrayList<>();
        for (Address to : addresses) {
            hashes.add(txHandler.transfer(to, amount));
        }
        for (Bytes hash : hashes) {
            assertSuccess(txHandler.getResult(hash));
        }
    }

    protected static void ensureIcxBalance(TransactionHandler txHandler, Address address,
                                           BigInteger oldVal, BigInteger newVal) throws Exception {
        long limitTime = System.currentTimeMillis() + Constants.DEFAULT_WAITING_TIME;
        while (true) {
            BigInteger icxBalance = txHandler.getBalance(address);
            String msg = "ICX balance of " + address + ": " + icxBalance;
            if (icxBalance.equals(oldVal)) {
                if (limitTime < System.currentTimeMillis()) {
                    throw new ResultTimeoutException();
                }
                try {
                    // wait until block confirmation
                    LOG.debug(msg + "; Retry in 1 sec.");
                    Thread.sleep(1000);
                } catch (InterruptedException e) {
                    e.printStackTrace();
                }
            } else if (icxBalance.equals(newVal)) {
                LOG.info(msg);
                break;
            } else {
                throw new IOException(String.format("ICX balance mismatch: expected <%s>, but was <%s>",
                        newVal, icxBalance));
            }
        }
    }
}
