package org.aion.kernel;

import i.RuntimeAssertionError;
import java.util.List;
import org.aion.avm.core.IExternalState;
import org.aion.types.InternalTransaction;
import org.aion.types.Log;
import org.aion.types.TransactionResult;
import org.aion.types.TransactionStatus;

/**
 * This class is used internally by the AVM to hold a {@link org.aion.types.TransactionResult} with
 * some additional data.
 *
 * It is strongly recommended that this class only ever be created via the {@link org.aion.avm.core.util.TransactionResultUtil} class!
 */
public final class AvmWrappedTransactionResult {
    private final TransactionResult result;
    public final AvmInternalError avmInternalError;
    public final Throwable exception;
    public final IExternalState externalState;

    public enum AvmInternalError {
        NONE                            ("", false, false),
        ABORTED                         ("Failed: aborted", true, false),
        FAILED_EXCEPTION                ("Failed: exception thrown", true, false),
        FAILED_UNEXPECTED               ("Failed: unexpected error", true, false),
        FAILED_OUT_OF_ENERGY            ("Failed: out of energy", true, false),
        FAILED_OUT_OF_STACK             ("Failed: out of stack", true, false),
        FAILED_CALL_DEPTH_LIMIT         ("Failed: call depth limit exceeded", true, false),
        FAILED_INVALID                  ("Failed: invalid", true, false),
        FAILED_INVALID_DATA             ("Failed: invalid data", true, false),
        FAILED_REJECTED_CLASS           ("Failed: rejected class", true, false),
        FAILED_REVERTED                 ("Failed: reverted", true, false),
        FAILED                          ("Failed", true, false),
        FAILED_RETRANSFORMATION         ("Failed: re-transformation failure", true, false),
        REJECTED_INVALID_VALUE          ("Rejected: invalid value", false, true),
        REJECTED_INVALID_ENERGY_PRICE   ("Rejected: invalid energy price", false, true),
        REJECTED_INVALID_ENERGY_LIMIT   ("Rejected: invalid energy limit", false, true),
        REJECTED_INVALID_NONCE          ("Rejected: invalid nonce", false, true),
        REJECTED_INSUFFICIENT_BALANCE   ("Rejected: insufficient balance", false, true);

        public final String error;
        private final boolean isFailed;
        private final boolean isRejected;
        AvmInternalError(String error, boolean isFailed, boolean isRejected) {
            this.error = error;
            this.isFailed = isFailed;
            this.isRejected = isRejected;
        }
    }

    /**
     * Constructs a new result wrapper that wraps the provided result and also contains additional
     * information such as the avm internal error that occurred, the external state, and any exception
     * that was thrown during execution.
     *
     * It is strongly recommended that this class only ever be created via the {@link org.aion.avm.core.util.TransactionResultUtil} class!
     *
     * Since this wrapped result contains additional error information in avmInternalError, this
     * constructor will throw an exception if the provided error information conflicts with the
     * result that is being wrapped. Only internally consistent wrappers can be created.
     *
     * @param result The result to wrap.
     * @param exception The exception.
     * @param externalState The external state.
     * @param avmInternalError The error.
     */
    public AvmWrappedTransactionResult(TransactionResult result, Throwable exception, IExternalState externalState, AvmInternalError avmInternalError) {
        if (result == null) {
            throw new NullPointerException("Cannot construct InternalTransactionResult with null result!");
        }
        if (avmInternalError == null) {
            throw new NullPointerException("Cannot construct InternalTransactionResult with null avmInternalError!");
        }

        // Ensure that the provided avmInternalError and result statuses are consistent.
        if (avmInternalError == AvmInternalError.NONE) {
            RuntimeAssertionError.assertTrue(result.transactionStatus.isSuccess());
        } else if (avmInternalError == AvmInternalError.ABORTED) {
            RuntimeAssertionError.assertTrue(result.transactionStatus.isFailed() && !result.transactionStatus.isReverted());
        } else if (avmInternalError == AvmInternalError.FAILED_EXCEPTION) {
            RuntimeAssertionError.assertTrue(result.transactionStatus.isFailed() && !result.transactionStatus.isReverted());
            RuntimeAssertionError.assertTrue(exception != null);
        } else if (avmInternalError == AvmInternalError.FAILED_UNEXPECTED) {
            RuntimeAssertionError.assertTrue(result.transactionStatus.isFailed() && !result.transactionStatus.isReverted());
            RuntimeAssertionError.assertTrue(exception != null);
        } else if (avmInternalError == AvmInternalError.FAILED_REVERTED) {
            RuntimeAssertionError.assertTrue(result.transactionStatus.isReverted());
        } else if (avmInternalError.isFailed) {
            RuntimeAssertionError.assertTrue(result.transactionStatus.isFailed() && !result.transactionStatus.isReverted());
        } else if (avmInternalError.isRejected) {
            RuntimeAssertionError.assertTrue(result.transactionStatus.isRejected());
        }

        this.result = result;
        this.exception = exception;
        this.externalState = externalState;
        this.avmInternalError = avmInternalError;
    }

