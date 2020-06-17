package foundation.icon.ee.types;

public class IllegalFormatException extends PredefinedException {
    public IllegalFormatException() {
        this(Status.getMessage(Status.IllegalFormat));
    }

    public IllegalFormatException(String message) {
        super(message);
    }

    public IllegalFormatException(String message, Throwable cause) {
        super(message, cause);
    }

    public IllegalFormatException(Throwable cause) {
        this(Status.getMessage(Status.IllegalFormat), cause);
    }

    public int getCode() {
        return Status.IllegalFormat;
    }
}
