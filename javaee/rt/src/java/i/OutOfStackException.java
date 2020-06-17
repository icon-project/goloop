package i;

import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.PredefinedException;

/**
 * Error that indicates the DApp runs out of stack.
 */
public class OutOfStackException extends PredefinedException {
    private static final long serialVersionUID = 1L;

    public int getCode() {
        return Status.StackOverflow;
    }
}
