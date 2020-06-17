package i;

import foundation.icon.ee.types.PredefinedException;

public class GenericPredefinedException extends PredefinedException {
    private int code;

    public GenericPredefinedException() {
        super();
    }

    public GenericPredefinedException(int code) {
        super();
        this.code = code;
    }

    public GenericPredefinedException(int code, String message) {
        super(message);
        this.code = code;
    }

    public GenericPredefinedException(int code, String message, Throwable cause) {
        super(message, cause);
        this.code = code;
    }

    public GenericPredefinedException(int code, Throwable cause) {
        super(cause);
        this.code = code;
    }

    public int getCode() {
        return code;
    }
}
