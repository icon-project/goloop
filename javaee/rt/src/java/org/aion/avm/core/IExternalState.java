package org.aion.avm.core;

import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Result;

import java.math.BigInteger;

/**
 * An interface into some external component that maintains and can answer state queries pertaining
 * to the blockchain.
 */
public interface IExternalState {
    int OPTION_QUERY = 1;
    int OPTION_TRACE = 2;

    /**
     * Commits the state changes in this IExternalState to its parent IExternalState.
     */
    void commit();

    /**
     * Commits the state changes in this IExternalState to the specified IExternalState.
     *
     * @param externalState The external state to commit to.
     */
    void commitTo(IExternalState externalState);

    /**
     * Returns a new IExternalState that is a child of this IExternalState.
     *
     * @return a new child of this external state.
     */
    public IExternalState newChildExternalState();

    /**
     * Creates a new account for the specified address.
     *
     * @param address The address of the new account.
     */
    void createAccount(Address address);

    /**
     * Returns {@code true} only if the specified address has any state associated with it.
     *
     * @param address The address whose account state is to be queried.
     * @return true only if the address has state.
     */
    boolean hasAccountState(Address address);

    /**
     * Returns the pre-transformed code associated with the specified address.
     *
     * Returns {@code null} if the address has no pre-transformed code.
     *
     * @param address The address whose code is to be returned.
     * @return the pre-transformed code or null.
     */
    byte[] getCode(Address address);

    /**
     * Saves the specified pre-transformed code to the given address.
     *
     * @param address The contract address.
     * @param code The code corresponding to the address.
     */
    void putCode(Address address, byte[] code);

    /**
     * Returns the transformed code associated with the specified address.
     *
     * Returns {@code null} if the address has no transformed code.
     *
     * @param address The address whose code is to be returned.
     * @return the transformed code or null.
     */
    byte[] getTransformedCode(Address address);

    /**
     * Saves the specified transformed code associated with the given address.
     *
     * @param address The contract address.
     * @param code The code corresponding to the address.
     */
    void setTransformedCode(Address address, byte[] code);

    /**
     * Saves the specified serialized bytes of the object graph to the given address.
     *
     * @param address The contract address.
     * @param objectGraph The bytes of the object graph.
     */
    void putObjectGraph(Address address, byte[] objectGraph);

    /**
     * Returns the serialized bytes of the object graph associated with the given address.
     *
     * Returns {@code null} if the address has no object graph.
     *
     * @param address The address whose object graph is to be returned.
     * @return the serialized bytes of the object graph or null.
     */
    byte[] getObjectGraph(Address address);

    /**
     * Saves the specified key-value pairing to the given address.
     *
     * If the specified key already exists as a key-value pairing for the given address, then that
     * pairing will be updated so that its old corresponding value is replaced by the new one.
     *
     * @param address The address.
     * @param key The key.
     * @param value The value.
     */
    void putStorage(Address address, byte[] key, byte[] value);

    /**
     * Removes any key-value pairing corresponding to the specified key for the given address if
     * any such pairing exists.
     *
     * @param address The address.
     * @param key The key.
     */
    void removeStorage(Address address, byte[] key);

    /**
     * Returns the value in the key-value pairing to the specified key for the given address if any
     * such pairing exists.
     *
     * Returns {@code null} otherwise, if no such key corresponds to the address.
     *
     * @param address The address.
     * @param key The key.
     * @return the value or null if there is no such value.
     */
    byte[] getStorage(Address address, byte[] key);

    /**
     * Deletes the specified address and any state corresponding to it, if such an address exists.
     *
     * @param address The address to be deleted.
     */
    void deleteAccount(Address address);

    /**
     * Returns the balance of the specified address.
     *
     * Returns {@link BigInteger#ZERO} if the specified address has no state associated with it.
     *
     * @param address The address whose balance is to be queried.
     * @return the account balance.
     */
    BigInteger getBalance(Address address);

    /**
     * Adds the specified amount of funding to the given address. If amount is positive then the
     * account balance will increase, and if amount is negative then the appropriate amount of funds
     * will be removed from the account.
     *
     * @param address The address whose balance is to be adjusted.
     * @param amount The amount by which to adjust the balance of the account.
     */
    void adjustBalance(Address address, BigInteger amount);

