package org.aion.avm.core;


import org.aion.avm.core.util.TransactionResultUtil;
import org.aion.kernel.AvmWrappedTransactionResult;

/**
 * This class handles exceptions that were thrown at some point during contract deployment
 * or a contract call. This class only handles those exceptions that DAppCreator or DAppExecutor
 * could not deal with.
 *
 * If we ever call into this class, it is because we have a bug in the AVM that we did not know about.
 * Correct behaviour for the AVM would mean that the throwable in question should not have been thrown,
 * or should have been handled in the DAppExecutor/DAppCreator.
 *
 * Previously, we handled these unexpected errors by treating them as fatal and shutting down the AVM.
 * However, we can contain the fallout, and keep the AVM alive, by treating any transaction that triggers
 * a fatal AVM shutdown as a "bad" transaction, and rejecting the transaction.
 *
 * In short, the AVM behaves as though it is infallible, and accuses the user of providing invalid data
 * if it ever enters an invalid state during transaction execution.
 */
public class DAppExceptionHandler {
    /**
     * Called to handle an exception thrown at some point during transaction execution.
     *
     * @param throwable The exception that we have been asked to handle.
     * @param result The AvmTransactionResult object that we will return for this transaction.
     */
    public static AvmWrappedTransactionResult handle(Throwable throwable, AvmWrappedTransactionResult result, long energyUsed, boolean verboseErrors) {
        // Anything else we couldn't handle more specifically needs to be passed further up to the top.
        if (verboseErrors) {
            System.err.println("Unknown error when executing this transaction: \"" + throwable.getMessage() + "\"");
            throwable.printStackTrace(System.err);
        }
        return TransactionResultUtil.setFailedUnexpected(result, throwable, energyUsed);
    }
}
