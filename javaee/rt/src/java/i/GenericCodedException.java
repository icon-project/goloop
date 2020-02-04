package i;

import foundation.icon.ee.types.CodedException;

public class GenericCodedException extends CodedException {
    private int code;

    public GenericCodedException() {
        super();
    }

    public GenericCodedException(int code) {
        super();
        this.code = code;
    }

    public GenericCodedException(int code, String message) {
        super(message);
        this.code = code;
    }

    public GenericCodedException(int code, String message, Throwable cause) {
        super(message, cause);
        this.code = code;
    }

    public GenericCodedException(int code, Throwable cause) {
        super(cause);
        this.code = code;
    }

    public int getCode() {
        return code;
    }
}
