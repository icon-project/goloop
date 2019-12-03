package org.aion.avm.core;

import a.ByteArray;
import foundation.icon.ee.utils.Shadower;
import foundation.icon.ee.utils.Unshadower;
import i.CallDepthLimitExceededException;
import i.IBlockchainRuntime;
import i.IInstrumentation;
import i.IObject;
import i.IObjectArray;
import i.IRuntimeSetup;
import i.InstrumentationHelpers;
import i.InvalidException;
import i.RevertException;
import i.RuntimeAssertionError;
import org.aion.avm.StorageFees;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.avm.core.util.LogSizeUtils;
import org.aion.avm.core.util.TransactionResultUtil;
import org.aion.kernel.AvmWrappedTransactionResult;
import org.aion.parallel.TransactionTask;
import org.aion.types.AionAddress;
import org.aion.types.InternalTransaction;
import org.aion.types.Log;
import org.aion.types.Transaction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import p.avm.Address;
import p.avm.CollectionDB;
import p.avm.CollectionDBImpl;
import p.avm.Result;
import p.avm.Value;
import p.avm.ValueBuffer;
import p.avm.VarDB;
import s.java.math.BigInteger;

import java.util.Arrays;
import java.util.List;

/**
 * The implementation of IBlockchainRuntime which is appropriate for exposure as a shadow Object instance within a DApp.
 */
public class BlockchainRuntimeImpl implements IBlockchainRuntime {
    private static final Logger logger = LoggerFactory.getLogger(BlockchainRuntimeImpl.class);
    private final IExternalState externalState;
    private final ReentrantDAppStack.ReentrantState reentrantState;

    private final TransactionTask task;
    private final AionAddress transactionSender;
    private final AionAddress transactionDestination;
    private final Transaction tx;
    private final IRuntimeSetup thisDAppSetup;
    private LoadedDApp dApp;
    private final boolean enablePrintln;

    private Address addressCache;
    private Address callerCache;
    private Address originCache;
    private Address ownerCache;
    private ByteArray transactionHashCache;
    private s.java.math.BigInteger valueCache;
    private s.java.math.BigInteger nonceCache;

    public BlockchainRuntimeImpl(IExternalState externalState,
                                 ReentrantDAppStack.ReentrantState reentrantState,
                                 TransactionTask task,
                                 AionAddress transactionSender,
                                 AionAddress transactionDestination,
                                 Transaction tx,
                                 IRuntimeSetup thisDAppSetup,
                                 LoadedDApp dApp,
                                 boolean enablePrintln) {
        this.externalState = externalState;
        this.reentrantState = reentrantState;
        this.task = task;
        this.transactionSender = transactionSender;
        this.transactionDestination = transactionDestination;
        this.tx = tx;
        this.thisDAppSetup = thisDAppSetup;
        this.dApp = dApp;
        this.enablePrintln = enablePrintln;
    }

    @Override
    public ByteArray avm_getTransactionHash() {
        if (null == this.transactionHashCache) {
            byte[] txHash = tx.copyOfTransactionHash();
            if (txHash != null) {
                this.transactionHashCache = new ByteArray(txHash);
            }
        }
        return this.transactionHashCache;
    }

    @Override
    public int avm_getTransactionIndex() {
        return tx.transactionIndex;
    }

    @Override
    public long avm_getTransactionTimestamp() {
        return tx.transactionTimestamp;
    }

    @Override
    public BigInteger avm_getTransactionNonce() {
        if (null == this.nonceCache) {
            this.nonceCache = new s.java.math.BigInteger(tx.nonce);
        }
        return this.nonceCache;
    }

    @Override
    public Address avm_getAddress() {
        if (null == this.addressCache) {
            this.addressCache = new Address(this.transactionDestination.toByteArray());
        }
        return this.addressCache;
    }

    @Override
    public Address avm_getCaller() {
        if (null == this.callerCache && this.transactionSender != null) {
            this.callerCache = new Address(this.transactionSender.toByteArray());
        }
        return this.callerCache;
    }

