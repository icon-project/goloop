package org.aion.avm.core;

import org.aion.avm.core.util.TransactionResultUtil;
import org.aion.kernel.AvmWrappedTransactionResult.AvmInternalError;
import org.aion.types.AionAddress;
import org.aion.types.Transaction;
import org.aion.kernel.*;

import java.io.IOException;
import java.lang.ref.SoftReference;
import java.math.BigInteger;
import java.util.HashSet;
import java.util.Set;
import java.util.function.Consumer;
import java.util.function.Predicate;

import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.avm.core.util.ByteArrayWrapper;
import org.aion.avm.core.util.SoftCache;
import i.IInstrumentation;
import i.IInstrumentationFactory;
import i.InstrumentationHelpers;
import i.JvmError;
import i.RuntimeAssertionError;
import org.aion.parallel.AddressResourceMonitor;
import org.aion.parallel.TransactionTask;


public class AvmImpl implements AvmInternal {
    private InternalLogger internalLogger;

    private final IInstrumentationFactory instrumentationFactory;
    private final IExternalCapabilities capabilities;

    // Long-lived state which is book-ended by the startup/shutdown calls.
    private static AvmImpl currentAvm;  // (only here for testing - makes sure that we properly clean these up between invocations)
    private SoftCache<ByteArrayWrapper, LoadedDApp> hotCache;
    private HandoffMonitor handoff;

    // Short-lived state which is reset for each batch of transaction request.
    private AddressResourceMonitor resourceMonitor;

    // Shared references to the stats structure - created when threads are started (since their stats are also held here).
    private AvmCoreStats stats;

    // Used in the case of a fatal JvmError in the background threads.  A shutdown() is the only option from this point.
    private AvmFailedException backgroundFatalError;

    private final int threadCount;
    private final boolean preserveDebuggability;
    private final boolean enableVerboseContractErrors;
    private final boolean enableVerboseConcurrentExecutor;
    private final boolean enableBlockchainPrintln;

    public AvmImpl(IInstrumentationFactory instrumentationFactory, IExternalCapabilities capabilities, AvmConfiguration configuration) {
        this.instrumentationFactory = instrumentationFactory;
        this.capabilities = capabilities;
        // Make sure that the threadCount isn't totally invalid.
        if (configuration.threadCount < 1) {
            throw new IllegalArgumentException("Thread count must be a positive integer");
        }
        this.threadCount = configuration.threadCount;
        this.preserveDebuggability = configuration.preserveDebuggability;
        this.enableVerboseContractErrors = configuration.enableVerboseContractErrors;
        this.enableVerboseConcurrentExecutor = configuration.enableVerboseConcurrentExecutor;
        this.enableBlockchainPrintln = configuration.enableBlockchainPrintln;
        this.internalLogger = new InternalLogger(System.err);
    }

    private class AvmExecutorThread extends Thread{
        public final AvmThreadStats stats = new AvmThreadStats();

        AvmExecutorThread(String name){
            super(name);
        }

