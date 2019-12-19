package org.aion.data;


/**
 * The abstract interface over the top-level of the account storage abstraction.
 */
public interface IDataStore {
    /**
     * Opens an existing account identified by address, returning null if it doesn't already exist.
     * 
     * @param address The address of the account.
     * @return The account, or null if no such account exists.
     */
    public IAccountStore openAccount(byte[] address);

    /**
     * Creates a new account with the given address identification.
     * It is invalid to create the same account multiple times.
     * 
     * @param address The address to use for the new account.
     * @return The account.
     */
    public IAccountStore createAccount(byte[] address);

    /**
     * Deletes the account identified by the given address.  Does nothing if the account doesn't exist.
     * 
     * @param address The address of the account.
     */
    public void deleteAccount(byte[] address);
}
