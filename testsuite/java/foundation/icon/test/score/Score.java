package foundation.icon.test.score;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.Utils;

import java.io.IOException;
import java.math.BigInteger;

public class Score {
    public static final BigInteger STEPS_DEFAULT = BigInteger.valueOf(2000000);
    public static final long DEFAULT_WAITING_TIME = 7000; // millisecond

    protected IconService service;
    protected Address scoreAddress;
    protected BigInteger nid;

    public Score(IconService service, Address scoreAddress, BigInteger nid) {
        this.service = service;
        this.scoreAddress = scoreAddress;
        this.nid = nid;
    }

    public static TransactionResult deployAndWaitResult(IconService service
            , Wallet wallet, String filePath, RpcObject params, long stepLimit)
            throws ResultTimeoutException, IOException {
        Bytes txHash = Utils.deployScore(service, wallet, filePath, params, stepLimit);
        return Utils.getTransactionResult(service, txHash, DEFAULT_WAITING_TIME);
    }

    public static Address mustDeploy(IconService service, Wallet wallet, String filePath, RpcObject params, long stepLimit)
            throws IOException, TransactionFailureException, ResultTimeoutException {
        TransactionResult result = deployAndWaitResult(service, wallet, filePath, params, stepLimit);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new TransactionFailureException(result.getFailure());
        }
        return new Address(result.getScoreAddress());
    }

    public static TransactionResult deployAndWaitResult(IconService service
            , Wallet wallet, String filePath, RpcObject params)
            throws ResultTimeoutException, IOException {
        Bytes txHash = Utils.deployScore(service, wallet, Constants.CHAINSCORE_ADDRESS, filePath, params);
        return Utils.getTransactionResult(service, txHash, DEFAULT_WAITING_TIME);
    }

    public static Address mustDeploy(IconService service, Wallet wallet, String filePath, RpcObject params)
            throws IOException, TransactionFailureException, ResultTimeoutException {
        TransactionResult result = deployAndWaitResult(service, wallet, filePath, params);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new TransactionFailureException(result.getFailure());
        }
        return new Address(result.getScoreAddress());
    }

    public void update(IconService service, Wallet wallet, String filePath, RpcObject params, BigInteger nid)
            throws TransactionFailureException, ResultTimeoutException, IOException {
        Bytes txHash = Utils.deployScore(service, wallet, this.scoreAddress, filePath, params);
        TransactionResult result = Utils.getTransactionResult(service, txHash, DEFAULT_WAITING_TIME);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new TransactionFailureException(result.getFailure());
        }
    }

    public RpcItem call(Address from, String method, RpcObject params)
            throws IOException {
        if (params == null) {
            params = new RpcObject.Builder().build();
        }
        Call<RpcItem> call = new Call.Builder()
                .from(from)
                .to(this.scoreAddress)
                .method(method)
                .params(params)
                .build();
        return this.service.call(call).execute();
    }

    public Bytes invoke(Wallet wallet, String method,
                        RpcObject params, long value, long steps)
            throws IOException {
        TransactionBuilder.Builder builder = TransactionBuilder.newBuilder()
                .nid(nid)
                .from(wallet.getAddress())
                .to(this.scoreAddress)
                .stepLimit(BigInteger.valueOf(steps));

        if (value != 0) {
            builder = builder.value(BigInteger.valueOf(value));
        }

        Transaction t = null;
        if (params != null) {
            t = builder.call(method).params(params).build();
        } else {
            t = builder.call(method).build();
        }

        return this.service
                .sendTransaction(new SignedTransaction(t, wallet))
                .execute();
    }

    public Bytes invoke(Wallet wallet, String method,
                        RpcObject params, BigInteger value, BigInteger steps)
            throws IOException {
        TransactionBuilder.Builder builder = TransactionBuilder.newBuilder()
                .nid(nid)
                .from(wallet.getAddress())
                .to(this.scoreAddress)
                .stepLimit(steps);

        if ((value != null) && value.bitLength() != 0) {
            builder = builder.value(value);
        }

        Transaction t = null;
        if (params != null) {
            t = builder.call(method).params(params).build();
        } else {
            t = builder.call(method).build();
        }

        return this.service
                .sendTransaction(new SignedTransaction(t, wallet))
                .execute();
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
        return Utils.getTransactionResult(this.service, txHash, DEFAULT_WAITING_TIME);
    }

    public TransactionResult waitResult(Bytes txHash, long waiting) throws ResultTimeoutException, IOException {
        return Utils.getTransactionResult(this.service, txHash, waiting);
    }

    public Address getAddress() {
        return this.scoreAddress;
    }

    @Override
    public String toString() {
        return "SCORE(" + this.scoreAddress.toString() + ")";
    }
}
