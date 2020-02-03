package avm;

public class RevertException extends RuntimeException {
    public RevertException() {
        super();
    }

    public RevertException(String message) {
        super(message);
    }

    public RevertException(String message, Throwable cause) {
        super(message, cause);
    }

    public RevertException(Throwable cause) {
        super(cause);
    }
}
