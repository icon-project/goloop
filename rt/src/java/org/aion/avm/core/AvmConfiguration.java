package org.aion.avm.core;


/**
 * A class to describe how to configure an AVM instance, when requesting that it be created.
 * The overall strategy with this class is just to make it into a plain-old-data struct, with each field set to a "default" value
 * which can be overwritten by a caller.
 */
public class AvmConfiguration {
    /**
     * The number of threads to start for running the incoming transactions.
     * A lower number will reduce maximum throughput but a higher number will increase the number of aborts experienced as a result
     * of data hazards.  The transaction restarts caused by these aborts may reduce throughput on such highly connected blocks.
     */
    public int threadCount;
    /**
     * Decides if debug data and names need to be preserved during deployment transformation.
     * Note that this must be set to false as a requirement of the security model but that prohibits local debugging.  Hence, it
     * should only be enabled for embedded use-cases during development, never in actual deployment in a node.
     * Some security violations will change into fatal assertion errors, instead of being rejected, if this is enabled.
     */
    public boolean preserveDebuggability;
    /**
     * If set to true, will log details of uncaught contract exceptions to stderr.
     * Enabling this is useful for local debugging cases.
     */
    public boolean enableVerboseContractErrors;
    /**
     * If set to true, will log more information about the state of the concurrent executor.
     * Enabling this is only really useful when actively modifying the concurrent executor.
     */
    public boolean enableVerboseConcurrentExecutor;
    /**
     * If set to true, will pass calls to Blockchain.println to the underlying stdout console.
     * If false, this call is still legal but will have no effect.
     */
    public boolean enableBlockchainPrintln;

    public AvmConfiguration() {
        // 4 threads is generally a safe, yet useful, number.
        this.threadCount = 4;
        // By default, we MUST reparent user code and discard debug data!  This is part of the security model so it should only be enabled to enable local contract debugging.
        this.preserveDebuggability = false;
        // By default, none of our verbose options are enabled.
        this.enableVerboseContractErrors = false;
        this.enableVerboseConcurrentExecutor = false;
        // While the system is still relatively new, we enable the Blockchain.println output, by default.
        this.enableBlockchainPrintln = true;
    }
}
