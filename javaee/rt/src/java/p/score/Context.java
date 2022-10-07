/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package p.score;

import a.ByteArray;
import i.IBlockchainRuntime;
import i.IInstrumentation;
import i.IObject;
import i.IObjectArray;
import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.lang.Class;
import s.java.lang.Object;
import s.java.lang.String;
import s.java.math.BigInteger;

public final class Context extends Object {
    public static IBlockchainRuntime blockchainRuntime;

    private Context() {
    }

    // Runtime-facing implementation.

    public static ByteArray avm_getTransactionHash() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getTransactionHash);
        return blockchainRuntime.avm_getTransactionHash();
    }

    public static int avm_getTransactionIndex() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getTransactionIndex);
        return blockchainRuntime.avm_getTransactionIndex();
    }

    public static long avm_getTransactionTimestamp() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getTransactionTimestamp);
        return blockchainRuntime.avm_getTransactionTimestamp();
    }

    public static BigInteger avm_getTransactionNonce() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getTransactionNonce);
        return blockchainRuntime.avm_getTransactionNonce();
    }

    public static Address avm_getAddress() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getAddress);
        return blockchainRuntime.avm_getAddress();
    }

    public static Address avm_getCaller() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getCaller);
        return blockchainRuntime.avm_getCaller();
    }

    public static Address avm_getOrigin() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getOrigin);
        return blockchainRuntime.avm_getOrigin();
    }

    public static Address avm_getOwner() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getOwner);
        return blockchainRuntime.avm_getOwner();
    }

    public static BigInteger avm_getValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getValue);
        return blockchainRuntime.avm_getValue();
    }

    public static long avm_getBlockTimestamp() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getBlockTimestamp);
        return blockchainRuntime.avm_getBlockTimestamp();
    }

    public static long avm_getBlockHeight() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getBlockHeight);
        return blockchainRuntime.avm_getBlockHeight();
    }

    public static BigInteger avm_getBalance(Address address) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getBalance);
        return blockchainRuntime.avm_getBalance(address);
    }

    public static IObject avm_call(Address targetAddress,
                                   String method,
                                   IObjectArray params) {
        return avm_call(null, null, targetAddress, method, params);
    }

    public static<T extends IObject> T avm_call(Class<T> cls,
                                                Address targetAddress,
                                                String method,
                                                IObjectArray params) {
        return avm_call(cls, null, targetAddress, method, params);
    }

    public static IObject avm_call(BigInteger value,
                                   Address targetAddress,
                                   String method,
                                   IObjectArray params) {
        return avm_call(null, value, targetAddress, method, params);
    }

    public static<T extends IObject> T avm_call(Class<T> cls,
                                                BigInteger value,
                                                Address targetAddress,
                                                String method,
                                                IObjectArray params) {
        @SuppressWarnings("unchecked")
        T res = (T)blockchainRuntime.avm_call(cls, value, targetAddress,
                method, params);
        return res;
    }

    public static void avm_transfer(Address targetAddress, BigInteger value) {
        avm_call(value, targetAddress, null, null);
    }

    public static Address avm_deploy(ByteArray content, IObjectArray params) {
        return avm_deploy(null, content, params);
    }

    public static Address avm_deploy(Address target, ByteArray content, IObjectArray params) {
        return blockchainRuntime.avm_deploy(target, content, params);
    }

    public static void avm_revert(int code, String message) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_revert);
        blockchainRuntime.avm_revert(code, message);
    }

    public static void avm_revert(int code) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_revert);
        blockchainRuntime.avm_revert(code);
    }

    public static void avm_revert(String message) {
        avm_revert(0, message);
    }

    public static void avm_revert() {
        avm_revert(0);
    }

    public static void avm_require(boolean condition, String message) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_require);
        blockchainRuntime.avm_require(condition, message);
    }

    public static void avm_require(boolean condition) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_require);
        blockchainRuntime.avm_require(condition);
    }

    public static void avm_println(String message) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_println);
        blockchainRuntime.avm_println(message);
    }

    public static ByteArray avm_hash(String alg, ByteArray data) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_hash_base
                    + RuntimeMethodFeeSchedule.BlockchainRuntime_avm_hash_per_bytes * (data != null ? data.length() : 0));
        return blockchainRuntime.avm_hash(alg, data);
    }

    public static boolean avm_verifySignature(String alg, ByteArray msg,
            ByteArray sig, ByteArray pubKey) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_verifySignature
                        + RuntimeMethodFeeSchedule.BlockchainRuntime_avm_verifySignature_per_bytes * (msg != null ? msg.length() : 0));
        return blockchainRuntime.avm_verifySignature(alg, msg, sig, pubKey);
    }

    public static ByteArray avm_recoverKey(String alg, ByteArray msg,
            ByteArray signature, boolean compressed) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_recoverKey
                        + RuntimeMethodFeeSchedule.BlockchainRuntime_avm_recoverKey_per_bytes * (msg != null ? msg.length() : 0));
        return blockchainRuntime.avm_recoverKey(alg, msg, signature, compressed);
    }

    public static ByteArray avm_aggregate(String type, ByteArray prevAgg, ByteArray values) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_aggregate
                        + RuntimeMethodFeeSchedule.BlockchainRuntime_avm_aggregate_per_bytes * (values != null ? values.length() : 0)
                        + RuntimeMethodFeeSchedule.BlockchainRuntime_avm_aggregate_per_bytes * (prevAgg != null ? prevAgg.length() : 0));
        return blockchainRuntime.avm_aggregate(type, prevAgg, values);
    }

    public static Address avm_getAddressFromKey(ByteArray publicKey) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getAddressFromKey);
        return blockchainRuntime.avm_getAddressFromKey(publicKey);
    }

    public static int avm_getFeeSharingProportion() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getFeeSharingProportion);
        return blockchainRuntime.avm_getFeeSharingProportion();
    }

    public static void avm_setFeeSharingProportion(int proportion) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_setFeeSharingProportion);
        blockchainRuntime.avm_setFeeSharingProportion(proportion);
    }

    public static BranchDB avm_newBranchDB(String id, Class<?> vc) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_newDictDB);
        return blockchainRuntime.avm_newAnyDB(id, vc);
    }

    public static DictDB avm_newDictDB(String id, Class<?> vc) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_newDictDB);
        return blockchainRuntime.avm_newAnyDB(id, vc);
    }

    public static ArrayDB avm_newArrayDB(String id, Class<?> vc) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_newArrayDB);
        return blockchainRuntime.avm_newAnyDB(id, vc);
    }

    public static VarDB avm_newVarDB(String id, Class<?> vc) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_newVarDB);
        return blockchainRuntime.avm_newAnyDB(id, vc);
    }

    public static void avm_logEvent(IObjectArray indexed, IObjectArray data) {
        // Charge steps in BlockchainRuntime
        blockchainRuntime.avm_logEvent(indexed, data);
    }

    public static ObjectReader avm_newByteArrayObjectReader(String codec,
            ByteArray byteArray) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_newByteArrayObjectReader);
        return blockchainRuntime.avm_newByteArrayObjectReader(codec, byteArray);
    }

    public static ByteArrayObjectWriter avm_newByteArrayObjectWriter(
            String codec) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_newByteArrayObjectWriter);
        return blockchainRuntime.avm_newByteArrayObjectWriter(codec);
    }
}
