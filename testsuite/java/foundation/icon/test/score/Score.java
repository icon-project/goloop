package foundation.icon.test.score;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.test.common.*;

import java.io.IOException;
import java.math.BigInteger;

public class Score {
    public static final BigInteger STEPS_DEFAULT = BigInteger.valueOf(2000000);
    public static final long DEFAULT_WAITING_TIME = 5000; // millisecond

    protected IconService service;
    protected Address scoreAddress;
    protected Env.Chain chain;

    public Score(IconService service, Env.Chain chain, Address scoreAddress) {
        this.service = service;
        this.chain = chain;
        this.scoreAddress = scoreAddress;
    }

    public static Address install(IconService service, Env.Chain chain, Wallet wallet, String filePath, RpcObject params)
            throws IOException, TransactionFailureException, ResultTimeoutException {
        return install(service, chain, wallet, filePath, params, -1);
    }

    public static Address install(IconService service, Env.Chain chain, Wallet wallet, String filePath, RpcObject params, long stepLimit)
            throws IOException, TransactionFailureException, ResultTimeoutException {
        Bytes txHash = Utils.installScore(service, chain, wallet, filePath, params, stepLimit);
        TransactionResult result = Utils.getTransactionResult(service, txHash, DEFAULT_WAITING_TIME);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new TransactionFailureException(result.getFailure());
        }
        return new Address(result.getScoreAddress());
    }

    public void update(IconService service, Env.Chain chain, Wallet wallet, String filePath, RpcObject params)
            throws TransactionFailureException, ResultTimeoutException, IOException {
        Bytes txHash = Utils.updateScore(service, chain, wallet, this.scoreAddress, filePath, params, -1);
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
                .nid(chain.networkId)
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
                .nid(chain.networkId)
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
