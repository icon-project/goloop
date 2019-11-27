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
    private static final Log LOG = Log.getGlobal();

    protected IconService service;
    protected Address scoreAddress;
    protected Env.Chain chain;

    public Score(IconService service, Env.Chain chain, Address scoreAddress) {
        this.service = service;
        this.chain = chain;
        this.scoreAddress = scoreAddress;
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
        return install(service, chain, wallet, contentPath, params, stepLimit, Constants.CONTENT_TYPE_ZIP);
    }

    public void update(IconService service, Env.Chain chain, Wallet wallet, String contentPath, RpcObject params)
            throws TransactionFailureException, ResultTimeoutException, IOException {
        Bytes txHash = Utils.deployScore(service, chain.networkId, wallet, this.scoreAddress, contentPath, params);
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
                .nid(BigInteger.valueOf(chain.networkId))
                .from(wallet.getAddress())
                .to(this.scoreAddress)
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
        return Utils.getTransactionResult(this.service, txHash, Constants.DEFAULT_WAITING_TIME);
    }

    public TransactionResult waitResult(Bytes txHash, long waiting) throws ResultTimeoutException, IOException {
        return Utils.getTransactionResult(this.service, txHash, waiting);
    }

    public Address getAddress() {
        return this.scoreAddress;
    }

    public void setAddress(Address addr) {
        this.scoreAddress = addr;
    }

    @Override
    public String toString() {
        return "SCORE(" + this.scoreAddress.toString() + ")";
    }
}
