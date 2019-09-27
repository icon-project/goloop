package org.aion.avm.core;

import i.RuntimeAssertionError;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.List;
import org.aion.avm.core.util.TransactionResultUtil;
import org.aion.kernel.AvmWrappedTransactionResult;
import org.aion.kernel.SideEffects;
import org.aion.parallel.TransactionTask;

import java.util.LinkedList;
import java.util.Queue;
import java.util.Set;
import org.aion.types.AionAddress;
import org.aion.types.InternalTransaction.RejectedStatus;


/**
 * Used by the AvmImpl to manage communication between its internal execution thread and the external calling thread.
 * This just provides monitor-protected blocking input/output variables, exception handling, and a safe way to shutdown.
 * Note that once an instance of this has been shutdown, it can't be started back up.
 * 
 * NOTE:  This currently assumes only one external thread is interacting with it at any given time.  This means that
 * attempting to send transactions from multiple threads or shutdown with one thread while running a transaction on another
 * would result in undefined behaviour.
 */
public class HandoffMonitor {
    private Set<Thread> internalThreads;
    private TransactionTask[] incomingTransactionTasks;

    private Queue<TransactionTask> taskQueue;

    private AvmWrappedTransactionResult[] outgoingResults;
    private Throwable backgroundThrowable;
    //private int nextTransactionIndex;

    public HandoffMonitor(Set<Thread> threadSet) {
        this.internalThreads = threadSet;
        this.taskQueue = new LinkedList<>();
    }

    /**
     * Called by the external thread.
     * Called to send new transactions to the internal thread.
     * 
     * @param tasks The tasks for each transaction to run.
     * @return The result of the transactions in the given tasks as a corresponding array of asynchronous futures.
     */
    public synchronized FutureResult[] sendTransactionsAsynchronously(TransactionTask[] tasks) {
        // We lock-step these, so there can't already be a transaction in the hand-off.
        RuntimeAssertionError.assertTrue(this.taskQueue.isEmpty());
        RuntimeAssertionError.assertTrue(null == this.outgoingResults);
        RuntimeAssertionError.assertTrue(tasks.length > 0);
        // Also, we can't have already been shut down.
        if (null == this.internalThreads) {
            throw new IllegalStateException("Thread already stopped");
        }

        this.incomingTransactionTasks = new TransactionTask[tasks.length];

        // Enqueue the new tasks and wake up the background thread.
        for (int i = 0; i < tasks.length; ++i ) {
            this.incomingTransactionTasks[i] = tasks[i];
            this.taskQueue.add(tasks[i]);
        }

        this.outgoingResults = new AvmWrappedTransactionResult[tasks.length];
        this.notifyAll();
        
        // Return the future result, which will do the waiting for us.
        FutureResult[] results = new FutureResult[tasks.length];
        for (int i = 0; i < results.length; ++i ) {
            results[i] = new FutureResult(this, i);
        }
        return results;
    }

    public synchronized AvmWrappedTransactionResult blockingConsumeResult(int index) {
        // Wait until we have the result or something went wrong.
        while ((null == this.outgoingResults[index]) && (null == this.backgroundThrowable)) {
            // It is an error to request a result while the avm is shut down.
            RuntimeAssertionError.assertTrue(this.internalThreads != null);

            // Otherwise, wait until state changes.
            try {
                this.wait();
            } catch (InterruptedException e) {
                // We don't use interruption.
                RuntimeAssertionError.unexpected(e);
            }
        }
        
        // Throw an exception, if there is one.
        handleThrowable();
        
        // Consume the result and return it.
        AvmWrappedTransactionResult result = this.outgoingResults[index];

        // Merge the logs and internal transactions from the side effects into the result.
        SideEffects sideEffects = incomingTransactionTasks[index].popSideEffects();
        result = TransactionResultUtil.addLogsAndInternalTransactions(result, sideEffects.getExecutionLogs(), sideEffects.getInternalTransactions());

        RuntimeAssertionError.assertTrue(incomingTransactionTasks[index].isSideEffectsStackEmpty());
        this.incomingTransactionTasks[index] = null;
        this.outgoingResults[index] = null;
        // If this is the last one in the list, drop it.
        // (note that this assumes the the results are consumed in-order - this requirement exists in more fundamental parts of the system, though).
        if ((index + 1) == this.outgoingResults.length) {
            this.incomingTransactionTasks = null;
            this.outgoingResults = null;
        }
        return result;
    }