    @Override
    public Address avm_getOrigin() {
        if (null == this.originCache && task.getOriginAddress() != null) {
            this.originCache = new Address(task.getOriginAddress().toByteArray());
        }
        return this.originCache;
    }

    @Override
    public Address avm_getOwner() {
        if (null == this.ownerCache) {
            this.ownerCache = new Address(this.externalState.getOwner().toByteArray());
        }
        return this.ownerCache;
    }

    @Override
    public s.java.math.BigInteger avm_getValue() {
        if (null == this.valueCache) {
            java.math.BigInteger value = tx.value;
            this.valueCache = new s.java.math.BigInteger(value);
        }
        return this.valueCache;
    }

    @Override
    public long avm_getBlockTimestamp() {
        return externalState.getBlockTimestamp();
    }

    @Override
    public long avm_getBlockHeight() {
        return externalState.getBlockHeight();
    }

    @Override
    public void avm_putStorage(ByteArray key, ByteArray value, boolean requiresRefund) {
        require(key != null, "Key can't be NULL");
        require(key.getUnderlying().length == 32, "Key must be 32 bytes");

        byte[] keyCopy = Arrays.copyOf(key.getUnderlying(), key.getUnderlying().length);
        byte[] valueCopy = (value == null) ? null : Arrays.copyOf(value.getUnderlying(), value.getUnderlying().length);

        if (value == null) {
            externalState.removeStorage(this.transactionDestination, keyCopy);
        } else {
            externalState.putStorage(this.transactionDestination, keyCopy, valueCopy);
        }
        if (requiresRefund){
            task.addResetStorageKey(this.transactionDestination, keyCopy);
        }
    }

    @Override
    public ByteArray avm_getStorage(ByteArray key) {
        require(key != null, "Key can't be NULL");
        require(key.getUnderlying().length == 32, "Key must be 32 bytes");

        byte[] data = this.externalState.getStorage(this.transactionDestination, key.getUnderlying());
        return (null != data)
            ? new ByteArray(Arrays.copyOf(data, data.length))
            : null;
    }

    @Override
    public s.java.math.BigInteger avm_getBalance(Address address) {
        require(null != address, "Address can't be NULL");
        return new s.java.math.BigInteger(this.externalState.getBalance(new AionAddress(address.toByteArray())));
    }

    @Override
    public IObject avm_call(Address targetAddress,
                            s.java.lang.String method,
                            IObjectArray sparams,
                            s.java.math.BigInteger value) throws IllegalArgumentException {
        java.math.BigInteger underlyingValue = value.getUnderlying();
        require(targetAddress != null, "Destination can't be NULL");
        require(underlyingValue.compareTo(java.math.BigInteger.ZERO) >= 0 , "Value can't be negative");
        require(underlyingValue.compareTo(externalState.getBalance(this.transactionDestination)) <= 0, "Insufficient balance");

        if (task.getTransactionStackDepth() == 9) {
            // since we increase depth in the upcoming call to runInternalCall(),
            // a current depth of 9 means we're about to go up to 10, so we fail
            throw new CallDepthLimitExceededException("Internal call depth cannot be more than 10");
        }
        var hash = IInstrumentation.attachedThreadInstrumentation.get().peekNextHashCode();
        int stepLeft = (int)IInstrumentation.attachedThreadInstrumentation.get().energyLeft();
        var rs = dApp.saveRuntimeState(hash, StorageFees.MAX_GRAPH_SIZE);
        var saveItem = new ReentrantDAppStack.SaveItem(dApp, rs);
        var aionAddr = new AionAddress(avm_getAddress().toByteArray());
        task.getReentrantDAppStack().getTop().getSaveItems().put(aionAddr, saveItem);
        task.incrementTransactionStackDepth();
        InstrumentationHelpers.temporarilyExitFrame(this.thisDAppSetup);
        Object[] params = new Object[sparams.length()];
        for (int i=0; i<params.length; i++) {
            var  p = sparams.get(i);
            params[i] = Unshadower.unshadow((s.java.lang.Object)sparams.get(i));
            if (p!=null && params[i]==null) {
                throw new IllegalArgumentException(String.format("invalid argument at index %d", i));
            }
        }
        foundation.icon.ee.types.Result res = externalState.call(
                new AionAddress(targetAddress.toByteArray()),
                method.getUnderlying(),
                params,
                value.getUnderlying(),
                stepLeft);
        InstrumentationHelpers.returnToExecutingFrame(this.thisDAppSetup);
        task.decrementTransactionStackDepth();
        var saveItems = task.getReentrantDAppStack().getTop().getSaveItems();
        var saveItemFinal = saveItems.remove(aionAddr);
        assert saveItemFinal!=null;
        dApp.loadRuntimeState(saveItemFinal.getRuntimeState());
        IInstrumentation.attachedThreadInstrumentation.get().forceNextHashCode(saveItemFinal.getRuntimeState().getGraph().getNextHash());
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(res.getStepUsed().intValue());
        if (res.getStatus()!=0) {
            // TODO: define exception
            throw new IllegalArgumentException(String.format("Call failed status=%d", res.getStatus()));
        }
        return Shadower.shadow(res.getRet());
    }

