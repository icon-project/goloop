package i;

import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.SystemException;

/**
 * Error that indicates the DApp runs out of stack.
 */
public class OutOfStackException extends SystemException {
    private static final long serialVersionUID = 1L;

    public int getCode() {
        return Status.StackOverflow;
    }
}
