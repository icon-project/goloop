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

import foundation.icon.icx.Transaction;
import foundation.icon.icx.TransactionBuilder;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;

import java.io.IOException;
import java.math.BigInteger;

public class FeeShareScore extends Score {
    private static final BigInteger STEPS = Constants.DEFAULT_STEPS;
    private final Wallet wallet;

    public FeeShareScore(Score other, Wallet wallet) {
        super(other);
        this.wallet = wallet;
    }

    public static FeeShareScore mustDeploy(TransactionHandler txHandler, Wallet ownerWallet)
            throws ResultTimeoutException, TransactionFailureException, IOException {
        return new FeeShareScore(
                txHandler.deploy(ownerWallet, getFilePath("fee_sharing"), null), ownerWallet);
    }

    public String getValue() throws IOException {
        RpcItem res = this.call("getValue", null);
        return res.asString();
    }

    public TransactionResult addToWhitelist(Address address, int proportion) throws IOException, ResultTimeoutException {
        return invokeAndWaitResult(wallet,
                "addToWhitelist",
                (new RpcObject.Builder())
                        .put("address", new RpcValue(address))
                        .put("proportion", new RpcValue(BigInteger.valueOf(proportion)))
                        .build());
    }

    public TransactionResult setValue(String value) throws IOException, ResultTimeoutException {
        return invokeAndWaitResult(wallet,
                "setValue",
                (new RpcObject.Builder())
                        .put("value", new RpcValue(value))
                        .build());
    }

    public TransactionResult addDeposit(BigInteger depositAmount)
            throws IOException, ResultTimeoutException {
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(getNetworkId())
                .from(wallet.getAddress())
                .to(getAddress())
                .value(depositAmount)
                .stepLimit(STEPS)
                .deposit()
                .add()
                .build();
        Bytes txHash = invoke(wallet, transaction);
        return getResult(txHash);
    }

    public TransactionResult withdrawDeposit(Bytes depositId)
            throws IOException, ResultTimeoutException {
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(getNetworkId())
                .from(wallet.getAddress())
                .to(getAddress())
                .stepLimit(STEPS)
                .deposit()
                .withdraw(depositId)
                .build();
        Bytes txHash = invoke(wallet, transaction);
        return getResult(txHash);
    }
}
