package org.aion.data;

import java.math.BigInteger;
import java.util.Map;

import org.aion.avm.core.util.ByteArrayWrapper;


/**
 * The abstract interface over a single account residing in an IDataStore instance.
 */
public interface IAccountStore {

    /**
     * @return The deploy code stored for this account.
     */
    public byte[] getCode();

    /**
     * @param code The code to store for this account.
     */
    public void setCode(byte[] code);

    /**
     * @return The transformed code stored for this account.
     */
    public byte[] getTransformedCode();

    /**
     * @param code The transformed code to store for this account.
     */
    public void setTransformedCode(byte[] code);
    /**
     * @return The account balance.
     */
    public BigInteger getBalance();

    /**
     * @param balance The new account balance.
     */
    public void setBalance(BigInteger balance);

    /**
     * @return The account nonce.
     */
    public long getNonce();

    /**
     * @param nonce The new account nonce.
     */
    public void setNonce(long nonce);

    /**
     * Reads the application key-value store.
     * 
     * @param key The key to read.
     * @return The value for the key (null if the key is not found).
     */
    public byte[] getData(byte[] key);

    /**
     * Writes the application key-value store.
     * 
     * @param key The key to read.
     * @param value The value to store for the key.
     */
    public void setData(byte[] key, byte[] value);

    /**
     * Removes the application key-value store.
     *
     * @param key The key to remove.
     */
    public void removeData(byte[] key);

    /**
     * Used only for testing and will be removed in the future.
     * 
     * @return A map of the entries in the account's application key-value store.
     */
    public Map<ByteArrayWrapper, byte[]> getStorageEntries();

    /**
     * Writes the serialized application object graph.
     * 
     * @param data The raw serialized graph to write.
     */
    public void setObjectGraph(byte[] data);

    /**
     * Reads the serialized application object graph.
     * 
     * @return The raw serialized graph read.
     */
    public byte[] getObjectGraph();
}
