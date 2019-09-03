package avm;

import java.math.BigInteger;

/**
 * Every DApp has an associated <code>Blockchain</code> which allows
 * the application to interface with the environment the app is running.
 * <p>
 * Typically, it includes the transaction and block context, and other blockchain
 * functionality.
 */
public final class Blockchain {

    private Blockchain() {
    }

    //===================
    // Transaction
    //===================

    /**
     * Returns the owner's address, whose state is being accessed.
     * That is, the address of the currently-running DApp.
     *
     * @return an address
     */
    public static Address getAddress() {
        return null;
    }

    /**
     * Returns the callers's address.
     * Note that the caller and the origin may be the same but differ in cross-calls: the origin is the sender
     * of the "first" invocation in the chain while the caller is whoever directly called the current DApp.
     *
     * @return an address
     */
    public static Address getCaller() {
        return null;
    }

    /**
     * Returns the originator's address.
     * Note that the caller and the origin may be the same but differ in cross-calls: the origin is the sender
     * of the "first" invocation in the chain while the caller is whoever directly called the current DApp.
     * Also, the origin never has associated code.
     *
     * @return an address
     */
    public static Address getOrigin() {
        return null;
    }

    /**
     * Returns the energy limit for this current invocation.
     * Note that this is the total limit for the entire invocation, not just what is remaining.
     *
     * @return the max consumable energy
     */
    public static long getEnergyLimit() {
        return 0;
    }

    /**
     * Returns the energy price specified in the transaction.
     *
     * @return energy price.
     */
    public static long getEnergyPrice() {
        return 0;
    }

    /**
     * Returns the value being transferred to this dapp.
     *
     * @return the value in 10^-18 Aion
     */
    public static BigInteger getValue() {
        return BigInteger.ZERO;
    }

    /**
     * Returns the data passed to this dapp.
     *
     * @return an byte array, non-NULL.
     */
    public static byte[] getData() {
        return null;
    }

    //===================
    // Block
    //===================

    /**
     * Returns the block timestamp.
     *
     * @return a timestamp indicates when the block is forged.
     */
    public static long getBlockTimestamp() {
        return 0;
    }

    /**
     * Returns the block number.
     *
     * @return the number of the block, in which the transaction is included
     */
    public static long getBlockNumber() {
        return 0;
    }

    /**
     * Returns the block energy limit.
     *
     * @return the energy cap of the block.
     */
    public static long getBlockEnergyLimit() {
        return 0;
    }

    /**
     * Returns the block coinbase.
     *
     * @return the miner's address of the block.
     */
    public static Address getBlockCoinbase() {
        return null;
    }

    /**
     * Returns the block difficulty.
     *
     * @return the PoW difficulty of the block.
     */
    public static BigInteger getBlockDifficulty() {
        return null;
    }

    //===================
    // Storage
    //===================

    /**
     *  puts the key-value data of an account
     *
     * @param key key of the key-value data pair
     * @param value value of the key-value data pair
     * @throws IllegalArgumentException when the arguments are invalud, e.g. NULL address
     */
    public static void putStorage(byte[] key, byte[] value) throws IllegalArgumentException {
    }

    /**
     * Returns the storage value
     *
     * @param key key of the key-value data pair
     * @return the value in storage associated to the given address and key
     * @throws IllegalArgumentException when the arguments are invalid, e.g. NULL address
     */
    public static byte[] getStorage(byte[] key) throws IllegalArgumentException {
        return null;
    }

    /**
     * Returns the balance of an account.
     *
     * @param address the account address.
     * @return the account balance, or 0 if the account does not exist
     * @throws IllegalArgumentException when the arguments are invalid, e.g. NULL address.
     */
    public static BigInteger getBalance(Address address) throws IllegalArgumentException {
        return BigInteger.ZERO;
    }

    /**
     * Returns the balance of the contract in which this method was invoked.
     *
     * @return the contract balance.
     */
    public static BigInteger getBalanceOfThisContract() {
        return BigInteger.ZERO;
    }

    /**
     * Returns the size of the code, of the given account.
     *
     * @param address the account address.
     * @return the code size in bytes, or 0 if no contract is deployed at that address
     * @throws IllegalArgumentException when the argument is invalid, e.g. NULL address.
     */
    public static int getCodeSize(Address address) throws IllegalArgumentException {
        return 0;
    }

    //===================
    // System
    //===================

    /**
     * Returns the remaining energy, at the moment this method is being called.
     *
     * @return the remaining energy
     */
    public static long getRemainingEnergy() {
        return 0;
    }

    /**
     * Calls another account, whether it's normal account or dapp.
     *
     * In terms of the provided {@code targetAddress}, a call is legitimate only if:
     *   1. The targetAddress has no code (ie. it is not a contract)
     *   2. The targetAddress has code and its code can be executed by the Avm.
     *
     * If neither of these conditions is true then this method will throw an exception.
     *
     * @param targetAddress the account address
     * @param value         the value to transfer
     * @param data          the value to pass
     * @param energyLimit   the max energy the invoked dapp can use.
     * @return the invocation result.
     * @throws IllegalArgumentException when the arguments are invalid, e.g. insufficient balance, NULL address
     * or the targetAddress is a contract that requires a foreign virtual machine in order to be executed.
     */
    public static Result call(Address targetAddress, BigInteger value, byte[] data, long energyLimit) throws IllegalArgumentException {
        return null;
    }

