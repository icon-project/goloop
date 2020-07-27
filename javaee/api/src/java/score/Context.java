/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package score;

import java.math.BigInteger;

/**
 * Every SCORE has an associated <code>Context</code> which allows the application to interface
 * with the environment the SCORE is running.
 * <p>
 * Typically, it includes the transaction and block context, and other blockchain functionality.
 */
public final class Context {

    private Context() {
    }

    //===================
    // Transaction
    //===================

    /**
     * Returns the hash of the transaction.
     *
     * @return the transaction hash
     */
    public static byte[] getTransactionHash() {
        return null;
    }

    /**
     * Returns the transaction index in a block.
     *
     * @return the transaction index
     */
    public static int getTransactionIndex() {
        return 0;
    }

    /**
     * Returns the timestamp of a transaction request.
     *
     * @return the transaction timestamp
     */
    public static long getTransactionTimestamp() {
        return 0L;
    }

    /**
     * Returns the nonce of a transaction request.
     *
     * @return the transaction nonce
     */
    public static BigInteger getTransactionNonce() {
        return BigInteger.ZERO;
    }

    /**
     * Returns the address of the currently-running SCORE.
     *
     * @return an address
     */
    public static Address getAddress() {
        return null;
    }

    /**
     * Returns the caller's address.
     * Note that the caller and the origin may be the same but differ in cross-calls: the origin is the sender
     * of the "first" invocation in the chain while the caller is whoever directly called the current SCORE.
     *
     * @return an address
     */
    public static Address getCaller() {
        return null;
    }

    /**
     * Returns the originator's address.
     * Note that the caller and the origin may be the same but differ in cross-calls: the origin is the sender
     * of the "first" invocation in the chain while the caller is whoever directly called the current SCORE.
     * Also, the origin never has associated code.
     *
     * @return an address
     */
    public static Address getOrigin() {
        return null;
    }

    /**
     * Returns the address of the account who initially deployed the contract.
     *
     * @return an address
     */
    public static Address getOwner() {
        return null;
    }

    /**
     * Returns the value being transferred to this SCORE.
     *
     * @return the value in loop (1 ICX == 10^18 loop)
     */
    public static BigInteger getValue() {
        return BigInteger.ZERO;
    }

    //===================
    // Block
    //===================

    /**
     * Returns the block timestamp.
     *
     * @return the timestamp of the current block in microseconds
     */
    public static long getBlockTimestamp() {
        return 0L;
    }

    /**
     * Returns the block height.
     *
     * @return the height of the block, in which the transaction is included
     */
    public static long getBlockHeight() {
        return 0L;
    }

    //===================
    // Storage
    //===================

    /**
     * Returns the balance of an account.
     *
     * @param address the account address
     * @return the account balance, or 0 if the account does not exist
     * @throws IllegalArgumentException if the address is invalid, e.g. NULL address
     */
    public static BigInteger getBalance(Address address) throws IllegalArgumentException {
        return BigInteger.ZERO;
    }

    //===================
    // System
    //===================

    /**
     * Calls the method of the given account address with the value.
     *
     * @param value         the value in loop to transfer
     * @param targetAddress the account address
     * @param method        method
     * @param params        parameters
     * @return the invocation result
     * @throws IllegalArgumentException if the arguments are invalid, e.g. insufficient balance, NULL address
     * @throws ScoreRevertException if call target reverts the newly created frame
     */
    public static Object call(BigInteger value,
                              Address targetAddress, String method, Object... params) {
        return null;
    }

    /**
     * Calls the method of the account designated by the targetAddress.
     *
     * @param targetAddress the account address
     * @param method        method
     * @param params        parameters
     * @return the invocation result
     * @throws IllegalArgumentException if the arguments are invalid, e.g. insufficient balance, NULL address
     * @throws ScoreRevertException if call target reverts the newly created frame
     */
    public static Object call(Address targetAddress, String method, Object... params) {
        return null;
    }

    /**
     * Transfers the value to the given target address from this SCORE's account.
     *
     * @param targetAddress the account address
     * @param value         the value in loop to transfer
     * @throws IllegalArgumentException if the arguments are invalid, e.g. insufficient balance, NULL address
     */
    public static void transfer(Address targetAddress, BigInteger value) {
    }

