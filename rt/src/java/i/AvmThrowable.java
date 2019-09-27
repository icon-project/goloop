package i;

/**
 * The root of the exception/error hierarchy in AVM. The DApp should not be able
 * to catch any of the exceptions or errors which extend from this class.
 */
public class AvmThrowable extends RuntimeException {
    private static final long serialVersionUID = 1L;

    protected AvmThrowable() {
        super();
    }

    protected AvmThrowable(String message) {
        super(message);
    }

    protected AvmThrowable(String message, Throwable cause) {
        super(message, cause);
    }

    protected AvmThrowable(Throwable cause) {
        super(cause);
    }
}
