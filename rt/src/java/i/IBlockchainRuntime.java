package i;

import p.avm.Address;
import p.avm.Result;
import a.ByteArray;
import s.java.math.BigInteger;
import s.java.lang.String;


/**
 * Represents the hub of AVM runtime.
 */
public interface IBlockchainRuntime {
    //================
    // transaction
    //================

    /**
     * Returns the owner's address, whose state is being accessed.
     */
    Address avm_getAddress();

    /**
     * Returns the caller's address.
     */
    Address avm_getCaller();

    /**
     * Returns the originator's address.
     */
    Address avm_getOrigin();

    /**
     * Returns the energy limit.
     */
    long avm_getEnergyLimit();

    /**
     * Returns the energy price.
     */
    long avm_getEnergyPrice();

    /**
     * Returns the value being transferred along the transaction.
     */
    BigInteger avm_getValue();

    /**
     * Returns the transaction data.
     */
    ByteArray avm_getData();


    //================
    // block
    //================

    /**
     * Block timestamp.
     *
     * @return The time of the current block, as seconds since the Epoch.
     */
    long avm_getBlockTimestamp();

    /**
     * Block number.
     *
     * @return The number of the current block.
     */
    long avm_getBlockNumber();

    /**
     * Block energy limit
     *
     * @return The block energy limit
     */
    long avm_getBlockEnergyLimit();

    /**
     * Block coinbase address
     *
     * @return the miner address of the block
     */
    Address avm_getBlockCoinbase();

    /**
     * Block difficulty
     *
     * @return the difficulty of the block.
     */
    BigInteger avm_getBlockDifficulty();

    //================
    // State
    //================

    /**
     * Puts the key-value data of an account.
     *
     * @param key key of the key-value data pair
     * @param value value of the key-value data pair
     */
     void avm_putStorage(ByteArray key, ByteArray value, boolean requiresRefund) throws IllegalArgumentException;

    /**
     * Returns the storage value.
     *
     * @param key of the key-value pair
     * @return the value in storage associated to the given key
     */
    ByteArray avm_getStorage(ByteArray key) throws IllegalArgumentException;

    /**
     * Returns the balance of an account.
     *
     * @param address account address
     * @return the balance of the account
     */
    BigInteger avm_getBalance(Address address) throws IllegalArgumentException;

    /**
     * Returns the balance of the contract in which this method is invoked.
     *
     * @return the balance of the contract.
     */
    BigInteger avm_getBalanceOfThisContract();

    /**
     * Returns the code size of an account.
     *
     * @param address account address
     * @return the code size of the account
     */
    int avm_getCodeSize(Address address) throws IllegalArgumentException;

    //================
    // System
    //================

    /**
     * Checks the current remaining energy.
     *
     * @return the remaining energy.
     */
    long avm_getRemainingEnergy();

    /**
     * Calls the contract denoted by the targetAddress, sending payload data and energyLimit for the invocation.  Returns the response of the contract.
     * NOTE:  This is likely to change as we work out the details of the ABI and cross-call semantics but exists to handle expectations of ported Solidity applications.
     *
     * @param targetAddress The address of the contract to call.
     * @param value         The value to transfer
     * @param data          The data payload to send to that contract.
     * @param energyLimit   The energy to send that contract.
     * @return The response of executing the contract.
     */
    Result avm_call(Address targetAddress, BigInteger value, ByteArray data, long energyLimit) throws IllegalArgumentException;

    Result avm_create(BigInteger value, ByteArray data, long energyLimit) throws IllegalArgumentException;

    /**
     * Destructs this Dapp and refund all balance to the beneficiary.
     *
     * @param beneficiary
     */
    void avm_selfDestruct(Address beneficiary) throws IllegalArgumentException;

    /**
     * Logs information for offline analysis or external listening.
     *
     * @param data arbitrary unstructured data.
     */
    void avm_log(ByteArray data) throws IllegalArgumentException;

    void avm_log(ByteArray topic1, ByteArray data) throws IllegalArgumentException;

    void avm_log(ByteArray topic1, ByteArray topic2, ByteArray data) throws IllegalArgumentException;

    void avm_log(ByteArray topic1, ByteArray topic2, ByteArray topic3, ByteArray data) throws IllegalArgumentException;

    void avm_log(ByteArray topic1, ByteArray topic2, ByteArray topic3, ByteArray topic4, ByteArray data) throws IllegalArgumentException;

    /**
     * Computes the Blake2b digest of the given data.
     *
     * @param data The data to hash.
     * @return The 32-byte digest.
     */
    ByteArray avm_blake2b(ByteArray data) throws IllegalArgumentException;

    /**
     * Computes the sha256 digest of the given data.
     *
     * @param data The data to hash.
     * @return The 32-byte digest.
     */
    ByteArray avm_sha256(ByteArray data);

    /**
     * Computes the keccak256 digest of the given data.
     *
     * @param data The data to hash.
     * @return The 32-byte digest.
     */
    ByteArray avm_keccak256(ByteArray data);

    /**
     * Stop the current execution, rollback any state changes, and refund the remaining energy to caller.
     */
    void avm_revert();

    /**
     * Stop the current execution, rollback any state changes, and consume all remaining energy.
     */
    void avm_invalid();

    /**
     * Requires that condition is true, otherwise triggers a revert.
     */
    void avm_require(boolean condition);

    /**
     * Prints a message to console for debugging purpose
     *
     * @param message
     */
    void avm_print(String message);

    /**
     * Prints a message to console for debugging purpose
     *
     * @param message
     */
    void avm_println(String message);

    /**
     * Verify that the given data is signed by providing the public key and the signed signature.
     *
     * @param data byte array representation of the data
     * @param signature of the signed data
     * @param publicKey of the signed data
     * @return result
     */
    boolean avm_edVerify(ByteArray data, ByteArray signature, ByteArray publicKey) throws IllegalArgumentException;
}
