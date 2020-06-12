/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package org.aion.avm.core;

import a.ByteArray;
import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.Transaction;
import foundation.icon.ee.util.Crypto;
import foundation.icon.ee.util.Shadower;
import foundation.icon.ee.util.Unshadower;
import i.GenericCodedException;
import i.IBlockchainRuntime;
import i.IInstrumentation;
import i.IObject;
import i.IObjectArray;
import i.IRuntimeSetup;
import i.InstrumentationHelpers;
import i.RuntimeAssertionError;
import org.aion.avm.StorageFees;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.parallel.TransactionTask;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import p.score.CollectionDB;
import p.score.Value;
import p.score.ValueBuffer;
import p.score.VarDB;
import pi.CollectionDBImpl;
import pi.VarDBImpl;
import score.RevertException;
import score.ScoreRevertException;

/**
 * The implementation of IBlockchainRuntime which is appropriate for exposure as a shadow Object instance within a DApp.
 */
public class BlockchainRuntimeImpl implements IBlockchainRuntime {
    private static final Logger logger = LoggerFactory.getLogger(BlockchainRuntimeImpl.class);
    private final IExternalState externalState;

    private final TransactionTask task;
    private final Address transactionSender;
    private final Address transactionDestination;
    private final Transaction tx;
    private final IRuntimeSetup thisDAppSetup;
    private final LoadedDApp dApp;
    private final boolean enablePrintln;

    private p.score.Address addressCache;
    private p.score.Address callerCache;
    private p.score.Address originCache;
    private p.score.Address ownerCache;
    private ByteArray transactionHashCache;
    private s.java.math.BigInteger valueCache;
    private s.java.math.BigInteger nonceCache;

    public BlockchainRuntimeImpl(IExternalState externalState,
                                 TransactionTask task,
                                 Address transactionSender,
                                 Address transactionDestination,
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
        return tx.getTxIndex();
    }

    @Override
    public long avm_getTransactionTimestamp() {
        return tx.getTxTimestamp();
    }

    @Override
    public s.java.math.BigInteger avm_getTransactionNonce() {
        if (null == this.nonceCache) {
            this.nonceCache = new s.java.math.BigInteger(tx.getNonce());
        }
        return this.nonceCache;
    }

    @Override
    public p.score.Address avm_getAddress() {
        if (null == this.addressCache) {
            this.addressCache = new p.score.Address(this.transactionDestination.toByteArray());
        }
        return this.addressCache;
    }

    @Override
    public p.score.Address avm_getCaller() {
        if (null == this.callerCache && this.transactionSender != null) {
            this.callerCache = new p.score.Address(this.transactionSender.toByteArray());
        }
        return this.callerCache;
    }

    @Override
    public p.score.Address avm_getOrigin() {
        if (null == this.originCache && task.getOriginAddress() != null) {
            this.originCache = new p.score.Address(task.getOriginAddress().toByteArray());
        }
        return this.originCache;
    }

    @Override
    public p.score.Address avm_getOwner() {
        if (null == this.ownerCache) {
            this.ownerCache = new p.score.Address(this.externalState.getOwner().toByteArray());
        }
        return this.ownerCache;
    }

