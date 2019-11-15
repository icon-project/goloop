package p.avm;

import a.ByteArray;
import i.DBImplBase;
import i.IBlockchainRuntime;
import i.IInstrumentation;
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

    public static Result avm_call(Address targetAddress, BigInteger value, ByteArray data, long energyLimit) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_call);
        return blockchainRuntime.avm_call(targetAddress, value, data, energyLimit);
    }

    public static Result avm_create(BigInteger value, ByteArray data, long energyLimit) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_create);
        return blockchainRuntime.avm_create(value, data, energyLimit);
    }

    public static void avm_log(ByteArray data) {
        int dataSize = data != null ? data.length() : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_log_base
                        + RuntimeMethodFeeSchedule.BlockchainRuntime_avm_log_per_data_byte * dataSize);
        blockchainRuntime.avm_log(data);
    }

    public static void avm_log(ByteArray topic1, ByteArray data) {
        int dataSize = data != null ? data.length() : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_log_base
                        + RuntimeMethodFeeSchedule.BlockchainRuntime_avm_log_per_topic
                        + RuntimeMethodFeeSchedule.BlockchainRuntime_avm_log_per_data_byte * dataSize);
        blockchainRuntime.avm_log(topic1, data);
    }

    public static void avm_log(ByteArray topic1, ByteArray topic2, ByteArray data) {
        int dataSize = data != null ? data.length() : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_log_base
                        + 2 * RuntimeMethodFeeSchedule.BlockchainRuntime_avm_log_per_topic
                        + RuntimeMethodFeeSchedule.BlockchainRuntime_avm_log_per_data_byte * dataSize);
        blockchainRuntime.avm_log(topic1, topic2, data);
    }

    public static void avm_log(ByteArray topic1, ByteArray topic2, ByteArray topic3, ByteArray data) {
        int dataSize = data != null ? data.length() : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_log_base
                        + 3 * RuntimeMethodFeeSchedule.BlockchainRuntime_avm_log_per_topic
                        + RuntimeMethodFeeSchedule.BlockchainRuntime_avm_log_per_data_byte * dataSize);
        blockchainRuntime.avm_log(topic1, topic2, topic3, data);
    }

    public static void avm_log(ByteArray topic1, ByteArray topic2, ByteArray topic3, ByteArray topic4, ByteArray data) {
        int dataSize = data != null ? data.length() : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_log_base
                        + 4 * RuntimeMethodFeeSchedule.BlockchainRuntime_avm_log_per_topic
                        + RuntimeMethodFeeSchedule.BlockchainRuntime_avm_log_per_data_byte * dataSize);
        blockchainRuntime.avm_log(topic1, topic2, topic3, topic4, data);
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

    public static void avm_print(String message) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_print);
        blockchainRuntime.avm_print(message);
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
