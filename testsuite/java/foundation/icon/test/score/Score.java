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

import foundation.icon.icx.Call;
import foundation.icon.icx.IconService;
import foundation.icon.icx.Transaction;
import foundation.icon.icx.TransactionBuilder;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.ScoreApi;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Log;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.common.Utils;

import java.io.IOException;
import java.math.BigInteger;
import java.util.List;

public class Score {
    private static final Log LOG = Log.getGlobal();
    private final TransactionHandler txHandler;
    private Address address;

    public Score(IconService service, Env.Chain chain, Address scoreAddress) {
        this.txHandler = new TransactionHandler(service, chain);
        this.address = scoreAddress;
    }

    public Score(TransactionHandler txHandler, Address scoreAddress) {
        this.txHandler = txHandler;
        this.address = scoreAddress;
    }

    public Score(Score other) {
        this(other.txHandler, other.address);
    }

    public static Address install(IconService service, Env.Chain chain, Wallet wallet, String contentPath, RpcObject params)
            throws IOException, TransactionFailureException, ResultTimeoutException {
        return install(service, chain, wallet, contentPath, params, Constants.DEFAULT_STEP_LIMIT);
    }

    public static Address install(IconService service,
                                  Env.Chain chain, Wallet wallet, String contentPath, RpcObject params, long stepLimit, String contentType)
            throws IOException, TransactionFailureException, ResultTimeoutException {
        Bytes txHash = Utils.deployScore(service, chain.networkId, wallet, Constants.CHAINSCORE_ADDRESS, contentPath, params, stepLimit, contentType);
        TransactionResult result = Utils.getTransactionResult(service, txHash, Constants.DEFAULT_WAITING_TIME);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new TransactionFailureException(result.getFailure());
        }

        try {
            Utils.acceptIfAuditEnabled(service, chain, txHash);
        }
        catch(TransactionFailureException ex) {
            LOG.infoExiting();
            throw ex;
        }
        return new Address(result.getScoreAddress());
    }

    public static Address install(IconService service, Env.Chain chain, Wallet wallet, String contentPath, RpcObject params, long stepLimit)
            throws IOException, TransactionFailureException, ResultTimeoutException {
        return install(service, chain, wallet, contentPath, params, stepLimit, Constants.CONTENT_TYPE_PYTHON);
    }

    public void update(IconService service, Env.Chain chain, Wallet wallet, String contentPath, RpcObject params)
            throws TransactionFailureException, ResultTimeoutException, IOException {
        Bytes txHash = Utils.deployScore(service, chain.networkId, wallet, getAddress(), contentPath, params);
        TransactionResult result = Utils.getTransactionResult(service, txHash, Constants.DEFAULT_WAITING_TIME);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new TransactionFailureException(result.getFailure());
        }
        try {
            Utils.acceptIfAuditEnabled(service, chain, txHash);
        }
        catch(TransactionFailureException ex) {
            LOG.infoExiting();
            throw ex;
        }
    }

    public RpcItem call(String method, RpcObject params)
            throws IOException {
        if (params == null) {
            params = new RpcObject.Builder().build();
        }
        Call<RpcItem> call = new Call.Builder()
                .to(getAddress())
                .method(method)
                .params(params)
                .build();
        return this.txHandler.call(call);
    }

    public Bytes invoke(Wallet wallet, String method, RpcObject params,
                        long value, long steps) throws IOException {
        return invoke(wallet, method, params, BigInteger.valueOf(value), BigInteger.valueOf(steps));
    }

    public Bytes invoke(Wallet wallet, String method, RpcObject params,
                        BigInteger value, BigInteger steps) throws IOException {
        return invoke(wallet, method, params, value, steps, null, null);
    }

    public Bytes invoke(Wallet wallet, String method, RpcObject params, BigInteger value,
                        BigInteger steps, BigInteger timestamp, BigInteger nonce) throws IOException {
        TransactionBuilder.Builder builder = TransactionBuilder.newBuilder()
                .nid(getNetworkId())
                .from(wallet.getAddress())
                .to(getAddress())
                .stepLimit(steps);

        if ((value != null) && value.bitLength() != 0) {
            builder.value(value);
        }
        if ((timestamp != null) && timestamp.bitLength() != 0) {
            builder.timestamp(timestamp);
        }
        if ((nonce != null) && nonce.bitLength() != 0) {
            builder.nonce(nonce);
        }

        Transaction t;
        if (params != null) {
            t = builder.call(method).params(params).build();
        } else {
            t = builder.call(method).build();
        }
        return this.txHandler.invoke(wallet, t);
    }

    public TransactionResult invokeAndWaitResult(Wallet wallet, String method,
                                                 RpcObject params, long value, long steps)
            throws ResultTimeoutException, IOException {
        Bytes txHash = this.invoke(wallet, method, params, value, steps);
        return waitResult(txHash);
    }

    public TransactionResult invokeAndWaitResult(Wallet wallet, String method,
                                                 RpcObject params, BigInteger value, BigInteger steps)
            throws ResultTimeoutException, IOException {
        Bytes txHash = this.invoke(wallet, method, params, value, steps);
        return waitResult(txHash);
    }

    public TransactionResult waitResult(Bytes txHash) throws ResultTimeoutException, IOException {
        return waitResult(txHash, Constants.DEFAULT_WAITING_TIME);
    }

    public TransactionResult waitResult(Bytes txHash, long waiting) throws ResultTimeoutException, IOException {
        return this.txHandler.getTransactionResult(txHash, waiting);
    }

    public Address getAddress() {
        return this.address;
    }

    public void setAddress(Address addr) {
        this.address = addr;
    }

    public BigInteger getNetworkId() {
        return txHandler.getNetworkId();
    }

    public List<ScoreApi> getScoreApi() throws IOException {
        return txHandler.getScoreApi(getAddress());
    }

    @Override
    public String toString() {
        return "SCORE(" + getAddress().toString() + ")";
    }
}
