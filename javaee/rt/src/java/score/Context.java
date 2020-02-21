/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package score;

import java.math.BigInteger;

/**
 * Every SCORE has an associated <code>Context</code> which allows the application to interface
 * with the environment the app is running.
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
     * Returns the address of the account who deployed the contract.
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
     * @return a timestamp indicates when the block is forged.
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
     * @param address the account address.
     * @return the account balance, or 0 if the account does not exist
     * @throws IllegalArgumentException when the arguments are invalid, e.g. NULL address.
     */
    public static BigInteger getBalance(Address address) throws IllegalArgumentException {
        return BigInteger.ZERO;
    }

    //===================
    // System
    //===================

    /**
     * Calls another account, whether it's normal account or SCORE.
     *
     * @param value         the value to transfer
     * @param stepLimit     step limit
     * @param targetAddress the account address
     * @param method        method
     * @param params        parameters
     * @return the invocation result.
     * @throws IllegalArgumentException when the arguments are invalid, e.g. insufficient balance, NULL address
     * @throws ScoreRevertException when call target reverts the newly created frame
     */
    public static Object call(BigInteger value, BigInteger stepLimit,
                              Address targetAddress, String method, Object... params) {
        return null;
    }

    public static Object call(BigInteger value,
                              Address targetAddress, String method, Object... params) {
        return null;
    }

    public static Object call(Address targetAddress, String method, Object... params) {
        return null;
    }

    /**
     * Stop the current execution and roll back all state changes.
     * <p>
     * the remaining energy will be refunded.
     */
    public static void revert(int code, String message) {
    }

    public static void revert(int code) {
    }

    public static void revert(String message) {
    }

    public static void revert() {
    }

    /**
     * Checks that the provided condition is true and if it is false, triggers a revert.
     * <p>
     * In other words, if {@code condition == true} this method does nothing, otherwise it is
     * equivalent to calling {@link Context#revert()}.
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
    public static void println(String message) {
    }

    //===================
    // Collection DB
    //===================

    public static<K, V> NestingDictDB<K, V> newNestingDictDB(String id, Class<?> leafValueClass) {
        return null;
    }

    public static<K, V> DictDB<K, V> newDictDB(String id, Class<V> valueClass) {
        return null;
    }

    public static<E> ArrayDB<E> newArrayDB(String id, Class<E> valueClass) {
        return null;
    }

    public static<E> VarDB<E> newVarDB(String id, Class<E> valueClass) {
        return null;
    }

    /**
     * Records a log on blockchain.
     *
     * @param indexed indexed data
     * @param data extra data
     */
    public static void log(Value[] indexed, Value[] data) {
    }
}
