package foundation.icon.ee.types;

import i.AvmException;

public abstract class SystemException extends AvmException {
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

    public abstract int getCode();
}
