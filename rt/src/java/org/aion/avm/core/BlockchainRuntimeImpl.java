package org.aion.avm.core;

import java.util.Arrays;
import org.aion.avm.core.util.TransactionResultUtil;
import org.aion.types.AionAddress;
import org.aion.types.Transaction;
import s.java.math.BigInteger;
import p.avm.Address;
import p.avm.Result;
import i.*;
import a.ByteArray;
import org.aion.types.InternalTransaction;
import org.aion.avm.core.util.LogSizeUtils;
import org.aion.kernel.*;
import org.aion.parallel.TransactionTask;
import org.aion.types.Log;

import java.util.List;

/**
 * The implementation of IBlockchainRuntime which is appropriate for exposure as a shadow Object instance within a DApp.
 */
public class BlockchainRuntimeImpl implements IBlockchainRuntime {
    private final IExternalCapabilities capabilities;
    private final IExternalState externalState;
    private final AvmInternal avm;
    private final ReentrantDAppStack.ReentrantState reentrantState;

    private Transaction tx;
    private final AionAddress transactionDestination;
    private final byte[] dAppData;
    private TransactionTask task;
    private final IRuntimeSetup thisDAppSetup;
    private final boolean enablePrintln;

    private ByteArray dAppDataCache;
    private Address addressCache;
    private Address callerCache;
    private Address originCache;
    private BigInteger valueCache;
    private Address blockCoinBaseCache;
    private BigInteger blockDifficultyCache;


    public BlockchainRuntimeImpl(IExternalCapabilities capabilities, IExternalState externalState, AvmInternal avm, ReentrantDAppStack.ReentrantState reentrantState, TransactionTask task, Transaction tx, byte[] dAppData, IRuntimeSetup thisDAppSetup, boolean enablePrintln) {
        this.capabilities = capabilities;
        this.externalState = externalState;
        this.avm = avm;
        this.reentrantState = reentrantState;
        this.tx = tx;
        this.transactionDestination = (tx.isCreate) ? capabilities.generateContractAddress(tx) : tx.destinationAddress;
        this.dAppData = dAppData;
        this.task = task;
        this.thisDAppSetup = thisDAppSetup;
        this.enablePrintln = enablePrintln;

        this.dAppDataCache = null;
        this.addressCache = null;
        this.callerCache = null;
        this.originCache = null;
        this.valueCache = null;
        this.blockCoinBaseCache = null;
        this.blockDifficultyCache = null;
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
        if (null == this.callerCache) {
            this.callerCache = new Address(tx.senderAddress.toByteArray());
        }

        return this.callerCache;
    }

    @Override
    public Address avm_getOrigin() {
        if (null == this.originCache) {
            this.originCache = new Address(task.getOriginAddress().toByteArray());
        }

        return this.originCache;
    }

    @Override
    public long avm_getEnergyLimit() {
        return tx.energyLimit;
    }

    @Override
    public long avm_getEnergyPrice() {
        return tx.energyPrice;
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
    public ByteArray avm_getData() {
        if (null == this.dAppDataCache) {
            this.dAppDataCache = (null != this.dAppData)
                    ? new ByteArray(this.dAppData.clone())
                    : null;
        }

        return this.dAppDataCache;
    }


    @Override
    public long avm_getBlockTimestamp() {
        return externalState.getBlockTimestamp();
    }

    @Override
    public long avm_getBlockNumber() {
        return externalState.getBlockNumber();
    }

    @Override
    public long avm_getBlockEnergyLimit() {
        return externalState.getBlockEnergyLimit();
    }

    @Override
    public Address avm_getBlockCoinbase() {
        if (null == this.blockCoinBaseCache) {
            this.blockCoinBaseCache = new Address(externalState.getMinerAddress().toByteArray());
        }

        return this.blockCoinBaseCache;
    }

    @Override
    public s.java.math.BigInteger avm_getBlockDifficulty() {
        if (null == this.blockDifficultyCache) {
            this.blockDifficultyCache = new s.java.math.BigInteger(externalState.getBlockDifficulty());
        }

        return this.blockDifficultyCache;
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
        if(requiresRefund){
            task.addResetStoragekey(this.transactionDestination, keyCopy);
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

        // Acquire resource before reading
        // Returned result of acquire is not checked, since an abort exception will be thrown by IInstrumentation during chargeEnergy if the task has been aborted
        avm.getResourceMonitor().acquire(address.toByteArray(), this.task);
        return new s.java.math.BigInteger(this.externalState.getBalance(new AionAddress(address.toByteArray())));
    }

    @Override
    public s.java.math.BigInteger avm_getBalanceOfThisContract() {
        // This method can be called inside clinit so CREATE is a valid context.
        // Acquire resource before reading
        // Returned result of acquire is not checked, since an abort exception will be thrown by IInstrumentation during chargeEnergy if the task has been aborted
        avm.getResourceMonitor().acquire(this.transactionDestination.toByteArray(), this.task);
        return new s.java.math.BigInteger(this.externalState.getBalance(this.transactionDestination));
    }

    @Override
    public int avm_getCodeSize(Address address) {
        require(null != address, "Address can't be NULL");

        // Acquire resource before reading
        // Returned result of acquire is not checked, since an abort exception will be thrown by IInstrumentation during chargeEnergy if the task has been aborted
        avm.getResourceMonitor().acquire(address.toByteArray(), this.task);
        byte[] vc = this.externalState.getCode(new AionAddress(address.toByteArray()));
        return vc == null ? 0 : vc.length;
    }

    @Override
    public long avm_getRemainingEnergy() {
        return IInstrumentation.attachedThreadInstrumentation.get().energyLeft();
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
                tx.energyPrice);
        
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
                tx.energyPrice);
        
        // Call the common run helper.
        return runInternalCall(internalTx);
    }