    @Override
    public Result avm_call(Address targetAddress, s.java.math.BigInteger value, ByteArray data, long energyLimit) {
        java.math.BigInteger underlyingValue = value.getUnderlying();
        require(targetAddress != null, "Destination can't be NULL");
        require(underlyingValue.compareTo(java.math.BigInteger.ZERO) >= 0 , "Value can't be negative");
        require(underlyingValue.compareTo(externalState.getBalance(this.transactionDestination)) <= 0, "Insufficient balance");
        require(data != null, "Data can't be NULL");
        require(energyLimit >= 0, "Energy limit can't be negative");

        if (task.getTransactionStackDepth() == 9) {
            // since we increase depth in the upcoming call to runInternalCall(),
            // a current depth of 9 means we're about to go up to 10, so we fail
            throw new CallDepthLimitExceededException("Internal call depth cannot be more than 10");
        }

        AionAddress target = new AionAddress(targetAddress.toByteArray());
        if (!externalState.destinationAddressIsSafeForThisVM(target)) {
            throw new IllegalArgumentException("Attempt to execute code using a foreign virtual machine");
        }

        // construct the internal transaction
        InternalTransaction internalTx = InternalTransaction.contractCallTransaction(
                InternalTransaction.RejectedStatus.NOT_REJECTED,
                this.transactionDestination,
                target,
                this.externalState.getNonce(this.transactionDestination),
                underlyingValue,
                data.getUnderlying(),
                restrictEnergyLimit(energyLimit),
                0L);
        
        // Call the common run helper.
        return runInternalCall(internalTx);
    }

    @Override
    public Result avm_create(s.java.math.BigInteger value, ByteArray data, long energyLimit) {
        java.math.BigInteger underlyingValue = value.getUnderlying();
        require(underlyingValue.compareTo(java.math.BigInteger.ZERO) >= 0 , "Value can't be negative");
        require(underlyingValue.compareTo(externalState.getBalance(this.transactionDestination)) <= 0, "Insufficient balance");
        require(data != null, "Data can't be NULL");
        require(energyLimit >= 0, "Energy limit can't be negative");

        if (task.getTransactionStackDepth() == 9) {
            // since we increase depth in the upcoming call to runInternalCall(),
            // a current depth of 9 means we're about to go up to 10, so we fail
            throw new CallDepthLimitExceededException("Internal call depth cannot be more than 10");
        }

        // construct the internal transaction
        InternalTransaction internalTx = InternalTransaction.contractCreateTransaction(
                InternalTransaction.RejectedStatus.NOT_REJECTED,
                this.transactionDestination,
                this.externalState.getNonce(this.transactionDestination),
                underlyingValue,
                data.getUnderlying(),
                restrictEnergyLimit(energyLimit),
                0L);
        
        // Call the common run helper.
        return runInternalCall(internalTx);
    }

    private void require(boolean condition, String message) {
        if (!condition) {
            throw new IllegalArgumentException(message);
        }
    }

