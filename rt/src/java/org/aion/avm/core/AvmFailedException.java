package org.aion.avm.core;


/**
 * Thrown by the AVM instance (or anything related to it) in the case of a fatal background thread error.
 * Generally, the entire process needs to be brought down if this is ever observed.
 */
public class AvmFailedException extends RuntimeException {
    private static final long serialVersionUID = 1L;

    public AvmFailedException(Throwable cause) {
        super(cause);
    }
}
