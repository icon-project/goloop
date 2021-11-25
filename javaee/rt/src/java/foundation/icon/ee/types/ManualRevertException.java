package foundation.icon.ee.types;

import i.AvmException;

public class ManualRevertException extends AvmException {
    private final int code;

    public ManualRevertException(int code) {
        super();
        this.code = code;
    }

    public ManualRevertException(int code, String message) {
        super(message);
        this.code = code;
    }

    public ManualRevertException(int code, Throwable cause) {
        super(cause);
        this.code = code;
    }

    public ManualRevertException(int code, String message, Throwable cause) {
        super(message, cause);
        this.code = code;
    }

    public int getCode() {
        return code;
    }

    public String getResultMessage() {
        var m = getMessage();
        return m != null ? m : Status.getMessage(getCode());
    }
}
