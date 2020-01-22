package org.aion.avm.core;

import a.ByteArray;
import avm.TargetRevertedException;
import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.SystemException;
import foundation.icon.ee.util.Shadower;
import foundation.icon.ee.util.Unshadower;
import i.CallDepthLimitExceededException;
import i.IBlockchainRuntime;
import i.IInstrumentation;
import i.IObject;
import i.IObjectArray;
import i.IRuntimeSetup;
import i.InstrumentationHelpers;
import i.RevertException;
import org.aion.avm.StorageFees;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.parallel.TransactionTask;
import org.aion.types.AionAddress;
import org.aion.types.Transaction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import p.avm.Address;
import p.avm.CollectionDB;
import pi.CollectionDBImpl;
import p.avm.Value;
import p.avm.ValueBuffer;
import p.avm.VarDB;
import pi.VarDBImpl;
import s.java.math.BigInteger;

import java.util.Arrays;

/**
 * The implementation of IBlockchainRuntime which is appropriate for exposure as a shadow Object instance within a DApp.
 */
public class BlockchainRuntimeImpl implements IBlockchainRuntime {
    private static final Logger logger = LoggerFactory.getLogger(BlockchainRuntimeImpl.class);
    private final IExternalState externalState;

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
                                 TransactionTask task,
                                 AionAddress transactionSender,
                                 AionAddress transactionDestination,
                                 Transaction tx,
                                 IRuntimeSetup thisDAppSetup,
                                 LoadedDApp dApp,
                                 boolean enablePrintln) {
        this.externalState = externalState;
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
    public IObject avm_call(s.java.math.BigInteger value,
                            s.java.math.BigInteger stepLimit,
                            Address targetAddress,
                            s.java.lang.String method,
                            IObjectArray sparams) {
        // FIXME
        if (value == null)
            value = new BigInteger(java.math.BigInteger.ZERO);
        java.math.BigInteger underlyingValue = value.getUnderlying();
        require(targetAddress != null, "Destination can't be NULL");
        require(underlyingValue.compareTo(java.math.BigInteger.ZERO) >= 0 , "Value can't be negative");

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
        if (res.getStatus() > 0 && res.getStatus() < Status.UserReversionStart) {
            throw new SystemException(res.getStatus(), String.format("address=%s method=%s status=%d %s", targetAddress, method, res.getStatus(), res.getRet()));
        } else if (res.getStatus() >= Status.UserReversionStart) {
            throw new TargetRevertedException(res.getStatus(), String.format("address=%s method=%s status=%d %s", targetAddress, method, res.getStatus(), res.getRet()));
        }
        return Shadower.shadow(res.getRet());
    }

    private void require(boolean condition, String message) {
        if (!condition) {
            throw new IllegalArgumentException(message);
        }
    }

    @Override
    public void avm_revert(int code, s.java.lang.String message) {
        throw new RevertException(code, message.getUnderlying());
    }

    @Override
    public void avm_revert(int code) {
        throw new RevertException(code);
    }

    @Override
    public void avm_require(boolean condition) {
        if (!condition) {
            throw new RevertException();
        }
    }

    @Override
    public void avm_println(s.java.lang.String message) {
        if (this.enablePrintln) {
            logger.trace("PRT| " + (message!=null ? message.toString() : "<null>"));
        }
    }

    public CollectionDB avm_newCollectionDB(int type,
                                            s.java.lang.String id,
                                            s.java.lang.Class<?> vc) {
        return new CollectionDBImpl(type, id, vc);
    }

    public VarDB avm_newVarDB(s.java.lang.String id, s.java.lang.Class<?> vc) {
        return new VarDBImpl(id, vc);
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
