package i;


/**
 * Indicates an internal runtime unexpected condition, especially for AVM execution rule violation.
 *
 * Note:  This class extends {@link RuntimeException} since they are not expected to be caught, but to unwind the
 * stack.  This means that we are expecting to force our way out of the user code, as quickly as possible, and
 * catch this only at the top-level entry-point we control.
 * Depending on the severity of the problem, this either implies a failure of the contract or a failure of the
 * node where we are trying to run.
 */
public abstract class AvmException extends AvmThrowable {
    private static final long serialVersionUID = 1L;

    protected AvmException() {
        super();
    }

    protected AvmException(String message) {
        super(message);
    }

    protected AvmException(String message, Throwable cause) {
        super(message, cause);
    }

    protected AvmException(Throwable cause) {
        super(cause);
    }

    public abstract int getCode();

    public abstract String getResultMessage();
}
