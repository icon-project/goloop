package i;

public class InvalidDBAccessException extends AvmException {
    public InvalidDBAccessException() {
    }

    public InvalidDBAccessException(String msg) {
        super(msg);
    }

    public InvalidDBAccessException(String message, Throwable cause) {
        super(message, cause);
    }

    public InvalidDBAccessException(Throwable cause) {
        super(cause);
    }
}