    @Override
    public void avm_log(ByteArray data) {
        require(null != data, "data can't be NULL");

        Log log = Log.dataOnly(this.transactionDestination.toByteArray(), data.getUnderlying());
        task.peekSideEffects().addLog(log);
    }

    @Override
    public void avm_log(ByteArray topic1, ByteArray data) {
        require(null != topic1, "topic1 can't be NULL");
        require(null != data, "data can't be NULL");

        Log log = Log.topicsAndData(this.transactionDestination.toByteArray(),
                List.of(LogSizeUtils.truncatePadTopic(topic1.getUnderlying())),
                data.getUnderlying()
        );
        task.peekSideEffects().addLog(log);
    }

    @Override
    public void avm_log(ByteArray topic1, ByteArray topic2, ByteArray data) {
        require(null != topic1, "topic1 can't be NULL");
        require(null != topic2, "topic2 can't be NULL");
        require(null != data, "data can't be NULL");

        Log log = Log.topicsAndData(this.transactionDestination.toByteArray(),
                List.of(LogSizeUtils.truncatePadTopic(topic1.getUnderlying()), LogSizeUtils.truncatePadTopic(topic2.getUnderlying())),
                data.getUnderlying()
        );
        task.peekSideEffects().addLog(log);
    }

    @Override
    public void avm_log(ByteArray topic1, ByteArray topic2, ByteArray topic3, ByteArray data) {
        require(null != topic1, "topic1 can't be NULL");
        require(null != topic2, "topic2 can't be NULL");
        require(null != topic3, "topic3 can't be NULL");
        require(null != data, "data can't be NULL");

        Log log = Log.topicsAndData(this.transactionDestination.toByteArray(),
                List.of(LogSizeUtils.truncatePadTopic(topic1.getUnderlying()), LogSizeUtils.truncatePadTopic(topic2.getUnderlying()), LogSizeUtils.truncatePadTopic(topic3.getUnderlying())),
                data.getUnderlying()
        );
        task.peekSideEffects().addLog(log);
    }

    @Override
    public void avm_log(ByteArray topic1, ByteArray topic2, ByteArray topic3, ByteArray topic4, ByteArray data) {
        require(null != topic1, "topic1 can't be NULL");
        require(null != topic2, "topic2 can't be NULL");
        require(null != topic3, "topic3 can't be NULL");
        require(null != topic4, "topic4 can't be NULL");
        require(null != data, "data can't be NULL");

        Log log = Log.topicsAndData(this.transactionDestination.toByteArray(),
                List.of(LogSizeUtils.truncatePadTopic(topic1.getUnderlying()), LogSizeUtils.truncatePadTopic(topic2.getUnderlying()), LogSizeUtils.truncatePadTopic(topic3.getUnderlying()), LogSizeUtils.truncatePadTopic(topic4.getUnderlying())),
                data.getUnderlying()
        );
        task.peekSideEffects().addLog(log);
    }

    @Override
    public void avm_revert() {
        throw new RevertException();
    }

    @Override
    public void avm_invalid() {
        throw new InvalidException();
    }

    @Override
    public void avm_require(boolean condition) {
        if (!condition) {
            throw new RevertException();
        }
    }

    @Override
    public void avm_print(s.java.lang.String message) {
        if (this.enablePrintln) {
            task.outputPrint(message.toString());
        }
    }

    @Override
    public void avm_println(s.java.lang.String message) {
        if (this.enablePrintln) {
            task.outputPrintln(message!=null ? message.toString() : null);
        }
    }

    private long restrictEnergyLimit(long energyLimit) {
        long remainingEnergy = IInstrumentation.attachedThreadInstrumentation.get().energyLeft();
        long maxAllowed = remainingEnergy - (remainingEnergy >> 6);
        return Math.min(maxAllowed, energyLimit);
    }

