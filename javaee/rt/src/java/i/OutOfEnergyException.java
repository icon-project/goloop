package i;

import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.CodedException;

/**
 * Error that indicates the DApp runs out of energy.
 */
public class OutOfEnergyException extends CodedException {
    private static final long serialVersionUID = 1L;

    public int getCode() {
        return Status.OutOfStep;
    }
}
