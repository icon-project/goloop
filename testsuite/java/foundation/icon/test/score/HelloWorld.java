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
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;

import java.io.IOException;

public class HelloWorld extends Score {
    public static final String INSTALL_PATH = Constants.SCORE_ROOT + "hello_world";
    public static final String UPDATE_PATH = Constants.SCORE_ROOT + "hello_world2";

    public HelloWorld(TransactionHandler txHandler, Address address) {
        super(txHandler, address);
    }

    public HelloWorld(Score other) {
        super(other);
    }

    public static HelloWorld install(TransactionHandler txHandler, Wallet wallet)
            throws TransactionFailureException, ResultTimeoutException, IOException {
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        return install(txHandler, wallet, params);
    }

    public static HelloWorld install(TransactionHandler txHandler, Wallet wallet, RpcObject params)
            throws TransactionFailureException, ResultTimeoutException, IOException {
        return new HelloWorld(txHandler.deploy(wallet, INSTALL_PATH, params));
    }

    public TransactionResult invokeHello(Wallet from) throws ResultTimeoutException, IOException {
        return invokeAndWaitResult(from, "hello", null);
    }
}