    /**
     * Returns the transaction result that this wrapper class wraps.
     *
     * This method should only be called when handing a result off to the external world, since the
     * external world should not have to know anything about our wrapper.
     *
     * All avm internals should use the wrapper.
     *
     * @return the wrapped result.
     */
    public TransactionResult unwrap() {
        return this.result;
    }

    /**
     * @return the logs.
     */
    public List<Log> logs() {
        return this.result.logs;
    }

    /**
     * @return the internal transactions.
     */
    public List<InternalTransaction> internalTransactions() {
        return this.result.internalTransactions;
    }

    /**
     * Returns the output of the transaction or null if no output.
     *
     * @return transaction output or null.
     */
    public byte[] output() {
        return this.result.copyOfTransactionOutput().orElse(null);
    }

    /**
     * @return the transaction status.
     */
    public TransactionStatus transactionStatus() {
        return this.result.transactionStatus;
    }

    /**
     * @return the energy used.
     */
    public long energyUsed() {
        return this.result.energyUsed;
    }

    /**
     * Returns {@code true} only if this transaction result is aborted.
     *
     * @return whether or not it is aborted.
     */
    public boolean isAborted() {
        return this.avmInternalError == AvmInternalError.ABORTED;
    }

    /**
     * Returns {@code true} only if this transaction result is rejected.
     *
     * @return whether or not it is rejected.
     */
    public boolean isRejected() {
        return this.result.transactionStatus.isRejected();
    }

    /**
     * Returns {@code true} only if this transaction result is successful.
     *
     * @return whether or not it is successful.
     */
    public boolean isSuccess() {
        return this.result.transactionStatus.isSuccess();
    }

    /**
     * Returns {@code true} only if this transaction result is failed due to an unexpected error.
     *
     * @return whether or not it is failed due to an unexpected error.
     */
    public boolean isFailedUnexpected() {
        return this.avmInternalError == AvmInternalError.FAILED_UNEXPECTED;
    }

    /**
     * Returns {@code true} only if this transaction result is failed due to an uncaught exception.
     *
     * @return whether or not it is failed due to an uncaught exception.
     */
    public boolean isFailedException() {
        return this.avmInternalError == AvmInternalError.FAILED_EXCEPTION;
    }

    /**
     * Returns {@code true} only if this transaction result is reverted.
     *
     * @return whether or not it is reverted.
     */
    public boolean isRevert() {
        return this.result.transactionStatus.isReverted();
    }

    /**
     * Returns {@code true} only if this transaction result is failed.
     *
     * @return whether or not it is failed.
     */
    public boolean isFailed() {
        return this.result.transactionStatus.isFailed();
    }

    @Override
    public String toString() {
        return "AvmWrappedTransactionResult { result = " + this.result + ", internal error = " + this.avmInternalError + " }";
    }
}
