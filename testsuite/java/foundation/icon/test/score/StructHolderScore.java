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

package foundation.icon.test.score;

import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;
import testcases.StructHolder;

import java.io.IOException;

import static foundation.icon.test.common.Env.LOG;

public class StructHolderScore extends Score {
    public StructHolderScore(Score other) {
        super(other);
    }

    public static StructHolderScore mustDeploy(TransactionHandler txHandler,
            Wallet owner)
            throws ResultTimeoutException, TransactionFailureException,
            IOException {
        LOG.infoEntering("deploy", "StructHolder");
        RpcObject params = new RpcObject.Builder()
                .build();
        Score score = txHandler.deploy(
                owner,
                new Class<?>[]{ StructHolder.class },
                params
        );
        LOG.info("scoreAddr = " + score.getAddress());
        LOG.infoExiting();
        return new StructHolderScore(score);
    }

    public TransactionResult setSimpleStruct(Wallet from, RpcObject params)
            throws IOException, ResultTimeoutException {
        return invokeAndWaitResult(from, "setSimpleStruct", params, null, Constants.DEFAULT_STEPS);
    }

    public TransactionResult setComplexStruct(Wallet from, RpcObject params)
            throws IOException, ResultTimeoutException {
        return invokeAndWaitResult(from, "setComplexStruct", params);
    }
}
