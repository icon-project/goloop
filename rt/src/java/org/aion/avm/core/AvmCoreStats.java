package org.aion.avm.core;


/**
 * Counters and timer data related to core AVM activities, more reflective of high-level usage than per-thread activities.
 * Mutable access to this structure is granted to all users since it is meant to be fast to access and has no internal
 * consistency requirements (although external consumers should take care to only make consistency assumptions when no
 * thread is in the AVM).
 */
public class AvmCoreStats {
    public final AvmThreadStats[] threadStats;
    public int transactionsConsumed;
    public int batchesConsumed;

    public AvmCoreStats(AvmThreadStats[] threadStats) {
        this.threadStats = threadStats;
    }

    public void clear() {
        for (AvmThreadStats stat : this.threadStats) {
            stat.clear();
        }
        this.transactionsConsumed = 0;
        this.batchesConsumed = 0;
    }
}
