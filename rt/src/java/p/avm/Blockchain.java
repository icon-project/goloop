package p.avm;

import org.aion.avm.StorageFees;
import a.ByteArray;
import i.IBlockchainRuntime;
import s.java.lang.Object;
import s.java.lang.String;
import s.java.math.BigInteger;

import i.IInstrumentation;
import org.aion.avm.RuntimeMethodFeeSchedule;


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

    public static long avm_getEnergyLimit() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getEnergyLimit);
        return blockchainRuntime.avm_getEnergyLimit();
    }

    public static long avm_getEnergyPrice() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getEnergyPrice);
        return blockchainRuntime.avm_getEnergyPrice();
    }

    public static BigInteger avm_getValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getValue);
        return blockchainRuntime.avm_getValue();
    }

    public static ByteArray avm_getData() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getData);
        return blockchainRuntime.avm_getData();
    }


    public static long avm_getBlockTimestamp() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getBlockTimestamp);
        return blockchainRuntime.avm_getBlockTimestamp();
    }

    public static long avm_getBlockNumber() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getBlockNumber);
        return blockchainRuntime.avm_getBlockNumber();
    }

    public static long avm_getBlockEnergyLimit() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getBlockEnergyLimit);
        return blockchainRuntime.avm_getBlockEnergyLimit();
    }

    public static Address avm_getBlockCoinbase() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getBlockCoinbase);
        return blockchainRuntime.avm_getBlockCoinbase();
    }

    public static BigInteger avm_getBlockDifficulty() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getBlockDifficulty);
        return blockchainRuntime.avm_getBlockDifficulty();
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

    public static BigInteger avm_getBalanceOfThisContract() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getBalanceOfThisContract);
        return blockchainRuntime.avm_getBalanceOfThisContract();
    }

    public static int avm_getCodeSize(Address address) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getCodeSize);
        return blockchainRuntime.avm_getCodeSize(address);
    }


    public static long avm_getRemainingEnergy() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_getRemainingEnergy);
        return blockchainRuntime.avm_getRemainingEnergy();
    }

    public static Result avm_call(Address targetAddress, BigInteger value, ByteArray data, long energyLimit) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_call);
        return blockchainRuntime.avm_call(targetAddress, value, data, energyLimit);
    }

    public static Result avm_create(BigInteger value, ByteArray data, long energyLimit) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_create);
        return blockchainRuntime.avm_create(value, data, energyLimit);
    }

    public static void avm_selfDestruct(Address beneficiary) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_selfDestruct);
        blockchainRuntime.avm_selfDestruct(beneficiary);
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

    public static ByteArray avm_blake2b(ByteArray data) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_blake2b_base
                        + RuntimeMethodFeeSchedule.BlockchainRuntime_avm_blake2b_per_10_bytes * (data != null ? (int) Math.ceil((double) data.length()/10) : 0));
        return blockchainRuntime.avm_blake2b(data);
    }

    public static ByteArray avm_sha256(ByteArray data) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_sha256_base
                    + RuntimeMethodFeeSchedule.BlockchainRuntime_avm_sha256_per_10_bytes * (data != null ?  (int) Math.ceil((double) data.length()/10) : 0));
        return blockchainRuntime.avm_sha256(data);
    }

    public static ByteArray avm_keccak256(ByteArray data){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_keccak256_base
                    + RuntimeMethodFeeSchedule.BlockchainRuntime_avm_keccak256_per_10_bytes * (data != null ?  (int) Math.ceil((double) data.length()/10) : 0));
        return blockchainRuntime.avm_keccak256(data);
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

    public static boolean avm_edVerify(ByteArray data, ByteArray signature, ByteArray publicKey) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BlockchainRuntime_avm_edverify);
        return blockchainRuntime.avm_edVerify(data, signature, publicKey);
    }
}
