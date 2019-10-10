package org.aion.avm.core;

import i.AvmError;

public class InvalidDBAccessException extends AvmError {
    InvalidDBAccessException() {
    }

    InvalidDBAccessException(String msg) {
        super(msg);
    }
}
