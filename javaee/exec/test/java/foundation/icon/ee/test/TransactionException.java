package foundation.icon.ee.test;

import foundation.icon.ee.types.Result;

public class TransactionException extends RuntimeException {
    private final Result result;

    public TransactionException(Result result) {
        this.result = result;
    }

    public TransactionException(String message, Result result) {
        super(message);
        this.result = result;
    }

    public TransactionException(String message, Throwable cause, Result result) {
        super(message, cause);
        this.result = result;
    }

    public TransactionException(Throwable cause, Result result) {
        super(cause);
        this.result = result;
    }

    public Result getResult() {
        return result;
    }

    public String toString() {
        return String.format("TransactionException{result:{%s}}", result);
    }
}
