package i;

import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.CodedException;

/**
 * Error that indicates the DApp exceeds the internal call depth limit.
 */
public class CallDepthLimitExceededException extends CodedException {
    private static final long serialVersionUID = 1L;

    public CallDepthLimitExceededException(String msg) {
        super(msg);
    }

    public int getCode() {
        return Status.StackOverflow;
    }
}