    /**
     * Stops the current execution and rolls back all state changes.
     * In case of cross-calls, {@code ScoreRevertException} would be raised to the caller
     * with the given code and message data.
     *
     * @param code an arbitrary user-defined code
     * @param message a message to be delivered to the caller
     */
    public static void revert(int code, String message) {
    }

    /**
     * Stops the current execution and rolls back all state changes.
     *
     * @param code an arbitrary user-defined code
     * @see #revert(int, String)
     */
    public static void revert(int code) {
    }

    /**
     * Stops the current execution and rolls back all state changes.
     *
     * @param message a message
     * @see #revert(int, String)
     */
    public static void revert(String message) {
    }

    /**
     * Stops the current execution and rolls back all state changes.
     * This is equivalent to {@code revert(0)}.
     *
     * @see #revert(int)
     */
    public static void revert() {
    }

    /**
     * Checks that the provided condition is true and if it is false, triggers a revert.
     * <p>
     * In other words, if {@code condition == true}, this method does nothing,
     * otherwise it is equivalent to calling {@link Context#revert()}.
     *
     * @param condition the condition that is required to be {@code true}.
     */
    public static void require(boolean condition) {
    }

    /**
     * Prints a message, for debugging purpose.
     *
     * @param message the message to print
     */
    public static void println(String message) {
    }

    /**
     * Computes the SHA3-256 hash using the input data.
     *
     * @param data the input data to be hashed
     * @return the hashed data in bytes
     */
    public static byte[] sha3_256(byte[] data) throws IllegalArgumentException {
        return null;
    }

    /**
     * Computes the SHA-256 hash using the input data.
     *
     * @param data the input data to be hashed
     * @return the hashed data in bytes
     */
    public static byte[] sha256(byte[] data) throws IllegalArgumentException {
        return null;
    }

    /**
     * Recovers the public key from the message hash and the recoverable signature.
     *
     * @param msgHash the 32 bytes hash data
     * @param signature signature_data(64) + recovery_id(1)
     * @param compressed the type of public key to be returned
     * @return the public key recovered from msgHash and signature
     *         (compressed: 33 bytes key, uncompressed: 65 bytes key)
     */
    public static byte[] recoverKey(byte[] msgHash, byte[] signature, boolean compressed) {
        return null;
    }

    /**
     * Returns the address that is associated with the given public key.
     *
     * @param publicKey a byte array that represents the public key
     * @return the address that is associated with the public key
     */
    public static Address getAddressFromKey(byte[] publicKey) {
        return null;
    }

    //===================
    // Collection DB
    //===================

    /**
     * Returns a new branch DB.
     *
     * @param id DB ID
     * @param leafValueClass class of leaf value. For example, a branch DB of
     *          type {@code BranchDB<BigInteger, DictDB<Address, Boolean>>} has
     *          Boolean.class as its leaf value class.
     * @param <K> key type
     * @param <V> sub-DB type
     * @return new branch DB
     * @see BranchDB
     */
    public static<K, V> BranchDB<K, V> newBranchDB(String id, Class<?> leafValueClass) {
        return null;
    }

    /**
     * Returns a new dictionary DB.
     *
     * @param id DB ID
     * @param valueClass class of {@code V}
     * @param <K> key type
     * @param <V> value type
     * @return new dictionary DB
     * @see DictDB
     */
    public static<K, V> DictDB<K, V> newDictDB(String id, Class<V> valueClass) {
        return null;
    }

    /**
     * Returns a new array DB.
     *
     * @param id DB ID
     * @param valueClass class of {@code E}
     * @param <E> element type
     * @return new array DB
     * @see ArrayDB
     */
    public static<E> ArrayDB<E> newArrayDB(String id, Class<E> valueClass) {
        return null;
    }

    /**
     * Returns a new variable DB.
     *
     * @param id DB ID
     * @param valueClass class of {@code E}
     * @param <E> variable type
     * @return new variable DB
     * @see ArrayDB
     */
    public static<E> VarDB<E> newVarDB(String id, Class<E> valueClass) {
        return null;
    }

    /**
     * Records a log on the blockchain. It is recommended to use
     * {@link score.annotation.EventLog} annotation rather than this method directly.
     *
     * @param indexed indexed data
     * @param data extra data
     */
    public static void logEvent(Object[] indexed, Object[] data) {
    }
}
