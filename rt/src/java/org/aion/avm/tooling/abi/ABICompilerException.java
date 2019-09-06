package org.aion.avm.tooling.abi;

public class ABICompilerException extends RuntimeException {

    public ABICompilerException(String exceptionString) {
        super(exceptionString);
    }

    public ABICompilerException(String exceptionString, String methodName) {
        super("Exception in method " + methodName + ": " + exceptionString);
    }
}
