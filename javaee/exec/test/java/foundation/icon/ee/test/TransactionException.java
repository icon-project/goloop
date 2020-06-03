package foundation.icon.ee.test;

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
}
