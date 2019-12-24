package i;

import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.SystemException;

/**
 * Error that indicates the DApp exceeds the internal call depth limit.
 */
public class CallDepthLimitExceededException extends SystemException {
    private static final long serialVersionUID = 1L;

    public CallDepthLimitExceededException(String message) {
        super(Status.StackOverflow, message);
    }
}