    @Override
    public s.java.math.BigInteger avm_getValue() {
        if (null == this.valueCache) {
            this.valueCache = new s.java.math.BigInteger(tx.getValue());
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
    public s.java.math.BigInteger avm_getBalance(p.score.Address address) {
        require(null != address, "Address can't be NULL");
        return new s.java.math.BigInteger(this.externalState.getBalance(new Address(address.toByteArray())));
    }

    @Override
    public IObject avm_call(s.java.math.BigInteger value,
                            p.score.Address targetAddress,
                            s.java.lang.String method,
                            IObjectArray sparams) {
        if (value == null) {
            value = s.java.math.BigInteger.avm_ZERO;
        }
        if (method == null) {
            method = new s.java.lang.String("");
        }
        if (sparams == null) {
            sparams = new a.ObjectArray(0);
        }
        require(targetAddress != null, "Destination can't be NULL");

        externalState.waitForCallbacks();
        IInstrumentation inst = IInstrumentation.attachedThreadInstrumentation.get();
        var hash = inst.peekNextHashCode();
        int stepLeft = (int)inst.energyLeft();
        var rs = dApp.saveRuntimeState(hash, StorageFees.MAX_GRAPH_SIZE);
        var saveItem = new ReentrantDAppStack.SaveItem(dApp, rs);
        var callerAddr = this.transactionDestination;
        task.getReentrantDAppStack().getTop().getSaveItems().put(callerAddr, saveItem);
        InstrumentationHelpers.temporarilyExitFrame(this.thisDAppSetup);
        Object[] params = new Object[sparams.length()];
        for (int i=0; i<params.length; i++) {
            params[i] = Unshadower.unshadow((s.java.lang.Object)sparams.get(i));
        }
        foundation.icon.ee.types.Result res = externalState.call(
                new Address(targetAddress.toByteArray()),
                method.getUnderlying(),
                params,
                value.getUnderlying(),
                stepLeft);
        InstrumentationHelpers.returnToExecutingFrame(this.thisDAppSetup);
        var saveItems = task.getReentrantDAppStack().getTop().getSaveItems();
        var saveItemFinal = saveItems.remove(callerAddr);
        assert saveItemFinal!=null;
        dApp.loadRuntimeState(saveItemFinal.getRuntimeState());
        inst.forceNextHashCode(saveItemFinal.getRuntimeState().getGraph().getNextHash());
        inst.chargeEnergy(res.getStepUsed().intValue());
        int s = res.getStatus();
        if (s == Status.Success) {
            return Shadower.shadow(res.getRet());
        } else if (s == Status.UnknownFailure) {
            throw new RevertException();
        } else if (s == Status.ContractNotFound
                || s == Status.MethodNotFound
                || s == Status.MethodNotPayable
                || s == Status.InvalidParameter
                || s == Status.OutOfBalance) {
            throw new IllegalArgumentException(Status.getMessage(s));
        } else if (s == Status.OutOfStep
                || s == Status.StackOverflow) {
            throw new GenericCodedException(s, Status.getMessage(s));
        } else if (s < Status.UserReversionStart) {
            throw new RevertException();
        } else if (s < Status.UserReversionEnd) {
            throw new ScoreRevertException(s - Status.UserReversionStart,
                    res.getRet().toString());
        }
        throw new RevertException();
    }

    private void require(boolean condition, String message) {
        if (!condition) {
            throw new IllegalArgumentException(message);
        }
    }

    @Override
    public void avm_revert(int code, s.java.lang.String message) {
        throw new GenericCodedException(code + Status.UserReversionStart, message.getUnderlying());
    }

    @Override
    public void avm_revert(int code) {
        throw new GenericCodedException(code + Status.UserReversionStart);
    }

    @Override
    public void avm_require(boolean condition) {
        if (!condition) {
            throw new GenericCodedException(Status.UserReversionStart);
        }
    }

    @Override
    public void avm_println(s.java.lang.String message) {
        if (this.enablePrintln) {
            logger.trace("PRT| " + (message!=null ? message.toString() : "<null>"));
        }
    }

    @Override
    public ByteArray avm_sha3_256(ByteArray data) {
        require(null != data, "Input data can't be NULL");
        return new ByteArray(Crypto.sha3_256(data.getUnderlying()));
    }

    @Override
    public ByteArray avm_recoverKey(ByteArray msgHash, ByteArray signature, boolean compressed) {
        require(null != msgHash && null != signature, "msgHash or signature is NULL");
        byte[] msgBytes = msgHash.getUnderlying();
        byte[] sigBytes = signature.getUnderlying();
        require(msgBytes.length == 32, "the length of msgHash must be 32");
        require(sigBytes.length == 65, "the length of signature must be 65");
        return new ByteArray(Crypto.recoverKey(msgBytes, sigBytes, compressed));
    }

    @Override
    public p.score.Address avm_getAddressFromKey(ByteArray publicKey) {
        require(null != publicKey, "publicKey is NULL");
        return new p.score.Address(Crypto.getAddressBytesFromKey(publicKey.getUnderlying()));
    }

    @Override
    public CollectionDB avm_newCollectionDB(int type,
                                            s.java.lang.String id,
                                            s.java.lang.Class<?> vc) {
        return new CollectionDBImpl(type, id, vc);
    }

    @Override
    public VarDB avm_newVarDB(s.java.lang.String id, s.java.lang.Class<?> vc) {
        return new VarDBImpl(id, vc);
    }

    @Override
    public void avm_log(IObjectArray indexed, IObjectArray data) {
        if (externalState.isReadOnly()) {
            throw new IllegalStateException();
        }
        if (logger.isTraceEnabled()) {
            logger.trace("Context.log indexed.len={} data.len={}", indexed.length(), data.length());
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
        int len = Address.LENGTH;
        byte[][] bindexed = new byte[indexed.length()][];
        for (int i=0; i<bindexed.length; i++) {
            Value v = (Value)indexed.get(i);
            if (v instanceof ValueBuffer) {
                bindexed[i] = ((ValueBuffer)v).asByteArray();
            } else {
                bindexed[i] = v.avm_asByteArray().getUnderlying();
            }
            len += bindexed[i].length;
        }
        byte[][] bdata = new byte[data.length()][];
        for (int i=0; i<bdata.length; i++) {
            Value v = (Value)data.get(i);
            if (v instanceof ValueBuffer) {
                bdata[i] = ((ValueBuffer)v).asByteArray();
            } else {
                bdata[i] = v.avm_asByteArray().getUnderlying();
            }
            len += bdata[i].length;
        }
        var stepCost = externalState.getStepCost();
        int evLogBase = stepCost.eventLogBase();
        int evLog = stepCost.eventLog();
        IInstrumentation.charge(Math.max(evLogBase, len) * evLog);
        externalState.log(bindexed, bdata);
    }
}
