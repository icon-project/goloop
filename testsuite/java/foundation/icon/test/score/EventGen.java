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
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;

import java.io.IOException;
import java.math.BigInteger;

public class EventGen extends Score {
    public EventGen(Score other) {
        super(other);
    }

    // install with the default parameter
    public static EventGen install(TransactionHandler txHandler, Wallet wallet, String contentType)
            throws TransactionFailureException, ResultTimeoutException, IOException {
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("EventGen"))
                .build();
        return install(txHandler, wallet, contentType, params);
    }

    // install with the passed parameter
    public static EventGen install(TransactionHandler txHandler,
                                   Wallet wallet, String contentType, RpcObject params)
            throws TransactionFailureException, ResultTimeoutException, IOException {
        if (contentType.equals(Constants.CONTENT_TYPE_PYTHON)) {
            return new EventGen(txHandler.deploy(wallet, getFilePath("event_gen"), params));
        } else if (contentType.equals(Constants.CONTENT_TYPE_JAVA)){
            return new EventGen(txHandler.deploy(wallet, testcases.EventGen.class, params));
        } else {
            throw new IllegalArgumentException("InvalidContentType("+contentType+")");
        }
    }

    public Bytes invokeGenerate(Wallet from, Address addr, BigInteger i, byte[] bytes) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_addr", new RpcValue(addr))
                .put("_int", new RpcValue(i))
                .put("_bytes", new RpcValue(bytes))
                .build();
        return invoke(from, "generate", params);
    }

    public TransactionResult invokeGenerateAndWait(Wallet from, Address addr, BigInteger i, byte[] bytes)
            throws ResultTimeoutException, IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_addr", new RpcValue(addr))
                .put("_int", new RpcValue(i))
                .put("_bytes", new RpcValue(bytes))
                .build();
        return invokeAndWaitResult(from, "generate", params);
    }
}
