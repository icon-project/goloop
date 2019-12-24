package i;

public class RevertException extends AvmException {
    private int code;

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

    public int getCode() {
        return code;
    }
}
