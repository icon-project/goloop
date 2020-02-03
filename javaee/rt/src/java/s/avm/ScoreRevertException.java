package s.avm;

import foundation.icon.ee.types.Status;
import s.java.lang.String;
import s.java.lang.Throwable;

public class ScoreRevertException extends RevertException {
    private static final int End = Status.UserReversionEnd - Status.UserReversionStart;

    private int statusCode;

    public ScoreRevertException() {
        super();
    }

    public ScoreRevertException(String message) {
        super(message);
    }

    public ScoreRevertException(String message, Throwable cause) {
        super(message, cause);
    }

    public ScoreRevertException(Throwable cause) {
        super(cause);
    }

    private void assumeValidCode(int code) {
        if (code < 0 || code >= End) {
            throw new IllegalArgumentException("invalid code " + code);
        }
    }

    public ScoreRevertException(int code) {
        super();
        assumeValidCode(code);
        statusCode = code;
    }

    public ScoreRevertException(int code, String message) {
        super(message);
        assumeValidCode(code);
        statusCode = code;
    }

    public ScoreRevertException(int code, String message, Throwable cause) {
        super(message, cause);
        assumeValidCode(code);
        statusCode = code;
    }

    public ScoreRevertException(int code, Throwable cause) {
        super(cause);
        assumeValidCode(code);
        statusCode = code;
    }

    public int getCode() {
        return statusCode;
    }
}
