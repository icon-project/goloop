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

package foundation.icon.test.score;

import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;

import java.io.IOException;
import java.math.BigInteger;

public class BTP2 extends Score {
    public BTP2(Score other) {
        super(other);
    }

    public static BTP2 install(TransactionHandler txHandler, Wallet wallet, Address address)
            throws TransactionFailureException, ResultTimeoutException, IOException {
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(address))
                .build();
        return new BTP2(txHandler.deploy(wallet, testcases.BTP2.class, params));
    }

    public TransactionResult sendAndRevert(Wallet from, BigInteger nid, byte[] msg, BigInteger msgCount, BigInteger revertNid)
            throws ResultTimeoutException, IOException {
        RpcObject params = new RpcObject.Builder()
                .put("nid", new RpcValue(nid))
                .put("msg", new RpcValue(msg))
                .put("msgCount", new RpcValue(msgCount))
                .put("revertNid", new RpcValue(revertNid))
                .build();
        return invokeAndWaitResult(from, "sendAndRevert", params, null, Constants.DEFAULT_STEPS);
    }
}
