package s.avm;

import foundation.icon.ee.types.Status;
import s.java.lang.RuntimeException;
import s.java.lang.String;
import s.java.lang.Throwable;

public class TargetRevertedException extends RuntimeException {
    private static final int End = Status.UserReversionEnd - Status.UserReversionStart;

    private int statusCode;

    public TargetRevertedException() {
        super();
    }

    public TargetRevertedException(String message) {
        super(message);
    }

    public TargetRevertedException(String message, Throwable cause) {
        super(message, cause);
    }

    public TargetRevertedException(Throwable cause) {
        super(cause);
    }

    private void assumeValidCode(int code) {
        if (code < 0 || code >= End) {
            throw new IllegalArgumentException("invalid code " + code);
        }
    }

    public TargetRevertedException(int code) {
        super();
        assumeValidCode(code);
        statusCode = code;
    }

    public TargetRevertedException(int code, String message) {
        super(message);
        assumeValidCode(code);
        statusCode = code;
    }

    public TargetRevertedException(int code, String message, Throwable cause) {
        super(message, cause);
        assumeValidCode(code);
        statusCode = code;
    }

    public TargetRevertedException(int code, Throwable cause) {
        super(cause);
        assumeValidCode(code);
        statusCode = code;
    }

    public int getCode() {
        return statusCode;
    }
}
