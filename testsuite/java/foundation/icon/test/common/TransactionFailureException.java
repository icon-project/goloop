package foundation.icon.test.common;

import foundation.icon.icx.data.TransactionResult;

import java.math.BigInteger;

public class TransactionFailureException extends Exception {
    private TransactionResult.Failure failure;

    public TransactionFailureException(TransactionResult.Failure failure) {
        this.failure = failure;
    }

    @Override
    public String toString() {
        return this.failure.toString();
    }

    public BigInteger getCode() {
        return this.failure.getCode();
    }

    public String getMessage() {
        return this.failure.getMessage();
    }
}