        @Override
        public void run() {
            IInstrumentation instrumentation = AvmImpl.this.instrumentationFactory.createInstrumentation();
            InstrumentationHelpers.attachThread(instrumentation);
            try {
                // Run as long as we have something to do (null means shutdown).
                AvmWrappedTransactionResult outgoingResult = null;
                long nanosRunningStop = System.nanoTime();
                long nanosSleepingStart = nanosRunningStop;
                TransactionTask incomingTask = AvmImpl.this.handoff.blockingPollForTransaction(null, null);
                long nanosRunningStart = System.nanoTime();
                long nanosSleepingStop = nanosRunningStart;
                this.stats.nanosSleeping += (nanosSleepingStop - nanosSleepingStart);

                while (null != incomingTask) {
                    int abortCounter = 0;

                    do {
                        if (AvmImpl.this.enableVerboseConcurrentExecutor) {
                            System.out.println(this.getName() + " start  " + incomingTask.getIndex());
                        }

                        // Attach the IInstrumentation helper to the task to support asynchronous abort
                        // Instrumentation helper will abort the execution of the transaction by throwing an exception during chargeEnergy call
                        // Aborted transaction will be retried later
                        incomingTask.startNewTransaction();
                        incomingTask.attachInstrumentationForThread();
                        outgoingResult = AvmImpl.this.backgroundProcessTransaction(incomingTask);
                        incomingTask.detachInstrumentationForThread();

                        if (outgoingResult.isAborted()) {
                            // If this was an abort, we want to clear the abort state on the instrumentation for this thread, since
                            // this is the point where that is "handled".
                            // Note that this is safe to do here since the instrumentation isn't exposed to any other threads.
                            instrumentation.clearAbortState();
                            
                            if (AvmImpl.this.enableVerboseConcurrentExecutor) {
                                System.out.println(this.getName() + " abort  " + incomingTask.getIndex() + " counter " + (++abortCounter));
                            }
                        }
                    }while (outgoingResult.isAborted());

                    if (AvmImpl.this.enableVerboseConcurrentExecutor) {
                        System.out.println(this.getName() + " finish " + incomingTask.getIndex() + " " + outgoingResult);
                    }

                    this.stats.transactionsProcessed += 1;
                    nanosRunningStop = System.nanoTime();
                    nanosSleepingStart = nanosRunningStop;
                    this.stats.nanosRunning += (nanosRunningStop - nanosRunningStart);
                    incomingTask = AvmImpl.this.handoff.blockingPollForTransaction(outgoingResult, incomingTask);
                    nanosRunningStart = System.nanoTime();
                    nanosSleepingStop = nanosRunningStart;
                    this.stats.nanosSleeping += (nanosSleepingStop - nanosSleepingStart);
                }
            } catch (JvmError e) {
                // This is a fatal error the AVM cannot generally happen so request an asynchronous shutdown.
                // We set the backgroundException without lock since any concurrently-written exception instance is equally valid.
                AvmFailedException backgroundFatalError = new AvmFailedException(e.getCause());
                AvmImpl.this.backgroundFatalError = backgroundFatalError;
                AvmImpl.this.handoff.setBackgroundThrowable(backgroundFatalError);
            } catch (Throwable t) {
                // Note that this case is primarily only relevant for unit tests or other new development which could cause internal exceptions.
                // Without this hand-off to the foreground thread, these exceptions would cause silent failures.
                // Uncaught exception - this is fatal but we need to communicate it to the outside.
                AvmImpl.this.handoff.setBackgroundThrowable(t);
            } finally {
                InstrumentationHelpers.detachThread(instrumentation);
                AvmImpl.this.instrumentationFactory.destroyInstrumentation(instrumentation);
            }
        }

    }

    public void start() {
        // An AVM instance can only be started once so we shouldn't yet have stats.
        RuntimeAssertionError.assertTrue(null == this.stats);
        
        // There are currently no consumers which have more than 1 AVM instance running concurrently so we enforce this in order to flag static errors.
        RuntimeAssertionError.assertTrue(null == AvmImpl.currentAvm);
        AvmImpl.currentAvm = this;
        
        RuntimeAssertionError.assertTrue(null == this.hotCache);
        this.hotCache = new SoftCache<>();

        RuntimeAssertionError.assertTrue(null == this.resourceMonitor);
        this.resourceMonitor = new AddressResourceMonitor();

        AvmThreadStats[] threadStats = new AvmThreadStats[this.threadCount];
        Set<Thread> executorThreads = new HashSet<>();
        for (int i = 0; i < this.threadCount; i++){
            AvmExecutorThread thread = new AvmExecutorThread("AVM Executor Thread " + i);
            executorThreads.add(thread);
            threadStats[i] = thread.stats;
        }
        this.stats = new AvmCoreStats(threadStats);

        RuntimeAssertionError.assertTrue(null == this.handoff);
        this.handoff = new HandoffMonitor(executorThreads);
        this.handoff.startExecutorThreads();
    }

