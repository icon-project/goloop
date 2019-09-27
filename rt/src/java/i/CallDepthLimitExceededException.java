package i;

/**
 * Error that indicates the DApp exceeds the internal call depth limit.
 */
public class CallDepthLimitExceededException extends AvmException {
    private static final long serialVersionUID = 1L;

    public CallDepthLimitExceededException(String message) {
        super(message);
    }
}
