/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package org.aion.avm.core;

import a.ByteArray;
import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.ManualRevertException;
import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.Transaction;
import foundation.icon.ee.util.Crypto;
import foundation.icon.ee.util.Shadower;
import foundation.icon.ee.util.Unshadower;
import foundation.icon.ee.util.ValueCodec;
import i.GenericPredefinedException;
import i.IBlockchainRuntime;
import i.IInstrumentation;
import i.IObject;
import i.IObjectArray;
import i.IRuntimeSetup;
import i.InstrumentationHelpers;
import org.aion.avm.StorageFees;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.parallel.TransactionTask;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import p.score.AnyDB;
import pi.AnyDBImpl;
import score.RevertException;
import score.ScoreRevertException;

import java.util.Map;

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
    public IObject avm_call(s.java.lang.Class<?> cls,
                            s.java.math.BigInteger value,
                            p.score.Address targetAddress,
                            s.java.lang.String method,
                            IObjectArray params) {
        if (value == null) {
            value = s.java.math.BigInteger.avm_ZERO;
        }
        if (method == null) {
            method = new s.java.lang.String("");
        }
        var dataObj = Map.of(
                "method", method.getUnderlying(),
                "params", getUnderlyingObjects(params)
        );
        return messageCall(cls, value, targetAddress, "call", dataObj);
    }

    @Override
    public p.score.Address avm_deploy(p.score.Address target,
                                      ByteArray content,
                                      IObjectArray params) {
        require(content != null, "Content cannot be NULL");
        if (target == null) {
            // make cx000...000
            byte[] raw = new byte[Address.LENGTH];
            raw[0] = 0x1;
            target = new p.score.Address(raw);
        }
        var dataObj = Map.of(
                "contentType", "application/java",
                "content", content.getUnderlying(),
                "params", getUnderlyingObjects(params)
        );
        return (p.score.Address) messageCall(
                target.avm_getClass(),
                s.java.math.BigInteger.avm_ZERO,
                target,
                "deploy",
                dataObj);
    }

    private Object[] getUnderlyingObjects(IObjectArray sparams) {
        if (sparams == null) {
            sparams = new a.ObjectArray(0);
        }
        Object[] params = new Object[sparams.length()];
        for (int i = 0; i < params.length; i++) {
            params[i] = Unshadower.unshadow(sparams.get(i));
        }
        return params;
    }

    private IObject messageCall(s.java.lang.Class<?> cls,
                                s.java.math.BigInteger value,
                                p.score.Address targetAddress,
                                String dataType,
                                Object dataObj) {
        require(targetAddress != null, "Destination can't be NULL");
        externalState.waitForCallbacks();
        IInstrumentation inst = IInstrumentation.attachedThreadInstrumentation.get();
        var hash = inst.peekNextHashCode();
        long stepLeft = inst.energyLeft();
        var rs = dApp.saveRuntimeState(hash, StorageFees.MAX_GRAPH_SIZE);
        var cid = externalState.getContractID();
        var rds = task.getReentrantDAppStack();
        rds.getTop().setRuntimeState(task.getEID(), rs, cid);
        InstrumentationHelpers.temporarilyExitFrame(this.thisDAppSetup);

        var prevState = rds.getTop();
        rds.pushState();
        foundation.icon.ee.types.Result res = externalState.call(
                new Address(targetAddress.toByteArray()),
                value.getUnderlying(),
                stepLeft,
                dataType,
                dataObj);
        if (res.getStatus() == 0 && prevState != null) {
            prevState.inherit(rds.getTop());
        }
        rds.popState();

        task.setEID(res.getEID());
        task.setPrevEID(res.getPrevEID());

        InstrumentationHelpers.returnToExecutingFrame(this.thisDAppSetup);
        var newRS = rds.getTop().getRuntimeState(task.getPrevEID());
        rds.getTop().removeRuntimeStatesByAddress(cid);
        assert newRS!=null;
        dApp.loadRuntimeState(newRS);
        dApp.invalidateStateCache();
        inst.forceNextHashCode(newRS.getGraph().getNextHash());
        inst.chargeEnergy(res.getStepUsed().longValue());
        int s = res.getStatus();
        if (s == Status.Success) {
            if (cls == null) {
                return Shadower.shadow(res.getRet());
            } else {
                return Shadower.shadowReturnValue(res.getRet(),
                        cls.getRealClass());
            }
        } else if (s == Status.UnknownFailure) {
            throw new RevertException();
        } else if (s == Status.ContractNotFound
                || s == Status.MethodNotFound
                || s == Status.MethodNotPayable
                || s == Status.InvalidParameter
                || s == Status.OutOfBalance
                || s == Status.PackageError) {
            throw new IllegalArgumentException(Status.getMessage(s));
        } else if (s == Status.OutOfStep
                || s == Status.StackOverflow) {
            throw new GenericPredefinedException(s, Status.getMessage(s));
        } else if (s < Status.UserReversionStart) {
            throw new RevertException();
        } else if (s < Status.UserReversionEnd) {
            throw new ScoreRevertException(s - Status.UserReversionStart,
                    res.getRet()==null ? null : res.getRet().toString());
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
        throw new ManualRevertException(code + Status.UserReversionStart, message.getUnderlying());
    }

    @Override
    public void avm_revert(int code) {
        throw new ManualRevertException(code + Status.UserReversionStart);
    }

    @Override
    public void avm_require(boolean condition) {
        if (!condition) {
            throw new ManualRevertException(Status.UserReversionStart);
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
    public ByteArray avm_sha256(ByteArray data) {
        require(null != data, "Input data can't be NULL");
        return new ByteArray(Crypto.sha256(data.getUnderlying()));
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
    public int avm_getFeeSharingProportion() {
        return externalState.getFeeSharingProportion();
    }

    @Override
    public void avm_setFeeSharingProportion(int proportion) {
        if (externalState.isReadOnly()) {
            throw new IllegalStateException();
        }
        if (proportion < 0 || 100 < proportion) {
            throw new IllegalArgumentException();
        }
        externalState.setFeeSharingProportion(proportion);
    }

    @Override
    public AnyDB avm_newAnyDB(s.java.lang.String id, s.java.lang.Class<?> vc) {
        return new AnyDBImpl(id, vc);
    }

    private static boolean isValidEventValue(IObject obj) {
        return (obj instanceof s.java.math.BigInteger ||
                obj instanceof s.java.lang.Boolean ||
                obj instanceof s.java.lang.String ||
                obj instanceof a.ByteArray ||
                obj instanceof p.score.Address);
    }

    @Override
    public void avm_logEvent(IObjectArray indexed, IObjectArray data) {
        if (externalState.isReadOnly()) {
            throw new IllegalStateException();
        }
        if (logger.isTraceEnabled()) {
            logger.trace("Context.logEvent indexed.len={} data.len={}",
                    indexed.length(), data.length());
        }
        int len = Address.LENGTH;
        byte[][] bindexed = new byte[indexed.length()][];
        for (int i=0; i<bindexed.length; i++) {
            IObject v = (IObject)indexed.get(i);
            if (!isValidEventValue(v))
                throw new IllegalArgumentException();
            bindexed[i] = ValueCodec.encode(v);
            len += bindexed[i].length;
            if (logger.isTraceEnabled()) {
                logger.trace("indexed[{}]={}", i, bindexed[i]);
            }
        }
        byte[][] bdata = new byte[data.length()][];
        for (int i=0; i<bdata.length; i++) {
            IObject v = (IObject)data.get(i);
            if (!isValidEventValue(v))
                throw new IllegalArgumentException();
            bdata[i] = ValueCodec.encode(v);
            len += bdata[i].length;
            if (logger.isTraceEnabled()) {
                logger.trace("data[{}]={}", i, bdata[i]);
            }
        }
        var stepCost = externalState.getStepCost();
        int evLogBase = stepCost.eventLogBase();
        int evLog = stepCost.eventLog();
        IInstrumentation.charge(Math.max(evLogBase, len) * evLog);
        externalState.log(bindexed, bdata);
    }
}
