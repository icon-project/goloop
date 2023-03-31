package i;

import a.ByteArray;
import p.score.Address;
import p.score.AnyDB;
import p.score.ByteArrayObjectWriter;
import p.score.ObjectReader;
import s.java.lang.Class;
import s.java.lang.String;
import s.java.math.BigInteger;

/**
 * Interface to the blockchain runtime.
 */
public interface IBlockchainRuntime {
    //================
    // Transaction
    //================

    /**
     * Returns the transaction hash of the origin transaction.
     */
    ByteArray avm_getTransactionHash();

    /**
     * Returns the transaction index in a block.
     */
    int avm_getTransactionIndex();

    /**
     * Returns the timestamp of a transaction request.
     */
    long avm_getTransactionTimestamp();

    /**
     * Returns the nonce of a transaction request.
     */
    BigInteger avm_getTransactionNonce();

    /**
     * Returns the address of the currently-running SCORE.
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
     * Returns the address of the account who deployed the contract.
     */
    Address avm_getOwner();

    /**
     * Returns the value being transferred along the transaction.
     */
    BigInteger avm_getValue();

    //================
    // Block
    //================

    /**
     * Block timestamp.
     *
     * @return The time of the current block, as seconds since the Epoch.
     */
    long avm_getBlockTimestamp();

    /**
     * Block height.
     *
     * @return The height of the current block.
     */
    long avm_getBlockHeight();

    //================
    // Storage
    //================

    /**
     * Returns the balance of an account.
     *
     * @param address account address
     * @return the balance of the account
     */
    BigInteger avm_getBalance(Address address) throws IllegalArgumentException;

    //================
    // System
    //================

    /**
     * Calls the contract denoted by the targetAddress.  Returns the response of the contract.
     *
     * @param value         The value to transfer
     * @param targetAddress The address of the contract to call.
     * @param method        method
     * @param params        parameters
     * @return The response of executing the contract.
     */
    Object avm_call(Class<?> cls, BigInteger value,
            Address targetAddress, String method, IObjectArray params);

    /**
     * Deploys a SCORE with the given byte streams to the target address.
     *
     * @param target the SCORE address that is to be updated
     * @param content the byte streams of the SCORE
     * @param params parameters
     * @return the SCORE address if the deployment was successful
     */
    Address avm_deploy(Address target, ByteArray content, IObjectArray params);

    /**
     * Stop the current execution, rollback any state changes, and refund the remaining energy to caller.
     */
    void avm_revert(int code, String message);

    void avm_revert(int code);

    /**
     * Requires that condition is true, otherwise triggers a revert.
     */
    void avm_require(boolean condition, String message);

    void avm_require(boolean condition);

    /**
     * Prints a message to console for debugging purpose
     */
    void avm_println(String message);

    ByteArray avm_hash(String alg, ByteArray msg);
    ByteArray avm_altBN128(String operation, ByteArray input);
    boolean avm_verifySignature(String alg, ByteArray msg, ByteArray sig,
            ByteArray pubKey);
    ByteArray avm_recoverKey(String alg, ByteArray msg, ByteArray sig,
            boolean compressed);
    ByteArray avm_aggregate(String type, ByteArray prevAgg, ByteArray values);

    /**
     * Returns the address that is associated with the given public key
     */
    Address avm_getAddressFromKey(ByteArray publicKey);

    /**
     * Returns the current fee sharing proportion of the SCORE.
     */
    int avm_getFeeSharingProportion();

    /**
     * Sets the proportion of transaction fees that the SCORE will pay.
     */
    void avm_setFeeSharingProportion(int proportion);

    /**
     * Returns a new AnyDB instance
     */
    AnyDB avm_newAnyDB(String id, Class<?> vc);

    /**
     * Emits event logs
     */
    void avm_logEvent(IObjectArray indexed, IObjectArray data);

    ObjectReader avm_newByteArrayObjectReader(String codec, ByteArray byteArray);

    ByteArrayObjectWriter avm_newByteArrayObjectWriter(String codec);
}