    /**
     * Creates an account.
     *
     * @param value       the value to transfer to the account to be created.
     * @param data        the data, in the format of <code>size_of_code + code + size_of_data + data</code>
     * @param energyLimit the max energy the invoked dapp can use.
     * @return the invocation result.
     * @throws IllegalArgumentException when the arguments are invalid, e.g. insufficient balance or NULL data.
     */
    public static Result create(BigInteger value, byte[] data, long energyLimit) throws IllegalArgumentException {
        return null;
    }

    /**
     * Destroys this dapp and refund all balance to the beneficiary address.
     *
     * @param beneficiary the beneficiary's address
     * @throws IllegalArgumentException when the arguments are invalid, e.g. NULL address.
     */
    public static void selfDestruct(Address beneficiary) throws IllegalArgumentException {
    }

    /**
     * Records a log on blockchain.
     *
     * @param data any arbitrary data, non-NULL
     * @throws IllegalArgumentException when the arguments are invalid, e.g. any are NULL.
     */
    public static void log(byte[] data) throws IllegalArgumentException {
    }

    /**
     * Records a log on blockchain.
     *
     * @param topic1 the 1st topic
     * @param data   any arbitrary data, non-NULL
     * @throws IllegalArgumentException when the arguments are invalid, e.g. any are NULL.
     */
    public static void log(byte[] topic1, byte[] data) throws IllegalArgumentException {
    }

    /**
     * Records a log on blockchain.
     *
     * @param topic1 the 1st topic
     * @param topic2 the 2nd topic
     * @param data   any arbitrary data, non-NULL
     * @throws IllegalArgumentException when the arguments are invalid, e.g. any are NULL.
     */
    public static void log(byte[] topic1, byte[] topic2, byte[] data) throws IllegalArgumentException {
    }

    /**
     * Records a log on blockchain.
     *
     * @param topic1 the 1st topic
     * @param topic2 the 2nd topic
     * @param topic3 the 3rd topic
     * @param data   any arbitrary data, non-NULL
     * @throws IllegalArgumentException when the arguments are invalid, e.g. any are NULL.
     */
    public static void log(byte[] topic1, byte[] topic2, byte[] topic3, byte[] data) throws IllegalArgumentException {
    }

    /**
     * Records a log on blockchain.
     *
     * @param topic1 the 1st topic
     * @param topic2 the 2nd topic
     * @param topic3 the 3rd topic
     * @param topic4 the 4th topic
     * @param data   any arbitrary data, non-NULL
     * @throws IllegalArgumentException when the arguments are invalid, e.g. any are NULL.
     */
    public static void log(byte[] topic1, byte[] topic2, byte[] topic3, byte[] topic4, byte[] data) throws IllegalArgumentException {
    }

    /**
     * Calculates the blake2b digest of the input data.
     *
     * @param data the input data
     * @return the hash digest
     * @throws IllegalArgumentException when the arguments are invalid, e.g. data is NULL.
     */
    public static byte[] blake2b(byte[] data) throws IllegalArgumentException {
        return null;
    }

    /**
     * Calculates the sha256 digest of the input data.
     *
     * @param data the input data
     * @return the hash digest
     * @throws IllegalArgumentException when the arguments are invalid, e.g. data is NULL.
     */
    public static byte[] sha256(byte[] data) throws IllegalArgumentException {
        return null;
    }

    /**
     * Calculates the keccak256 digest of the input data.
     *
     * @param data the input data
     * @return the hash digest
     * @throws IllegalArgumentException when the arguments are invalid, e.g. data is NULL.
     */
    public static byte[] keccak256(byte[] data) throws IllegalArgumentException {
        return null;
    }

    /**
     * Stop the current execution and roll back all state changes.
     * <p>
     * the remaining energy will be refunded.
     */
    public static void revert() {
    }

    /**
     * Stop the current execution and roll back all state changes.
     * <p>
     * the remaining energy will be consumed.
     */
    public static void invalid() {
    }

    /**
     * Checks that the provided condition is true and if it is false, triggers a revert.
     * <p>
     * In other words, if {@code condition == true} this method does nothing, otherwise it is
     * equivalent to calling {@link Blockchain#revert()}.
     *
     * @param condition The condition that is required to be {@code true}.
     */
    public static void require(boolean condition) {
    }

    /**
     * Prints a message, for debugging purpose
     *
     * @param message the message to print
     */
    public static void print(String message) {

    }

    /**
     * Prints a message, for debugging purpose
     *
     * @param message the message to print
     */
    public static void println(String message) {

    }

    /**
     * Verify that the given data is signed by providing the public key and the signed signature.
     *
     * @param data message to be signed
     * @param signature signature of the message
     * @param publicKey public key of the keypair used to sign the message
     * @return result
     * @throws IllegalArgumentException thrown when an input parameter has the wrong size
     */
    public static boolean edVerify(byte[] data, byte[] signature, byte[] publicKey) throws IllegalArgumentException {
        return true;
    }
}
