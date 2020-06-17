package foundation.icon.ee.types;

public class UnknownFailureException extends PredefinedException {
    public UnknownFailureException() {
        super();
    }

    public UnknownFailureException(String message) {
        super(message);
    }

    public UnknownFailureException(String message, Throwable cause) {
        super(message, cause);
    }

    public UnknownFailureException(Throwable cause) {
        super(cause);
    }

    public int getCode() {
        return Status.UnknownFailure;
    }
}
