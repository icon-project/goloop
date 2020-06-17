package i;

import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.PredefinedException;

/**
 * Error that indicates the DApp runs out of energy.
 */
public class OutOfEnergyException extends PredefinedException {
    private static final long serialVersionUID = 1L;

    public int getCode() {
        return Status.OutOfStep;
    }
}