    public FutureResult[] run(IExternalState kernel, Transaction[] transactions, ExecutionType executionType, long commonMainchainBlockNumber) throws IllegalStateException {
        long currentBlockNum = kernel.getBlockNumber();

        if (transactions.length <= 0) {
            throw new IllegalArgumentException("Number of transactions must be larger than 0");
        }

        // validate commonMainchainBlockNumber based on execution type
        if (executionType == ExecutionType.ASSUME_MAINCHAIN || executionType == ExecutionType.ASSUME_SIDECHAIN || executionType == ExecutionType.MINING) {
            // This check generally true for mining but it's added for rare cases of mining on top of an imported block which is not the latest
            if (currentBlockNum != commonMainchainBlockNumber + 1) {
                throw new IllegalArgumentException("Invalid commonMainchainBlockNumber for " + executionType + " currentBlock = " + currentBlockNum + " , commonMainchainBlockNumber = " + commonMainchainBlockNumber);
            }
        } else if (executionType == ExecutionType.ASSUME_DEEP_SIDECHAIN && commonMainchainBlockNumber != 0) {
            throw new IllegalArgumentException("commonMainchainBlockNumber must be zero for " + executionType);
        }

        // validate cache based on execution type
        if (executionType == ExecutionType.ASSUME_MAINCHAIN || executionType == ExecutionType.MINING) {
            validateCodeCache(currentBlockNum);
        } else if (executionType == ExecutionType.SWITCHING_MAINCHAIN) {
            // commonMainchainBlockNumber is the last valid block so anything after that should be removed from the cache
            validateCodeCache(commonMainchainBlockNumber + 1);
            purgeDataCache();
        }

        if (null != this.backgroundFatalError) {
            throw this.backgroundFatalError;
        }
        // Clear the states of resources
        this.resourceMonitor.clear();

        // Create tasks for these new transactions and send them off to be asynchronously executed.
        TransactionTask[] tasks = new TransactionTask[transactions.length];
        for (int i = 0; i < transactions.length; i++){
            tasks[i] = new TransactionTask(kernel, transactions[i], i, transactions[i].senderAddress, executionType, commonMainchainBlockNumber);
        }

        this.stats.batchesConsumed += 1;
        this.stats.transactionsConsumed += transactions.length;
        return this.handoff.sendTransactionsAsynchronously(tasks);
    }

    public AvmCoreStats getStats() {
        return this.stats;
    }

    private AvmWrappedTransactionResult backgroundProcessTransaction(TransactionTask task) {
        // to capture any error during validation
        AvmInternalError error = AvmInternalError.NONE;

        RuntimeAssertionError.assertTrue(task != null);
        Transaction tx = task.getTransaction();
        RuntimeAssertionError.assertTrue(tx != null);

        // value/energyPrice/energyLimit sanity check
        BigInteger value = tx.value;
        if (value.compareTo(BigInteger.ZERO) < 0) {
            error = AvmInternalError.REJECTED_INVALID_VALUE;
        }
        if (tx.energyPrice <= 0) {
            error = AvmInternalError.REJECTED_INVALID_ENERGY_PRICE;
        }
        
        if (tx.isCreate) {
            if (!task.getThisTransactionalKernel().isValidEnergyLimitForCreate(tx.energyLimit)) {
                error = AvmInternalError.REJECTED_INVALID_ENERGY_LIMIT;
            }
        } else {
            if (!task.getThisTransactionalKernel().isValidEnergyLimitForNonCreate(tx.energyLimit)) {
                error = AvmInternalError.REJECTED_INVALID_ENERGY_LIMIT;
            }
        }

        // Acquire both sender and target resources
        AionAddress sender = tx.senderAddress;
        AionAddress target = (tx.isCreate) ? capabilities.generateContractAddress(tx) : tx.destinationAddress;

        AvmWrappedTransactionResult result = null;

        boolean isSenderAcquired = this.resourceMonitor.acquire(sender.toByteArray(), task);
        boolean isTargetAcquired = this.resourceMonitor.acquire(target.toByteArray(), task);

        if (isSenderAcquired && isTargetAcquired) {
            // nonce check
            if (!task.getThisTransactionalKernel().accountNonceEquals(sender, tx.nonce)) {
                error = AvmInternalError.REJECTED_INVALID_NONCE;
            }

            if (AvmInternalError.NONE == error) {
                // The CREATE/CALL case is handled via the common external invoke path.
                result = runExternalInvoke(task.getThisTransactionalKernel(), task, tx);
            } else {
                result = TransactionResultUtil.newRejectedResultWithEnergyUsed(error, tx.energyLimit);
            }
        } else {
            result = TransactionResultUtil.newAbortedResultWithZeroEnergyUsed();
        }

        // Task transactional kernel commits are serialized through address resource monitor
        // This should be done for all transaction result cases, including FAILED_ABORT, because one of the addresses might have been acquired
        if (!this.resourceMonitor.commitKernelForTask(task, result.isRejected())) {
            // A transaction task can be aborted even after it has finished.
            result = TransactionResultUtil.newAbortedResultWithZeroEnergyUsed();
        }

        if (!result.isAborted()){
            result = TransactionResultUtil.setExternalState(result, task.getThisTransactionalKernel());
        }

        return result;
    }

