package foundation.icon.ee.types;

import i.AvmException;

public abstract class CodedException extends AvmException {
    public CodedException() {
        super();
    }

    public CodedException(String message) {
        super(message);
    }

    public CodedException(String message, Throwable cause) {
        super(message, cause);
    }

    public CodedException(Throwable cause) {
        super(cause);
    }

    public abstract int getCode();
}
