package org.aion.avm.core;


/**
 * Counters and timer data written by the owning AvmThread.
 * Mutable access to this structure is granted to all users since it is meant to be fast to access and has no internal
 * consistency requirements (although external consumers should take care to only make consistency assumptions when
 * the thread is not running).
 */
public class AvmThreadStats {
    public int transactionsProcessed;
    public long nanosRunning;
    public long nanosSleeping;

    public void clear() {
        this.transactionsProcessed = 0;
        this.nanosRunning = 0;
        this.nanosSleeping = 0;
    }
}
