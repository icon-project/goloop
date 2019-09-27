package i;


/**
 * This is a specific sub-tree of the {#link InternalError} hierarchy, specifically designed to describe
 * a node-level failure.  That is, these cases are so severe that we are expecting not even fail the
 * contract code, but drop it and bring the system down.
 * These cases are so severe that calling "System.exit()" open wanting to instantiate one would be a
 * reasonable implementation.
 */
public abstract class AvmError extends AvmThrowable {
    private static final long serialVersionUID = 1L;

    protected AvmError() {
        super();
    }

    protected AvmError(String message) {
        super(message);
    }

    protected AvmError(String message, Throwable cause) {
        super(message, cause);
    }

    protected AvmError(Throwable cause) {
        super(cause);
    }
}
