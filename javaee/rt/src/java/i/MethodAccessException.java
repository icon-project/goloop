package i;


/**
 * An exception which the user can't catch since it refers to an ABI problem or other method access which is a static error
 * on the part of us or the user who deployed the DApp.
 */
public class MethodAccessException extends AvmException {
    private static final long serialVersionUID = 1L;

    public MethodAccessException() {
        super();
    }

    public MethodAccessException(String message) {
        super(message);
    }

    public MethodAccessException(String message, Throwable cause) {
        super(message, cause);
    }

    public MethodAccessException(Throwable cause) {
        super(cause);
    }
}
