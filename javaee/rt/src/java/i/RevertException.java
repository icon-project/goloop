package i;

import foundation.icon.ee.types.SystemException;

public class RevertException extends SystemException {
    private int code;

    public RevertException() {
        super();
    }

    public RevertException(int code) {
        super();
        this.code = code;
    }

    public RevertException(int code, String message) {
        super(message);
        this.code = code;
    }

    public RevertException(int code, String message, Throwable cause) {
        super(message, cause);
        this.code = code;
    }

    public RevertException(int code, Throwable cause) {
        super(cause);
        this.code = code;
    }

    public int getCode() {
        return code;
    }
}
