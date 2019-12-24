package foundation.icon.ee.types;

import i.AvmException;

public class SystemException extends AvmException {
    private int status = Status.UnknownFailure;

    public SystemException() {
        super();
    }

    public SystemException(String message) {
        super(message);
    }

    public SystemException(String message, Throwable cause) {
        super(message, cause);
    }

    public SystemException(Throwable cause) {
        super(cause);
    }

    public SystemException(int status) {
        super();
        this.status = status;
    }

    public SystemException(int status, String message) {
        super(message);
        this.status = status;
    }

    public SystemException(int status, String message, Throwable cause) {
        super(message, cause);
        this.status = status;
    }

    public SystemException(int status, Throwable cause) {
        super(cause);
        this.status = status;
    }

    public int getStatus() {
        return status;
    }
}