    /**
     * Called by the internal thread.
     * The main blocking point for the internal thread.  It passes in the result from the last transaction it just completed
     * and then waits until a new transaction comes in or a shutdown is requested.
     * 
     * @param previousResult The result of the previous transaction returned by this call.
     * @return The next transaction to run or null if we should shut down.
     */
    public synchronized TransactionTask blockingPollForTransaction(
        AvmWrappedTransactionResult previousResult, TransactionTask previousTask) {
        // We may have been given these transactions as a list but we hand them out to the caller individually.
        
        // First, write-back any results that we have and notify anyone listening for that, on the front.
        if (null != previousResult) {
            this.outgoingResults[previousTask.getIndex()] = previousResult;
        }
        this.notifyAll();
        
        // This means that we only actually block when the incoming transactions are null (we make it null when we consume the last element and it becomes non-null when new data enqueued).
        while ((null != this.internalThreads) && (this.taskQueue.isEmpty())) {
            try {
                this.wait();
            } catch (InterruptedException e) {
                // We don't use interruption.
                RuntimeAssertionError.unexpected(e);
            }
        }
        
        // Unless this was a shutdown request, get the next transaction.
        TransactionTask nextTask = null;
        if (null != this.internalThreads) {
            // Make sure that we don't already have a response for the transaction we want to hand out.
            RuntimeAssertionError.assertTrue(null == this.outgoingResults[this.taskQueue.peek().getIndex()]);

            nextTask = this.taskQueue.poll();
        }
        return nextTask;
    }

    /**
     * Called by the internal thread.
     * This is called if something goes wrong while running the transaction on the internal thread to communicate this problem to the external.
     * 
     * @param throwable The exception (expected to be RuntimeException or Error).
     */
    public synchronized void setBackgroundThrowable(Throwable throwable) {
        // This will terminate anything the foreground is doing so notify them.
        this.backgroundThrowable = throwable;
        this.notifyAll();
    }

    /**
     * Called by the external thread.
     * Requests all the internal executor threads start.
     */
    public synchronized void startExecutorThreads(){
        for (Thread t: this.internalThreads){
            t.start();
        }
    }

    /**
     * Called by the external thread.
     * Requests that the internal thread stop.  Only returns once the internal thread has terminated.
     */
    public void stopAndWaitForShutdown() {
        // (called by the foreground thread)
        // Stop the thread and wait for it to join.
        Set<Thread> backgroundThreads = null;
        synchronized (this) {
            backgroundThreads = this.internalThreads;
            this.internalThreads = null;
            this.notifyAll();
        }
        
        // Join on the thread and throw any exceptions left over.
        // (note that we can't join under monitor since the thread needs the monitor to exit).
        try {
            for (Thread t : backgroundThreads){
                t.join();
            }
        } catch (InterruptedException e) {
            // We don't use interruption.
            RuntimeAssertionError.unexpected(e);
        }
        handleThrowable();
    }


    /**
     * Called by the external thread.
     */
    private void handleThrowable() {
        // WARNING:  This is not always called under monitor but this should be safe so long as backgroundThrowable saturates to non-null.
        if (null != this.backgroundThrowable) {
            // Only RuntimeExceptions and Errors can actually be handled here.
            try {
                throw this.backgroundThrowable;
            } catch (RuntimeException e) {
                throw e;
            } catch (Error e) {
                throw e;
            } catch (Throwable t) {
                // This can't happen since we only store those 2.
                RuntimeAssertionError.unexpected(t);
            }
        }
    }
}