    public void shutdown() {
        // Note that we can fail due to either a RuntimeException or an Error, so catch either and be explicit about re-throwing.
        Error errorDuringShutdown = null;
        RuntimeException exceptionDuringShutdown = null;
        try {
            this.handoff.stopAndWaitForShutdown();
        } catch (RuntimeException e) {
            // Note that this is usually the same instance as backgroundFatalError can fail for other reasons.  Catch this, complete
            // the shutdown, then re-throw it.
            exceptionDuringShutdown = e;
        } catch (Error e) {
            // Same thing for Error.
            errorDuringShutdown = e;
        }
        this.handoff = null;
        RuntimeAssertionError.assertTrue(this == AvmImpl.currentAvm);
        AvmImpl.currentAvm = null;
        this.hotCache = null;
        
        // Note that we don't want to hide the background exception, if one happened, but we do want to complete the shutdown, so we do this at the end.
        if (null != errorDuringShutdown) {
            throw errorDuringShutdown;
        }
        if (null != exceptionDuringShutdown) {
            throw exceptionDuringShutdown;
        }
        if (null != this.backgroundFatalError) {
            throw this.backgroundFatalError;
        }
    }

    @Override
    public AvmWrappedTransactionResult runInternalTransaction(IExternalState parentKernel, TransactionTask task, Transaction tx) {
        if (null != this.backgroundFatalError) {
            throw this.backgroundFatalError;
        }
        RuntimeAssertionError.assertTrue(!task.isSideEffectsStackEmpty());
        task.pushSideEffects(new SideEffects());
        AvmWrappedTransactionResult result = commonInvoke(parentKernel, task, tx, 0);
        SideEffects txSideEffects = task.popSideEffects();
        if (!result.isSuccess()) {
            txSideEffects.getExecutionLogs().clear();
            // unsuccessful transaction result can either be due to an error or an abort case. In abort case the rejection status will be overridden.
            txSideEffects.markAllInternalTransactionsAsRejected();
        }
        task.peekSideEffects().merge(txSideEffects);
        return result;
    }

    private AvmWrappedTransactionResult runExternalInvoke(IExternalState parentKernel, TransactionTask task, Transaction tx) {
        // to capture any error during validation
        AvmInternalError error = AvmInternalError.NONE;

        // Sanity checks around energy pricing and nonce are done in the caller.
        // balance check
        AionAddress sender = tx.senderAddress;
        long energyPrice = tx.energyPrice;
        BigInteger value = tx.value;

        long basicTransactionCost = BillingRules.getBasicTransactionCost(tx.copyOfTransactionData());
        BigInteger balanceRequired = BigInteger.valueOf(tx.energyLimit).multiply(BigInteger.valueOf(energyPrice)).add(value);

        if (basicTransactionCost > tx.energyLimit) {
            error = AvmInternalError.REJECTED_INVALID_ENERGY_LIMIT;
        }
        else if (!parentKernel.accountBalanceIsAtLeast(sender, balanceRequired)) {
            error = AvmInternalError.REJECTED_INSUFFICIENT_BALANCE;
        }

        // exit if validation check fails
        if (error != AvmInternalError.NONE) {
            return TransactionResultUtil.newRejectedResultWithEnergyUsed(error, tx.energyLimit);
        }

        /*
         * After this point, no rejection should occur.
         */

        // Deduct the total energy cost
        parentKernel.adjustBalance(sender, BigInteger.valueOf(tx.energyLimit).multiply(BigInteger.valueOf(energyPrice).negate()));

        // Run the common logic with the parent kernel as the top-level one.
        AvmWrappedTransactionResult result = commonInvoke(parentKernel, task, tx, basicTransactionCost);

        // Refund energy for transaction
        BigInteger refund = BigInteger.valueOf(tx.energyLimit - result.energyUsed()).multiply(BigInteger.valueOf(energyPrice));
        parentKernel.refundAccount(sender, refund);

        // Transfer fees to miner
        parentKernel.adjustBalance(parentKernel.getMinerAddress(), BigInteger.valueOf(result.energyUsed()).multiply(BigInteger.valueOf(energyPrice)));

        if (!result.isSuccess()) {
            task.peekSideEffects().getExecutionLogs().clear();
            // unsuccessful transaction result can either be due to an error or an abort case. In abort case the rejection status will be overridden.
            task.peekSideEffects().markAllInternalTransactionsAsRejected();
        }

        return result;
    }

