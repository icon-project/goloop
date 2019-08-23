package org.aion.avm.userlib.abi;


/**
 * General exception thrown when ABI classes are used incorrectly.
 *
 * A common example of where this is thrown is when an attempt is made to decode a specific type but the data extent describes a different type.
 */
public class ABIException extends RuntimeException {
    private static final long serialVersionUID = 1L;

    public ABIException(String message) {
        super(message);
    }

    // only used for jar optimization
    public ABIException() {
        super();
    }

}
