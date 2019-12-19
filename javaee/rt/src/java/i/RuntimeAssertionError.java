package i;


/**
 * A class of our fatal errors specifically for internal assertion failures.
 * Instances of this class are only intended to be instantiated from its static helpers.
 * 
 * Note that the methods which throw on all paths also define that they will return the exception, so the caller can satisfy
 * the compiler by throwing the response (important for reachability detection).
 * This idea is useful in cases where all paths throw.
 */
public class RuntimeAssertionError extends AvmError {
    private static final long serialVersionUID = 1L;

    /**
     * Verifies that a statement is true, throwing RuntimeAssertionError if not.
     * 
     * @param statement The statement to check.
     */
    public static void assertTrue(boolean statement) {
        if (!statement) {
            throw new RuntimeAssertionError("Statement MUST be true");
        }
    }

    /**
     * Called when a throwable is encountered at a point where it should have been handled earlier, such that the catch point
     * can only treat it as a fatal exception
     * 
     * @param t The unexpected throwable.
     * @return The thrown exception (for caller reachability convenience).
     */
    public static RuntimeAssertionError unexpected(Throwable t) {
        throw new RuntimeAssertionError("Unexpected Throwable", t);
    }

    /**
     * Called when a code-path thought impossible to enter is executed.  In general, this is used to denote that an interface
     * method is not called in a certain configuration/implementation.
     * 
     * @param message The message explaining why this shouldn't be called.
     * @return The thrown exception (for caller reachability convenience).
     */
    public static RuntimeAssertionError unreachable(String message) {
        throw new RuntimeAssertionError("Unreachable code reached: " + message);
    }

    /**
     * Note that unimplemented paths are mostly just to enable incremental development and deep
     * prototyping.  Cutting off a path with unimplemented will make it easier for us to find, later.
     * All callers of this should be implemented (or commuted to a more specific assertion) before production use.
     * 
     * @param message The message to describe why this is unimplemented.
     */
    public static RuntimeAssertionError unimplemented(String message) {
        throw new RuntimeAssertionError("Unimplemented path: " + message);
    }


    private RuntimeAssertionError(String message) {
        super(message);
    }

    private RuntimeAssertionError(String message, Throwable cause) {
        super(message, cause);
    }
}