    private AvmWrappedTransactionResult commonInvoke(IExternalState parentKernel, TransactionTask task, Transaction tx, long transactionBaseCost) {
        // Invoke calls must build their transaction on top of an existing "parent" kernel.
        TransactionalState thisTransactionKernel = new TransactionalState(parentKernel);

        AvmWrappedTransactionResult result = TransactionResultUtil.newSuccessfulResultWithEnergyUsed(transactionBaseCost);

        // grab the recipient address as either the new contract address or the given account address.
        AionAddress recipient = (tx.isCreate) ? capabilities.generateContractAddress(tx) : tx.destinationAddress;

        // conduct value transfer
        BigInteger value = tx.value;
        thisTransactionKernel.adjustBalance(tx.senderAddress, value.negate());
        thisTransactionKernel.adjustBalance(recipient, value);

        // At this stage, transaction can no longer be rejected.
        // The nonce increment will be done regardless of the transaction result.
        task.getThisTransactionalKernel().incrementNonce(tx.senderAddress);

        // do nothing for balance transfers of which the recipient is not a DApp address.
        if (tx.isCreate) {
            result = DAppCreator.create(this.capabilities, thisTransactionKernel, this, task, tx, result, this.preserveDebuggability, this.enableVerboseContractErrors, this.enableBlockchainPrintln);
        } else { // call
            // See if this call is trying to reenter one already on this call-stack.  If so, we will need to partially resume its state.
            ReentrantDAppStack.ReentrantState stateToResume = task.getReentrantDAppStack().tryShareState(recipient);

            LoadedDApp dapp = null;
            byte[] transformedCode = thisTransactionKernel.getTransformedCode(recipient);
            // The reentrant cache is obviously the first priority.
            // (note that we also want to check the kernel we were given to make sure that this DApp hasn't been deleted since we put it in the cache.
            if ((null != stateToResume) && (null != transformedCode)) {
                dapp = stateToResume.dApp;
                // Call directly and don't interact with DApp cache (we are reentering the state, not the origin of it).
                result = DAppExecutor.call(this.capabilities, thisTransactionKernel, this, dapp, stateToResume, task, tx, result, this.enableVerboseContractErrors, true, this.enableBlockchainPrintln);
            } else {
                long currentBlockNumber = parentKernel.getBlockNumber();

                // If we didn't find it there (that is only for reentrant calls so it is rarely found in the stack), try the hot DApp cache.
                ByteArrayWrapper addressWrapper = new ByteArrayWrapper(recipient.toByteArray());
                LoadedDApp dappInHotCache = null;

                // This reflects if the dapp should be checked back into either cache after transaction execution
                boolean writeToCacheEnabled;

                // This reflects if the DApp data can be loaded from the cache.
                // It is used in DAppExecutor to determine whether to load the data by making a call to the database or from DApp object
                boolean readFromDataCacheEnabled;

                boolean updateDataCache;

                // there are no interactions with either cache in ASSUME_DEEP_SIDECHAIN
                if (task.executionType != ExecutionType.ASSUME_DEEP_SIDECHAIN) {
                    dappInHotCache = this.hotCache.checkout(addressWrapper);
                }

                if (task.executionType == ExecutionType.ASSUME_MAINCHAIN || task.executionType == ExecutionType.SWITCHING_MAINCHAIN) {
                    // cache has been validated for these two types before getting here
                    writeToCacheEnabled = true;
                    readFromDataCacheEnabled = dappInHotCache != null && dappInHotCache.hasValidCachedData(currentBlockNumber);
                    updateDataCache = true;
                } else if (task.executionType == ExecutionType.ASSUME_SIDECHAIN || task.executionType == ExecutionType.ETH_CALL) {
                    if (dappInHotCache != null) {
                        // Check if the code is valid at this height. The last valid block for code cache is the CommonMainchainBlockNumber
                        if (!dappInHotCache.hasValidCachedCode(task.commonMainchainBlockNumber + 1)) {
                            // if we cannot use the cache, put the dapp back and work with the database
                            this.hotCache.checkin(addressWrapper, dappInHotCache);
                            writeToCacheEnabled = false;
                            dappInHotCache = null;
                        } else {
                            // only dapp code is written back to the cache.
                            // this is enabled only if the dapp has valid code cache at current height
                            writeToCacheEnabled = true;
                        }
                    } else {
                        // if the dapp could not be found in the cache, do not write to the cache
                        writeToCacheEnabled = false;
                    }
                    readFromDataCacheEnabled = dappInHotCache != null && dappInHotCache.hasValidCachedData(task.commonMainchainBlockNumber + 1);
                    updateDataCache = false;
                } else if (task.executionType == ExecutionType.ASSUME_DEEP_SIDECHAIN) {
                    writeToCacheEnabled = false;
                    readFromDataCacheEnabled = false;
                    updateDataCache = false;
                } else {
                    // ExecutionType.MINING
                    // if the dapp was already present in the cache, its code is written back to the cache.
                    writeToCacheEnabled = dappInHotCache != null;
                    readFromDataCacheEnabled = false;
                    updateDataCache = false;
                }

                // lazily re-transform the code
                if (transformedCode == null) {
                    byte[] code = parentKernel.getCode(recipient);
                    //'parentKernel.getCode(recipient) != null' means this recipient's DApp is not self-destructed.
                    if (code != null) {
                        transformedCode = CodeReTransformer.transformCode(code, parentKernel.getBlockTimestamp(), this.preserveDebuggability, this.enableVerboseContractErrors);
                        if (transformedCode == null) {
                            // re-transformation of the code failed
                            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED_RETRANSFORMATION, tx.energyLimit);
                        } else {
                            parentKernel.setTransformedCode(recipient, transformedCode);
                        }
                    }
                } else {
                // do not use the cache if the code has not been transformed for the latest version
                    dapp = dappInHotCache;
                }
                if (null == dapp) {
                    // If we didn't find it there, just load it.
                    try {
                        dapp = DAppLoader.loadFromGraph(transformedCode, this.preserveDebuggability);

                        // If the dapp is freshly loaded, we set the block num
                        if (null != dapp){
                            dapp.setLoadedCodeBlockNum(currentBlockNumber);
                        }

                    } catch (IOException e) {
                        throw RuntimeAssertionError.unexpected(e); // the jar was created by AVM; IOException is unexpected
                    }
                }

                if (null != dapp) {
                    result = DAppExecutor.call(this.capabilities, thisTransactionKernel, this, dapp, stateToResume, task, tx, result, this.enableVerboseContractErrors, readFromDataCacheEnabled, this.enableBlockchainPrintln);

                    if (writeToCacheEnabled) {
                        if (result.isSuccess() && updateDataCache) {
                            dapp.updateLoadedBlockForSuccessfulTransaction(currentBlockNumber);
                            this.hotCache.checkin(addressWrapper, dapp);
                        } else {
                            // For ASSUME_SIDECHAIN, ETH_CALL, MINING cases.
                            dapp.clearDataState();
                            this.hotCache.checkin(addressWrapper, dapp);
                        }
                    }
                }
            }
        }

        if (result.isSuccess()) {
            thisTransactionKernel.commit();
        } else if (result.isFailedUnexpected()) {
            internalLogger.logFatal(result.exception);
        }

        return result;
    }

    @Override
    public AddressResourceMonitor getResourceMonitor() {
        if (null != this.backgroundFatalError) {
            throw this.backgroundFatalError;
        }
        return resourceMonitor;
    }

    private void validateCodeCache(long blockNum){
        // getLoadedDataBlockNum will always be either equal or less than getLoadedCodeBlockNum
        Predicate<SoftReference<LoadedDApp>> condition = (v) -> {
            LoadedDApp dapp = v.get();
            // remove the map entry if the soft reference has been cleared and the referent is null, or dapp has been loaded after blockNum
            return dapp == null || dapp.getLoadedCodeBlockNum() >= blockNum;
        };
        this.hotCache.removeValueIf(condition);
    }

    private void purgeDataCache(){
        Consumer<SoftReference<LoadedDApp>> softReferenceConsumer = (value) -> {
            LoadedDApp dapp = value.get();
            if(dapp != null){
                dapp.clearDataState();
            }
        };
        this.hotCache.apply(softReferenceConsumer);
    }
}
