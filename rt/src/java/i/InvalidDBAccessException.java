package i;

public class InvalidDBAccessException extends AvmException {
    public InvalidDBAccessException() {
    }

    public InvalidDBAccessException(String msg) {
        super(msg);
    }
}
