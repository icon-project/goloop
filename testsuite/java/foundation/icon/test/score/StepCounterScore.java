/*
 * Copyright 2019 ICON Foundation
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

package foundation.icon.test.score;

import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;

import java.io.IOException;
import java.math.BigInteger;

public class StepCounterScore extends Score {
    private static final BigInteger STEPS = BigInteger.valueOf(200000);

    public StepCounterScore(Score other) {
        super(other);
    }

    public static StepCounterScore mustDeploy(TransactionHandler txHandler, Wallet wallet)
            throws IOException, TransactionFailureException, ResultTimeoutException {
        return new StepCounterScore(
                txHandler.deploy(wallet, getFilePath("step_counter"), null)
        );
    }

    public TransactionResult increaseStep(Wallet wallet) throws ResultTimeoutException, IOException {
        return this.invokeAndWaitResult(wallet,
                "increaseStep", null, null, STEPS);
    }

    public TransactionResult setStep(Wallet wallet, BigInteger step) throws ResultTimeoutException, IOException {
        return this.invokeAndWaitResult(wallet,
                "setStep",
                (new RpcObject.Builder())
                    .put("step", new RpcValue(step))
                    .build(),
                null, STEPS);
    }

    public TransactionResult resetStep(Wallet wallet, BigInteger step) throws ResultTimeoutException, IOException {
        return this.invokeAndWaitResult(wallet,
                "resetStep",
                (new RpcObject.Builder())
                        .put("step", new RpcValue(step))
                        .build(),
                null, STEPS);
    }

    public TransactionResult setStepOf(Wallet wallet, Address target, BigInteger step) throws ResultTimeoutException, IOException {
        return this.invokeAndWaitResult(wallet,
                "setStepOf",
                (new RpcObject.Builder())
                    .put("step", new RpcValue(step))
                    .put("addr", new RpcValue(target))
                    .build(),
                null, STEPS);
    }

    public TransactionResult trySetStepWith(Wallet wallet, Address target, BigInteger step) throws ResultTimeoutException, IOException {
        return this.invokeAndWaitResult(wallet,
                "trySetStepWith",
                (new RpcObject.Builder())
                        .put("step", new RpcValue(step))
                        .put("addr", new RpcValue(target))
                        .build(),
                null, STEPS);
    }

    public BigInteger getStep(Address from) throws IOException {
        RpcItem res = this.call("getStep", null);
        return res.asInteger();
    }
}
