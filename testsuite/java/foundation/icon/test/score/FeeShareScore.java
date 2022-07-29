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
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;
import testcases.FeeSharing;

import java.io.IOException;
import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;

public class FeeShareScore extends Score {
    private static final BigInteger STEPS = Constants.DEFAULT_STEPS;
    private final Wallet wallet;

    public FeeShareScore(Score other, Wallet wallet) {
        super(other);
        this.wallet = wallet;
    }

    public static FeeShareScore mustDeploy(TransactionHandler txHandler, Wallet ownerWallet, String contentType)
            throws ResultTimeoutException, TransactionFailureException, IOException {
        LOG.infoEntering("deploy", "FeeSharing");
        Score score;
        if (contentType.equals(Constants.CONTENT_TYPE_PYTHON)) {
            score = txHandler.deploy(ownerWallet, getFilePath("fee_sharing"), null);
        } else if (contentType.equals(Constants.CONTENT_TYPE_JAVA)) {
            score = txHandler.deploy(ownerWallet, FeeSharing.class, null);
        } else {
            throw new IllegalArgumentException("Unknown content type");
        }
        LOG.info("scoreAddr = " + score.getAddress());
        LOG.infoExiting();
        return new FeeShareScore(score, ownerWallet);
    }

    public String getValue() throws IOException {
        return this.call("getValue", null).asString();
    }

    public BigInteger getProportion(Address address) throws IOException {
        var params = new RpcObject.Builder()
                .put("address", new RpcValue(address))
                .build();
        return this.call("getProportion", params).asInteger();
    }

    public TransactionResult addToWhitelist(Address address, BigInteger proportion)
            throws IOException, ResultTimeoutException {
        return invokeAndWaitResult(wallet,
                "addToWhitelist",
                (new RpcObject.Builder())
                        .put("address", new RpcValue(address))
                        .put("proportion", new RpcValue(proportion))
                        .build());
    }

    public TransactionResult setValue(String value) throws IOException, ResultTimeoutException {
        return invokeAndWaitResult(wallet,
                "setValue",
                (new RpcObject.Builder())
                        .put("value", new RpcValue(value))
                        .build());
    }

    public TransactionResult setValues(String value, Address[] others) throws IOException, ResultTimeoutException {
        RpcItem othersValue = null;
        if (others != null) {
            var builder = new RpcArray.Builder();
            for (int i = 0; i < others.length; i++) {
                builder.add(new RpcValue(others[i]));
            }
            othersValue = builder.build();
        }
        return invokeAndWaitResult(wallet,
                "setValues",
                (new RpcObject.Builder())
                        .put("value", new RpcValue(value))
                        .put("others", othersValue)
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

    public TransactionResult withdrawDeposit(BigInteger amount)
            throws IOException, ResultTimeoutException {
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(getNetworkId())
                .from(wallet.getAddress())
                .to(getAddress())
                .stepLimit(STEPS)
                .deposit()
                .withdraw(amount)
                .build();
        Bytes txHash = invoke(wallet, transaction);
        return getResult(txHash);
    }

    public TransactionResult withdrawDeposit()
            throws IOException, ResultTimeoutException {
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(getNetworkId())
                .from(wallet.getAddress())
                .to(getAddress())
                .stepLimit(STEPS)
                .deposit()
                .withdraw()
                .build();
        Bytes txHash = invoke(wallet, transaction);
        return getResult(txHash);
    }
}
