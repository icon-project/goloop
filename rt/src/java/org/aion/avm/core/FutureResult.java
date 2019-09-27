package org.aion.avm.core;


import org.aion.kernel.AvmWrappedTransactionResult;
import org.aion.types.TransactionResult;

/**
 * A simple alternative to {@link java.util.concurrent.Future} that provides a blocking
 * {@code getResult()} method, which blocks until a transaction result is ready to be consumed, a
 * blocking {@code getExternalState()} method, which blocks until the state changes of a transaction
 * are ready to be consumed, as well as a blocking {@code getException()} method, which blocks until
 * the transaction has been processed.
 *
 * All of these methods require that the transaction be finished processing, and once it is finished
 * all of these methods will no longer block.
 *
 * These methods are thread-safe.
 */
public final class FutureResult {
    private final HandoffMonitor handoffMonitor;
    private final int index;
    private AvmWrappedTransactionResult cachedResult;

    public FutureResult(HandoffMonitor handoffMonitor, int index) {
        this.handoffMonitor = handoffMonitor;
        this.index = index;
    }

    /**
     * Returns a transaction result, blocking if no result is ready to be consumed yet.
     *
     * @return a transaction result.
     */
    public TransactionResult getResult() {
        if (null == this.cachedResult) {
            this.cachedResult = this.handoffMonitor.blockingConsumeResult(this.index);
        }
        return this.cachedResult.unwrap();
    }

    /**
     * Returns the {@link IExternalState} object which holds all of the state changes caused by the
     * execution of the transaction, blocking if the state changes are not ready to be consumed yet.
     *
     * @return the post-execution state.
     */
    public IExternalState getExternalState() {
        if (null == this.cachedResult) {
            this.cachedResult = this.handoffMonitor.blockingConsumeResult(this.index);
        }
        return this.cachedResult.externalState;
    }

    /**
     * Returns an exception if one was thrown or else {@code null}, blocking if the transaction has
     * not finished executing yet.
     *
     * The returned exception must have been thrown either by a contract itself (and was not caught)
     * or else internally within the Avm. This information is exposed for logging purposes if the
     * caller wishes to have this information.
     *
     * @return an exception if one was thrown during execution or null.
     */
    public Throwable getException() {
        if (null == this.cachedResult) {
            this.cachedResult = this.handoffMonitor.blockingConsumeResult(this.index);
        }
        return this.cachedResult.exception;
    }
}
