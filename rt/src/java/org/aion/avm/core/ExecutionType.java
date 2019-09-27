package org.aion.avm.core;

/**
 * This is used in conjunction with commonMainchainBlockNumber to determine whether to read or write to the cache.
 * Note that even cases which don't update a cache may still invalidate it (eviction).
 */
public enum ExecutionType {
    /**
     * The case where the new block will be added to the current mainchain best block.
     * commonMainchainBlockNumber must be equal to currentBlockNumber - 1.
     * Both code and data cache will be read and written.
     */
    ASSUME_MAINCHAIN,
    /**
     * The case where the new block is on a sidechain, but its parent is a mainchain block.
     * commonMainchainBlockNumber must be the new block's immediate parent and it must be on the mainchain.
     * commonMainchainBlockNumber must be equal to currentBlockNumber - 1.
     * Any cache (code and data) up to and including commonMainchainBlockNumber is valid to use.
     * No caches are updated.
     */
    ASSUME_SIDECHAIN,
    /**
     * The case where the new block is on a sidechain, and its parent is on the sidechain as well.
     * commonMainchainBlockNumber must be zero and is not used because the exact fork point is not known.
     * No caches are read.
     * No caches are updated.
     */
    ASSUME_DEEP_SIDECHAIN,
    /**
     * The case where the main chain is switched/reorganized and a sidechain is marked as the new mainchain.
     * commonMainchainBlockNumber reflects the common ancestor of the new and old mainchain (fork point).
     * Only code caches up to and including commonMainchainBlockNumber are valid because the blocks between the common mainchain block and current block are not reflected in the cache.
     * The data cache is completely invalidated and not read.
     * Both caches are written.
     */
    SWITCHING_MAINCHAIN,
    /**
     * The case where mining operation is being performed.
     * commonMainchainBlockNumber must be equal to currentBlockNumber - 1.
     * Only the code cache is read.
     * No caches are updated.
     */
    MINING,
    /**
     * Used for calls that do not store the result including eth_call and eth_estimategas.
     * commonMainchainBlockNumber reflects the block and state where the request should be executed.
     * Both code and data caches up to and including commonMainchainBlockNumber will be read.
     * No caches are updated.
     */
    ETH_CALL,
}
