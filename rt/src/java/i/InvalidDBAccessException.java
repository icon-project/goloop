package i;

public class InvalidDBAccessException extends AvmError {
    public InvalidDBAccessException() {
    }

    public InvalidDBAccessException(String msg) {
        super(msg);
    }
}
