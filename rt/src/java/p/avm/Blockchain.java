package p.avm;

import a.ByteArray;
import i.DBImplBase;
import i.IBlockchainRuntime;
import i.IInstrumentation;
import i.IObject;
import i.IObjectArray;
import org.aion.avm.RuntimeMethodFeeSchedule;
import org.aion.avm.StorageFees;
import s.java.lang.Object;
import s.java.lang.String;
import s.java.math.BigInteger;


public final class Blockchain extends Object {
    public static IBlockchainRuntime blockchainRuntime;

    private Blockchain() {
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

    public static void avm_putStorage(ByteArray key, ByteArray value) {
        boolean requiresRefund =  false;
        int valueSize = value != null ? value.length() : 0;
        ByteArray storage = blockchainRuntime.avm_getStorage(key);
        if (storage == null && value != null) {
            // zero to nonzero
            IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_setStorage + StorageFees.WRITE_PRICE_PER_BYTE * valueSize);
        } else if (storage != null && value == null) {
            // nonzero to zero
            IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_resetStorage);
            requiresRefund = true;
        } else if (storage == null && value == null) {
            // zero to zero
            IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_resetStorage);
        } else {
            //nonzero to nonzero
            IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                    RuntimeMethodFeeSchedule.BlockchainRuntime_avm_resetStorage + StorageFees.WRITE_PRICE_PER_BYTE * valueSize);
        }
        blockchainRuntime.avm_putStorage(key, value, requiresRefund);
    }

    public static ByteArray avm_getStorage(ByteArray key) {
        // Note that we must charge the linear portion of the read _after_ the read happens.
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getStorage);
        ByteArray value = blockchainRuntime.avm_getStorage(key);
        int valueSize = value != null ? value.length() : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(StorageFees.READ_PRICE_PER_BYTE * valueSize);
        return value;
    }

    public static BigInteger avm_getBalance(Address address) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getBalance);
        return blockchainRuntime.avm_getBalance(address);
    }

    public static IObject avm_call(Address targetAddress, String method, IObjectArray params,
                                   BigInteger value) throws IllegalArgumentException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_call);
        return blockchainRuntime.avm_call(targetAddress, method, params, value);
    }

    public static void avm_revert() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_revert);
        blockchainRuntime.avm_revert();
    }

    public static void avm_invalid() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_invalid);
        blockchainRuntime.avm_invalid();
    }

    public static void avm_require(boolean condition) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_require);
        blockchainRuntime.avm_require(condition);
    }

    public static void avm_println(String message) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_println);
        blockchainRuntime.avm_println(message);
    }

    public static NestingDictDB avm_newNestingDictDB(String id) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_newDictDB);
        return blockchainRuntime.avm_newCollectionDB(DBImplBase.TYPE_DICT_DB, id);
    }

    public static DictDB avm_newDictDB(String id) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_newDictDB);
        return blockchainRuntime.avm_newCollectionDB(DBImplBase.TYPE_DICT_DB, id);
    }

    public static ArrayDB avm_newArrayDB(String id) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_newArrayDB);
        return blockchainRuntime.avm_newCollectionDB(DBImplBase.TYPE_ARRAY_DB, id);
    }

    public static VarDB avm_newVarDB(String id) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_newVarDB);
        return blockchainRuntime.avm_newVarDB(id);
    }

    public static void avm_log(IObjectArray indexed, IObjectArray data) {
        blockchainRuntime.avm_log(indexed, data);
    }
}
