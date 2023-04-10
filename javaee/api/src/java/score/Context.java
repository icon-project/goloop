/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package score;

import java.math.BigInteger;

/**
 * Every SCORE has an associated <code>Context</code> which allows the application to interface
 * with the environment the SCORE is running.
 *
 * <p>Typically, it includes the transaction and block context, and other blockchain functionality.
 *
 * <p>Unless otherwise noted, passing a null argument to a method in this class will cause a {@code NullPointerException} to be thrown.
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
     * @param <T>           return type
     * @param cls           class of return type
     * @param value         the value in loop to transfer
     * @param targetAddress the account address
     * @param method        method
     * @param params        parameters
     * @return the invocation result
     * @throws IllegalArgumentException if the arguments are invalid, e.g. insufficient balance, etc.
     * @throws RevertedException if call target reverts the newly created frame
     * @throws UserRevertedException if call target reverts the newly created frame by calling {@link Context#revert}
     * @throws ArithmeticException if returned value is out of range
     * @see score.annotation.Keep
     */
    public static<T> T call(Class<T> cls, BigInteger value,
            Address targetAddress, String method, Object... params) {
        return null;
    }

    /**
     * Calls the method of the given account address with the value.
     *
     * @param value         the value in loop to transfer
     * @param targetAddress the account address
     * @param method        method
     * @param params        parameters
     * @return the invocation result
     * @throws IllegalArgumentException if the arguments are invalid, e.g. insufficient balance, etc.
     * @throws RevertedException if call target reverts the newly created frame
     * @throws UserRevertedException if call target reverts the newly created frame by calling {@link Context#revert}
     * @throws ArithmeticException if returned value is out of range
     * @see score.annotation.Keep
     */
    public static Object call(BigInteger value,
                              Address targetAddress, String method, Object... params) {
        return null;
    }

    /**
     * Calls the method of the account designated by the targetAddress.
     *
     * @param <T>           return type
     * @param cls           class of return type
     * @param targetAddress the account address
     * @param method        method
     * @param params        parameters
     * @return the invocation result
     * @throws IllegalArgumentException if the arguments are invalid, e.g. insufficient balance, etc.
     * @throws RevertedException if call target reverts the newly created frame
     * @throws UserRevertedException if call target reverts the newly created frame by calling {@link Context#revert}
     * @throws ArithmeticException if returned value is out of range
     * @see score.annotation.Keep
     */
    public static<T> T call(Class<T> cls, Address targetAddress, String method,
            Object... params) {
        return null;
    }

    /**
     * Calls the method of the account designated by the targetAddress.
     *
     * @param targetAddress the account address
     * @param method        method
     * @param params        parameters
     * @return the invocation result
     * @throws IllegalArgumentException if the arguments are invalid, e.g. insufficient balance, etc.
     * @throws RevertedException if call target reverts the newly created frame
     * @throws UserRevertedException if call target reverts the newly created frame by calling {@link Context#revert}
     * @throws ArithmeticException if returned value is out of range
     * @see score.annotation.Keep
     */
    public static Object call(Address targetAddress, String method, Object... params) {
        return null;
    }

    /**
     * Transfers the value to the given target address from this SCORE's account.
     *
     * @param targetAddress the account address
     * @param value         the value in loop to transfer
     * @throws IllegalArgumentException if the arguments are invalid, e.g. insufficient balance, etc.
     */
    public static void transfer(Address targetAddress, BigInteger value) {
    }

    /**
     * Deploys a SCORE with the given byte streams.
     *
     * @param content the byte streams of the SCORE
     * @param params parameters
     * @return the newly created SCORE address
     * @throws IllegalArgumentException if the arguments are invalid, e.g. corrupted content, etc.
     * @see score.annotation.Keep
     */
    public static Address deploy(byte[] content, Object... params) {
        return null;
    }

    /**
     * Deploys a SCORE with the given byte streams to the target address.
     *
     * @param targetAddress the SCORE address that is to be updated
     * @param content the byte streams of the SCORE
     * @param params parameters
     * @return the target SCORE address
     * @throws IllegalArgumentException if the arguments are invalid, e.g. corrupted content, etc.
     * @see score.annotation.Keep
     */
    public static Address deploy(Address targetAddress, byte[] content, Object... params) {
        return null;
    }

    /**
     * Stops the current execution and rolls back all state changes.
     * In case of cross-calls, {@link UserRevertedException} would be raised to the caller
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
     * otherwise it is equivalent to calling {@link Context#revert(String)}.
     *
     * @param condition the condition that is required to be {@code true}.
     * @param message a message
     */
    public static void require(boolean condition, String message) {
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
     * Returns hash value of the given message.
     * @param alg hash algorithm. One of sha-256, sha3-256, keccak-256, xxhash-128,
     *            blake2b-128 and blake2b-256.
     * @param msg message
     * @return hash value
     * @throws IllegalArgumentException if the algorithm is unsupported.
     */
    public static byte[] hash(String alg, byte[] msg) {
        return null;
    }

    /**
     * Returns {@code true} if the given signature for the given message by
     * the given public key is correct.
     * @param alg signature algorithm. One of ed25519, ecdsa-secp256k1 and
     *            bls12-381-g2
     * @param msg message
     * @param sig signature
     * @param pubKey public key
     * @return {@code true} if the given signature for the given message by
     * the given public key is correct.
     * @throws IllegalArgumentException if the algorithm is unsupported.
     */
    public static boolean verifySignature(String alg, byte[] msg, byte[] sig, byte[] pubKey) {
        return false;
    }

    /**
     * Recovers the public key from the message and the recoverable signature.
     * @param alg signature algorithm. ecdsa-secp256k1 is supported.
     * @param msg message
     * @param sig signature
     * @param compressed the type of public key to be returned
     * @return the public key recovered from message and signature
     * @throws IllegalArgumentException if the algorithm is unsupported.
     */
    public static byte[] recoverKey(String alg, byte[] msg, byte[] sig, boolean compressed) {
        return null;
    }

    /**
     * Aggregates cryptographic values. This method can be used to aggregate
     * public keys or signatures.
     * @param type value type. bls12-381-g1 is supported.
     * @param prevAgg previous aggregation. null if there is no previous
     *                aggregation.
     * @param values concatenated values to be aggregated.
     * @return aggregation of previous aggregation and values.
     */
    public static byte[] aggregate(String type, byte[] prevAgg, byte[] values) {
        return null;
    }


    /**
     * Returns result of point addition as bigendian integers:
     *   bls12-381-g1: (x, y) 96 bytes or (flag | x) 48 bytes for compressed
     *   bls12-381-g2: (x_u * u + x, y_u * u + y) 192 bytes or (flag | x_u * u + x) 96 bytes for compressed
     * @param curve bls12-381-g1, bls12-381-g2
     * @param data set of points each encoded as 96 bytes (or 48 bytes for compressed) bigendian integers 
     * @param compressed flag to represent compressed point
     * @return binary representation of point addition result
     * @throws IllegalArgumentException if the arguments are invalid
     */
    public static byte[] ecAdd(String curve, byte[] data, boolean compressed) {
        return null;
    }

    /**
     * Returns result of scalar multiplication as bigendian integers:
     *   bls12-381-g1: (x, y) 96 bytes or (flag | x) 48 bytes for compressed
     *   bls12-381-g2: (x_u * u + x, y_u * u + y) 192 bytes or (flag | x_u * u + x) 96 bytes for compressed
     * @param curve bls12-381-g1, bls12-381-g2
     * @param data set of points each encoded as 96 bytes (or 48 bytes for compressed) bigendian integers 
     * @param scalar 32 bytes scalar
     * @param compressed flag to represent compressed point
     * @return binary representation of scalar multiplication result
     * @throws IllegalArgumentException if the arguments are invalid
     */
    public static byte[] ecScalarMul(String curve, byte[] scalar, byte[] data, boolean compressed) {
        return null;
    }

    /**
     * Returns {@code true} if log_G1(a1) * log_G2(a2) + ... + log_G1(z1) + log_G2(z2) = 0
     * @param curve bls12-381
     * @param data set of alternating 
     *             G1 ((x, y) 96 bytes or (flag | x) 48 bytes for compressed bigendian integers) and 
     *             G2 points ((x_u * u + x, y_u * u + y) 192 bytes or (flag | x_u * u + x) 96 bytes for compressed bigendian integers) 
     * @param compressed flag to represent compressed points
     * @return boolean representing pairing check result
     * @throws IllegalArgumentException if the arguments are invalid
     */
    public static boolean ecPairingCheck(String curve, byte[] data, boolean compressed) {
        return false;
    }

    /**
     * Returns the address that is associated with the given public key.
     *
     * @param pubKey a byte array that represents the public key
     * @return the address that is associated with the public key
     */
    public static Address getAddressFromKey(byte[] pubKey) {
        return null;
    }

    //===================
    // Fee Sharing
    //===================

    /**
     * Returns the current fee sharing proportion of the SCORE.
     * 100 means the SCORE will pay 100% of transaction fees on behalf of the transaction sender.
     *
     * @return the current fee sharing proportion that the SCORE will pay (0 to 100)
     */
    public static int getFeeSharingProportion() {
        return 0;
    }

    /**
     * Sets the proportion of transaction fees that the SCORE will pay.
     * {@code proportion} should be between 0 and 100.
     * If this method is invoked multiple times, the last proportion value will be used.
     *
     * @param proportion the desired proportion of transaction fees that the SCORE will pay
     * @throws IllegalArgumentException if the proportion is not between 0 and 100
     */
    public static void setFeeSharingProportion(int proportion) {
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

    /**
     * Returns a new object reader reading from a byte array.
     * @param codec codec, currently "RLP" and "RLPn" are supported.
     * @param byteArray byte array.
     * @return object reader.
     * @throws IllegalArgumentException if the codec is unsupported.
     */
    public static ObjectReader newByteArrayObjectReader(String codec,
            byte[] byteArray) {
        return null;
    }

    /**
     * Returns a new object writer writing to a byte array.
     * @param codec codec, currently "RLP" and "RLPn" are supported.
     * @return byte array object writer
     * @throws IllegalArgumentException if the codec is unsupported.
     */
    public static ByteArrayObjectWriter newByteArrayObjectWriter(
            String codec) {
        return null;
    }
}
