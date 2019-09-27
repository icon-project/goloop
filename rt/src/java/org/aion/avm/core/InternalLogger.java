package org.aion.avm.core;

import java.io.PrintStream;

/**
 * This class logs messages that are of interest to the AVM itself, but are not relevant
 * to someone sending transactions to the AVM for execution.
 */
public class InternalLogger {

    private final PrintStream output;

    public InternalLogger(PrintStream output) {
        this.output = output;
    }

    public void logFatal(Throwable throwable) {
        output.println("INTERNAL LOG: Unexpected error during transaction execution!");
        output.println(throwable.getMessage());
        throwable.printStackTrace(output);
    }
}
