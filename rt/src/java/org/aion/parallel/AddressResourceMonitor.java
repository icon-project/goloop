package org.aion.parallel;

import i.RuntimeAssertionError;

import java.util.HashMap;
import java.util.HashSet;
import java.util.Set;

/**
 * Used by executor threads to communicate with each other.
 * Executor threads can only acquire/release {@link AddressResource}, commit result through this monitor.
 * A new monitor will be created for each batch of transactions.
 */
public class AddressResourceMonitor {
    static boolean DEBUG = false;

    // Map for resource retrieval
    private HashMap<AddressWrapper, AddressResource> resources;

    // Ownership records for each task. It provide fast resource release.
    private HashMap<TransactionTask, Set<AddressResource>> ownerships;

    // Private monitor for safety
    private final Object sync;

    // Commit counter used to serialize transaction commit
    private int commitCounter;

    public AddressResourceMonitor()
    {
        this.resources = new HashMap<>();
        this.ownerships = new HashMap<>();
        this.sync = new Object();
        this.commitCounter = 0;
    }

    /**
     * Reset the state of the address resource monitor.
     * This method will be called for each batch of transaction request.
     *
     */
    public void clear(){
        synchronized (sync) {
            this.resources.clear();
            this.ownerships.clear();
            this.commitCounter = 0;
        }
    }

    /**
     * Acquire a resource for given task.
     * Called by executor thread when access of a address is needed.
     *
     * This method block when another executor thread is holding the resource
     *
     * This method return when
     *      The resource is sucessfully acquired (This method support reentrant)
     *      OR
     *      The requester thread need to abort
     *
     * @param address The address requested.
     * @param task The requester task.
     * @return true if the address was acquired by the task, false otherwise
     */
    public boolean acquire(byte[] address, TransactionTask task){
        synchronized (sync) {
            AddressWrapper addressWrapper = new AddressWrapper(address);
            AddressResource resource = getResource(addressWrapper);

            // Add task to the waiting queue.
            if (resource.addToWaitingQueue(task)) {
                sync.notifyAll();
            }

            long startTime = 0;
            long endTime = 0;

            if (DEBUG) {
                int holder = null != resource.getOwnedBy() ? resource.getOwnedBy().getIndex() : -1;
                int nextOwner = null != resource.getNextOwner() ? resource.getNextOwner().getIndex() : -1;
                System.out.println("Request " + task.getIndex() + " " + resource.toString() + " hold by " + holder +
                        " nextOwner " + nextOwner + " locked " + resource.isOwned() + " inAbortState " + task.inAbortState());
                startTime = System.nanoTime();
            }

            // Resource res is granted to task iff
            // res is not hold by other task && task is the next owner
            while ((resource.isOwned() || !resource.isNextOwner(task)) && task != resource.getOwnedBy()
                    && !task.inAbortState()){
                try {
                    sync.wait();
                }catch (InterruptedException e){
                    RuntimeAssertionError.unreachable("Waiting executor thread received interruption: ACQUIRE");
                }
            }

            boolean isAborted = task.inAbortState();
            if (!isAborted) {
                if (DEBUG) {
                    endTime = System.nanoTime();
                    System.out.println("Acquire " + task.getIndex() + " " + resource.toString()
                            + " waitingTime " + (endTime - startTime)/1000 + " \u00B5s");
                }
                resource.setOwner(task);
                recordOwnership(resource, task);
            }else{
                if (DEBUG) {
                    endTime = System.nanoTime();
                    System.out.println("Abort   " + task.getIndex() + " " + resource.toString()
                            + " waitingTime " + (endTime - startTime)/1000 + " \u00B5s");
                }
            }

            if (DEBUG) System.out.flush();
            return !isAborted;
        }
    }

    /**
     * Release all resource holding by given task.
     * Called by executor thread when the task finished/need restart.
     *
     * This method will not block.
     *
     * @param task The requesting task.
     */
    private void releaseResourcesForTask(TransactionTask task){
        RuntimeAssertionError.assertTrue(Thread.holdsLock(sync));

        Set<AddressResource> toRemove = ownerships.remove(task);
        if (null != toRemove) {
            for (AddressResource resource : toRemove) {
                resource.removeFromWaitingQueue(task);
                if (task == resource.getOwnedBy()) {
                    resource.setOwner(null);
                }
                if (DEBUG) {
                    int nextOwner = null != resource.getNextOwner() ? resource.getNextOwner().getIndex() : -1;
                    System.out.println("Release " + task.getIndex() + " " + resource.toString() + " nextOwner " + nextOwner);
                }
            }
        }
    }

    /**
     * Try commit the task transactional kernel of the given task.
     * The commit will be serialized as the index of the task.
     * All resource hold by task will be released after this method return.
     *
     * The executor thread of the task will block until
     *      It is the task's turn to commit result
     *      OR
     *      The task need to abort to yield a address resource
     *
     * @param task The requesting task.
     * @param isRejected True only if the transaction relating to this task was rejected.
     *
     * @return True if commit is successful. False if task need to abort.
     */
    public boolean commitKernelForTask(TransactionTask task, boolean isRejected){
        boolean ret = false;

        synchronized (sync){

            while (this.commitCounter != task.getIndex() && !task.inAbortState()){
                try {
                    sync.wait();
                }catch (InterruptedException e){
                    RuntimeAssertionError.unreachable("Waiting executor thread received interruption: COMMIT");
                }
            }

            if (!task.inAbortState()){
                if (!isRejected) {
                    task.getThisTransactionalKernel().commit();
                    task.outputFlush();
                }
                this.commitCounter++;
                ret = true;
            }

            releaseResourcesForTask(task);

            // Only wake up others when all resources are released
            sync.notifyAll();
        }

        return ret;
    }

    private AddressResource getResource(AddressWrapper addr){
        RuntimeAssertionError.assertTrue(Thread.holdsLock(sync));

        AddressResource ret = resources.get(addr);
        if (null == ret){
            ret = new AddressResource();
            resources.put(addr, ret);
        }
        return ret;
    }

    private void recordOwnership(AddressResource res, TransactionTask task){
        RuntimeAssertionError.assertTrue(Thread.holdsLock(sync));

        Set<AddressResource> entry = ownerships.get(task);
        if (null == entry){
            entry = new HashSet<>();
            ownerships.put(task, entry);
        }
        entry.add(res);
    }

    void testReleaseResourcesForTask(TransactionTask task){
        synchronized (sync) {
            releaseResourcesForTask(task);
            sync.notifyAll();
        }
    }
}

