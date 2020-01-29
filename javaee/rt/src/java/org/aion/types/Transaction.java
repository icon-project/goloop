package org.aion.types;

import foundation.icon.ee.types.Address;

import java.math.BigInteger;
import java.util.Arrays;
import java.util.Objects;

public final class Transaction {
    public final Address senderAddress;
    public final Address destinationAddress;
    private final byte[] transactionHash;
    public final int transactionIndex;
    public final long transactionTimestamp;
    public final BigInteger value;
    public final BigInteger nonce;
    public final long energyLimit;
    public final boolean isCreate;
    public final String method;
    private final Object[] params;

    private Transaction(Address senderAddress,
                        Address destinationAddress,
                        byte[] transactionHash,
                        int transactionIndex,
                        long transactionTimestamp,
                        BigInteger value,
                        BigInteger nonce,
                        long energyLimit,
                        String method,
                        Object[] params,
                        boolean isCreate) {
        if (null == senderAddress && transactionHash != null) {
            throw new NullPointerException("No sender");
        }
        if (null == transactionHash && senderAddress != null) {
            throw new NullPointerException("No transaction hash");
        }
        if (null == value) {
            throw new NullPointerException("No value");
        }
        if (null == nonce) {
            throw new NullPointerException("No nonce");
        }
        if (value.compareTo(BigInteger.ZERO) < 0) {
            throw new IllegalArgumentException("Negative value");
        }
        if (nonce.compareTo(BigInteger.ZERO) < 0) {
            throw new IllegalArgumentException("Negative nonce");
        }
        if (energyLimit < 0) {
            throw new IllegalArgumentException("Negative energy limit");
        }
        if (null == method && !isCreate) {
            throw new NullPointerException("Null method for call transaction");
        }
        if (null == params) {
            throw new NullPointerException("Null params");
        }

        this.senderAddress = senderAddress;
        this.destinationAddress = destinationAddress;
        if (transactionHash != null) {
            this.transactionHash = new byte[transactionHash.length];
            System.arraycopy(transactionHash, 0, this.transactionHash, 0, transactionHash.length);
        } else {
            this.transactionHash = null;
        }
        this.transactionIndex = transactionIndex;
        this.transactionTimestamp = transactionTimestamp;
        this.value = value;
        this.nonce = nonce;
        this.energyLimit = energyLimit;
        this.isCreate = isCreate;
        this.method = method;
        // take ownership of params
        this.params = params;
    }

    /**
     * Creates a new transaction.
     *
     * @param sender The sender of the transaction.
     * @param destination The contract to be called or account to have value transferred to.
     * @param txHash The hash of the transaction.
     * @param txIndex The transaction index in a block.
     * @param value The amount of value to be transferred from the sender to the destination.
     * @param nonce The nonce of the sender.
     * @param method The name of method.
     * @param params The list of parameters for the method.
     * @param energyLimit The maximum amount of energy to be used by the transaction.
     * @param isCreate True if this transaction is for contract creation.
     * @return a Transaction object
     */
    public static Transaction newTransaction(Address sender, Address destination,
                                             byte[] txHash, int txIndex, long txTimestamp, BigInteger value, BigInteger nonce,
                                             String method, Object[] params, long energyLimit, boolean isCreate) {
        return new Transaction(sender, destination, txHash, txIndex, txTimestamp, value, nonce,
                               energyLimit, method, params, isCreate);
    }

    public byte[] copyOfTransactionHash() {
        if (transactionHash == null) {
            return null;
        }
        byte[] transactionHashCopy = new byte[transactionHash.length];
        System.arraycopy(transactionHash, 0, transactionHashCopy, 0, transactionHash.length);
        return transactionHashCopy;
    }

    public Object[] getParams() {
        Object[] paramsCopy = new Object[params.length];
        System.arraycopy(params, 0, paramsCopy, 0, params.length);
        return paramsCopy;
    }

    @Override
    public boolean equals(Object obj) {
        if (this == obj) {
            return true;
        } else if (!(obj instanceof Transaction)) {
            return false;
        } else {
            Transaction otherObject = (Transaction) obj;
            // compare method and params on transactionData field removal
            return Objects.equals(this.senderAddress, otherObject.senderAddress)
                    && Objects.equals(this.destinationAddress, otherObject.destinationAddress)
                    && this.transactionIndex == otherObject.transactionIndex
                    && this.value.equals(otherObject.value)
                    && this.nonce.equals(otherObject.nonce)
                    && this.energyLimit == otherObject.energyLimit
                    && this.isCreate == otherObject.isCreate
                    && Arrays.equals(this.transactionHash, otherObject.transactionHash);
        }
    }

    @Override
    public String toString() {
        return "TransactionData ["
            + "hash="
            + Arrays.toString(transactionHash)
            + ", index="
            + transactionIndex
            + ", nonce="
            + nonce
            + ", destinationAddress="
            + destinationAddress
            + ", value="
            + value
            + ", energyLimit="
            + this.energyLimit
            + "]";
    }

    @Override
    public int hashCode() {
        int result = Objects.hash(senderAddress, destinationAddress, transactionIndex, value, nonce, energyLimit, isCreate);
        if (transactionHash != null) {
            result = 31 * result + Arrays.hashCode(transactionHash);
        }
        return result;
    }
}