    /**
     * Returns the nonce of the specified address.
     *
     * Returns {@link BigInteger#ZERO} if the specified address has no state associated with it.
     *
     * @param address The address whose nonce is to be queried.
     * @return The account nonce.
     */
    BigInteger getNonce(Address address);

    /**
     * Increments the nonce of the specified address by one.
     *
     * If the specified address has no state associated with it, it will now have state. Namely, it
     * will have a nonce of one.
     *
     * @param address The address whose nonce is to be incremented.
     */
    void incrementNonce(Address address);

    /**
     * Refunds the given address by the specified refund amount.
     *
     * This method is equivalent to calling {@code adjustBalance(address, refund)}.
     *
     * This method is only ever invoked when refunding the account at the end of a transaction.
     *
     * @param address The address whose balance is to be updated.
     * @param refund The amount by which to increase the account balance.
     */
    void refundAccount(Address address, BigInteger refund);

    /**
     * Returns the hash of the block whose height is the specified number.
     *
     * Returns {@code null} if the specified block height does not exist and therefore no block hash
     * exists.
     *
     * @param blockHeight The block height whose hash is to be returned.
     * @return the hash of the specified block or null.
     */
    byte[] getBlockHashByHeight(long blockHeight);

    /**
     * Returns {@code true} only if the given address has a nonce equal to the specified nonce.
     *
     * Returns {@code false} otherwise.
     *
     * @param address The address whose nonce is to be tested.
     * @param nonce The nonce to check for.
     * @return whether the account has the given nonce.
     */
    boolean accountNonceEquals(Address address, BigInteger nonce);

    /**
     * Returns {@code true} only if the balance of the given address is greater than or equal to
     * the specified amount.
     *
     * Returns {@code false} otherwise.
     *
     * @param address The address whose balance is to be tested.
     * @param amount The amount to check for.
     * @return whether the account has a balance greater than or equal to amount.
     */
    boolean accountBalanceIsAtLeast(Address address, BigInteger amount);

    /**
     * Returns {@code true} only if the specified energy limit is a valid energy limit that can be
     * used by a contract create transaction.
     *
     * Returns {@code false} otherwise.
     *
     * @param limit The energy limit to test.
     * @return whether the energy limit is valid for a transaction create.
     */
    boolean isValidEnergyLimitForCreate(long limit);

    /**
     * Returns {@code true} only if the specified energy limit is a valid energy limit that can be
     * used by a contract call transaction.
     *
     * Returns {@code false} otherwise.
     *
     * @param limit The energy limit to test.
     * @return whether the energy limit is valid for a transaction call.
     */
    boolean isValidEnergyLimitForNonCreate(long limit);

    /**
     * Returns {@code true} only if the specified address is safe for the Avm to interact with.
     * Returns {@code false} otherwise.
     *
     * An address is unsafe only if it is a non-java contract address (ie. a precompiled contract or
     * a solidity contract).
     *
     * @param address The address to be tested.
     * @return whether or not the address is Avm-safe.
     */
    boolean destinationAddressIsSafeForThisVM(Address address);

    /**
     * Returns the block height of the current block.
     *
     * @return the current block height.
     */
    long getBlockHeight();

    /**
     * Returns the timestamp of the current block.
     *
     * @return the current block timestamp.
     */
    long getBlockTimestamp();

    /**
     * Returns the address of the contract owner
     *
     * @return the owner address
     */
    Address getOwner();

    /**
     * Emits events log
     */
    void log(byte[][] indexed, byte[][]data);

    /**
     * Calls external method of target contract.
     *
     * @param method
     * @param params
     * @return
     * @throws IllegalArgumentException
     */
    Result call(Address address,
                       String method,
                       Object[] params,
                       BigInteger value,
                       int stepLimit);

    int getOption();

    default boolean isQuery() {
        return (getOption() & OPTION_QUERY) != 0;
    }

    default boolean isTrace() {
        return (getOption() & OPTION_TRACE) != 0;
    }
}
