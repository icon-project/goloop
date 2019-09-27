package org.aion.avm.core;

import org.aion.types.AionAddress;
import org.aion.types.Transaction;


/**
 * Defines the abstract behavioural requirements which must be provided by a consumer of the AVM.
 * These are provided from outside since they are either specific to a blockchain, not the AVM, or are generic
 * hashing or cryptographic utilities which the AVM need not duplicate.
 */
public interface IExternalCapabilities {
    /**
     * Computes the SHA-256 hash of the given data.
     * 
     * @param data The data to hash.
     * @return The SHA-256 hash of this data.
     */
    byte[] sha256(byte[] data);

    /**
     * Computes the BLAKE2B hash of the given data.
     * 
     * @param data The data to hash.
     * @return The BLAKE2B hash of this data.
     */
    byte[] blake2b(byte[] data);

    /**
     * Computes the KECCAK256 hash of the given data.
     * 
     * @param data The data to hash.
     * @return The KECCAK256 hash of this data.
     */
    byte[] keccak256(byte[] data);

    /**
     * Verifies that the given data, when signed by the private key counterpart to the given public key, produces the given signature.
     * 
     * @param data The data which was signed.
     * @param signature The signature produced.
     * @param publicKey The public key corresponding to the private key used to produce signature.
     * @return True if this public key verifies the signature.
     */
    boolean verifyEdDSA(byte[] data, byte[] signature, byte[] publicKey);

    /**
     * Determines the new contract address of the given transaction.
     * Note that this call must have NO SIDE-EFFECTS as it may be called multiple times on the same transaction.
     *
     * @param tx The transaction performing the creation.
     * @return The address of the new contract this transaction would create.
     */
    AionAddress generateContractAddress(Transaction tx);
}