    private Result runInternalCall(InternalTransaction internalTx) {
        // add the internal transaction to result
        task.peekSideEffects().addInternalTransaction(internalTx);

        // we should never leave this method without decrementing this
        task.incrementTransactionStackDepth();

        IInstrumentation currentThreadInstrumentation = IInstrumentation.attachedThreadInstrumentation.get();
        if (null != this.reentrantState) {
            // Note that we want to save out the current nextHashCode.
            int nextHashCode = currentThreadInstrumentation.peekNextHashCode();
            this.reentrantState.updateNextHashCode(nextHashCode);
        }
        // Temporarily detach from the DApp we were in.
        InstrumentationHelpers.temporarilyExitFrame(this.thisDAppSetup);

        // Create the Transaction.
//        Transaction transaction = AvmTransactionUtil.fromInternalTransaction(internalTx);

        // Acquire the target of the internal transaction
//        AionAddress destination = (transaction.isCreate) ? this.capabilities.generateContractAddress(transaction) : transaction.destinationAddress;
        boolean isAcquired = false; //avm.getResourceMonitor().acquire(destination.toByteArray(), task);

        // execute the internal transaction
        AvmWrappedTransactionResult newResult = null;
        try {
            if(isAcquired) {
                //newResult = this.avm.runInternalTransaction(this.externalState, this.task, transaction);
            } else {
                // Unsuccessful acquire means transaction task has been aborted.
                // In abort case, internal transaction will not be executed.
                newResult = TransactionResultUtil.newAbortedResultWithZeroEnergyUsed();
            }
        } finally {
            // Re-attach.
            InstrumentationHelpers.returnToExecutingFrame(this.thisDAppSetup);
        }
        
        if (null != this.reentrantState) {
            // Update the next hashcode counter, in case this was a reentrant call and it was changed.
            currentThreadInstrumentation.forceNextHashCode(this.reentrantState.getNextHashCode());
        }

        // Note that we can only meaningfully charge energy if the transaction was NOT aborted and it actually ran something (balance transfers report zero energy used, here).
        if (isAcquired) {
            // charge energy consumed
            long energyUsed = newResult.energyUsed();
            if (0L != energyUsed) {
                // We know that this must be a positive integer.
                RuntimeAssertionError.assertTrue(energyUsed > 0L);
                RuntimeAssertionError.assertTrue(energyUsed <= (long)Integer.MAX_VALUE);
                currentThreadInstrumentation.chargeEnergy((int)energyUsed);
            }
        }

        task.decrementTransactionStackDepth();

        // TODO
        byte[] output = new byte[0];
        return new Result(newResult.isSuccess(), output == null ? null : new ByteArray(output));
    }

    public CollectionDB avm_newCollectionDB(int type, s.java.lang.String id) {
        return new CollectionDBImpl(type, id);
    }

    public VarDB avm_newVarDB(s.java.lang.String id) {
        return new p.avm.VarDBImpl(id);
    }

    public void avm_log(IObjectArray indexed, IObjectArray data) {
        if (logger.isTraceEnabled()) {
            logger.trace("Blockchain.log indxed.len={} data.len={}", indexed.length(), data.length());
            for (int i=0; i<indexed.length(); i++) {
                var v = indexed.get(i);
                if (v instanceof ValueBuffer) {
                    logger.trace("indexed[{}]={}", i, ((ValueBuffer)v).asByteArray());
                }
            }
            for (int i=0; i<data.length(); i++) {
                var v = data.get(i);
                if (v instanceof ValueBuffer) {
                    logger.trace("data[{}]={}", i, ((ValueBuffer)v).asByteArray());
                }
            }
        }
        byte[][] bindexed = new byte[indexed.length()][];
        for (int i=0; i<bindexed.length; i++) {
            Value v = (Value)indexed.get(i);
            if (v instanceof ValueBuffer) {
                bindexed[i] = ((ValueBuffer)v).asByteArray();
            } else {
                bindexed[i] = v.avm_asByteArray().getUnderlying();
            }
        }
        byte[][] bdata = new byte[data.length()][];
        for (int i=0; i<bdata.length; i++) {
            Value v = (Value)data.get(i);
            if (v instanceof ValueBuffer) {
                bdata[i] = ((ValueBuffer)v).asByteArray();
            } else {
                bdata[i] = v.avm_asByteArray().getUnderlying();
            }
        }
        externalState.log(bindexed, bdata);
    }
}