    private void require(boolean condition, String message) {
        if (!condition) {
            throw new IllegalArgumentException(message);
        }
    }

    @Override
    public void avm_selfDestruct(Address beneficiary) {
        require(null != beneficiary, "Beneficiary can't be NULL");

        // Acquire beneficiary address, the address of current contract is already locked at this stage.
        // Returned result of acquire is not checked, since an abort exception will be thrown by IInstrumentation during chargeEnergy if the task has been aborted
        this.avm.getResourceMonitor().acquire(beneficiary.toByteArray(), this.task);

        // Value transfer
        java.math.BigInteger balanceToTransfer = this.externalState.getBalance(this.transactionDestination);
        this.externalState.adjustBalance(this.transactionDestination, balanceToTransfer.negate());
        this.externalState
            .adjustBalance(new AionAddress(beneficiary.toByteArray()), balanceToTransfer);

        // Delete Account
        // Note that the account being deleted means it will still run but no DApp which sees this delete
        // (the current one and any callers, or any later transactions, assuming this commits) will be able
        // to invoke it (the code will be missing).
        this.externalState.deleteAccount(this.transactionDestination);
        task.addSelfDestructAddress(this.transactionDestination);
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
    public ByteArray avm_blake2b(ByteArray data) {
        require(null != data, "Input data can't be NULL");

        return new ByteArray(this.capabilities.blake2b(data.getUnderlying()));
    }

    @Override
    public ByteArray avm_sha256(ByteArray data){
        require(null != data, "Input data can't be NULL");

        return new ByteArray(this.capabilities.sha256(data.getUnderlying()));
    }

    @Override
    public ByteArray avm_keccak256(ByteArray data){
        require(null != data, "Input data can't be NULL");

        return new ByteArray(this.capabilities.keccak256(data.getUnderlying()));
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
            task.outputPrintln(message.toString());
        }
    }

    @Override
    public boolean avm_edVerify(ByteArray data, ByteArray signature, ByteArray publicKey) throws IllegalArgumentException {
        require(null != data, "Input data can't be NULL");
        require(null != signature, "Input signature can't be NULL");
        require(null != publicKey, "Input public key can't be NULL");

        return this.capabilities.verifyEdDSA(data.getUnderlying(), signature.getUnderlying(), publicKey.getUnderlying());
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
        Transaction transaction = AvmTransactionUtil.fromInternalTransaction(internalTx);

        // Acquire the target of the internal transaction
        AionAddress destination = (transaction.isCreate) ? this.capabilities.generateContractAddress(transaction) : transaction.destinationAddress;
        boolean isAcquired = avm.getResourceMonitor().acquire(destination.toByteArray(), task);

        // execute the internal transaction
        AvmWrappedTransactionResult newResult = null;
        try {
            if(isAcquired) {
                newResult = this.avm.runInternalTransaction(this.externalState, this.task, transaction);
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

        // charge energy consumed
        currentThreadInstrumentation.chargeEnergy(newResult.energyUsed());

        task.decrementTransactionStackDepth();

        byte[] output = newResult.output();
        return new Result(newResult.isSuccess(), output == null ? null : new ByteArray(output));
    }
}
