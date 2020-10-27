/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package foundation.icon.ee.tooling.abi;

public class ABICompilerException extends RuntimeException {

    public ABICompilerException(String exceptionString) {
        super(exceptionString);
    }

    public ABICompilerException(String exceptionString, String methodName) {
        super("Exception in method \"" + methodName + "\": " + exceptionString);
    }
}
