package org.aion.parallel;

import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Transaction;
import i.IInstrumentation;
import i.RuntimeAssertionError;
import org.aion.avm.core.IExternalState;
import org.aion.avm.core.ReentrantDAppStack;

/**
 * A TransactionTask represent a complete transaction chain started from an external transaction.
 */
public class TransactionTask {
    private final IExternalState parentKernel;
    private Transaction externalTransaction;
    private IInstrumentation threadOwningTask;
    private ReentrantDAppStack reentrantDAppStack;
    private Address origin;
    private int depth;
    private int eid;
    private int prevEID;

    public TransactionTask(IExternalState parentKernel, Transaction tx, Address origin) {
        this.parentKernel = parentKernel;
        this.externalTransaction = tx;
        this.threadOwningTask = null;
        this.reentrantDAppStack = new ReentrantDAppStack();
        this.origin = origin;
        this.depth = 0;
    }

    public void startNewTransaction() {
        this.threadOwningTask = null;
        this.reentrantDAppStack = new ReentrantDAppStack();
    }

    /**
     * Attach an {@link IInstrumentation} to the current task.
     * If the task is already in abort state, set the helper abort state as well.
     */
    public void attachInstrumentationForThread() {
        RuntimeAssertionError.assertTrue(null == this.threadOwningTask);
        this.threadOwningTask = IInstrumentation.attachedThreadInstrumentation.get();
        RuntimeAssertionError.assertTrue(null != this.threadOwningTask);
    }

    public void detachInstrumentationForThread() {
        RuntimeAssertionError.assertTrue(IInstrumentation.attachedThreadInstrumentation.get() == this.threadOwningTask);
        this.threadOwningTask = null;
    }

    /**
     * Get the ReentrantDAppStack of the current task.
     *
     * @return The ReentrantDAppStack of the task.
     */
    public ReentrantDAppStack getReentrantDAppStack() {
        return reentrantDAppStack;
    }

    /**
     * Get the entry (external) transaction of the current task.
     *
     * @return The entry (external) transaction of the task.
     */
    public Transaction getTransaction() {
        return externalTransaction;
    }

    /**
     * Get the per task transactional kernel of the current task.
     *
     * @return The task transactional kernel of the task.
     */
    public IExternalState getThisTransactionalKernel() {
        return parentKernel;
    }

    public Address getOriginAddress() {
        return origin;
    }

    public int getTransactionStackDepth() {
        return depth;
    }

    public void incrementTransactionStackDepth() {
        depth++;
    }

    public void decrementTransactionStackDepth() {
        depth--;
    }

    public int getEID() {
        return eid;
    }

    public void setEID(int eid) {
        this.eid = eid;
    }

    public int getPrevEID() {
        return prevEID;
    }

    public void setPrevEID(int prevEID) {
        this.prevEID = prevEID;
    }
}
