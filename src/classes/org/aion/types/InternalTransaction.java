package org.aion.types;

import java.math.BigInteger;
import java.util.Arrays;

/**
 * An internal transaction is a transaction that is generated as a result of executing contract code
 * (as opposed to a regular -- external -- transaction, which is created externally and sent to the
 * blockchain).
 *
 * An internal transaction is immutable.
 */
public final class InternalTransaction {
    private final byte[] data;
    public final AionAddress sender;
    public final AionAddress destination;
    public final BigInteger senderNonce;
    public final BigInteger value;
    public final long energyLimit;
    public final long energyPrice;
    public final boolean isCreate;
    public final boolean isRejected;

    public enum RejectedStatus { REJECTED, NOT_REJECTED }

    private InternalTransaction(RejectedStatus status, AionAddress sender, AionAddress destination, BigInteger senderNonce, BigInteger value, byte[] data, long energyLimit, long energyPrice, boolean isCreate) {
        if (status == null) {
            throw new NullPointerException("Cannot create InternalTransaction with null status!");
        }
        if (sender == null) {
            throw new NullPointerException("Cannot create InternalTransaction with null sender!");
        }
        if (senderNonce == null) {
            throw new NullPointerException("Cannot create InternalTransaction with null senderNonce!");
        }
        if (value == null) {
            throw new NullPointerException("Cannot create InternalTransaction with null value!");
        }
        if (data == null) {
            throw new NullPointerException("Cannot create InternalTransaction with null data!");
        }
        if (senderNonce.compareTo(BigInteger.ZERO) < 0) {
            throw new IllegalArgumentException("Cannot create InternalTransaction with negative senderNonce: " + senderNonce);
        }
        if (value.compareTo(BigInteger.ZERO) < 0) {
            throw new IllegalArgumentException("Cannot create InternalTransaction with negative value: " + value);
        }
        if (energyLimit < 0) {
            throw new IllegalArgumentException("Cannot create InternalTransaction with negative energyLimit: " + energyLimit);
        }
        if (energyPrice <= 0) {
            throw new IllegalArgumentException("Cannot create InternalTransaction with non-positive energyPrice: " + energyPrice);
        }

        this.sender = sender;
        this.destination = destination;
        this.senderNonce = senderNonce;
        this.value = value;
        this.energyLimit = energyLimit;
        this.energyPrice = energyPrice;
        this.isRejected = (status == RejectedStatus.REJECTED);
        this.isCreate = isCreate;
        this.data = copyOf(data);
    }

    /**
     * Constructs a new internal transaction that will attempt to create/deploy a new contract.
     *
     * This transaction will be marked as rejected only if {@code status == RejectedStatus.REJECTED}.
     *
     * @param status Whether this transaction it to be marked as rejected or not.
     * @param sender The sender of the transaction.
     * @param senderNonce The nonce of the sender.
     * @param value The amount of value to be transferred from the sender to the destination.
     * @param data The transaction data.
     * @param energyLimit The maximum amount of energy to be used by the transaction.
     * @param energyPrice The price per unit of energy to be charged.
     * @return a new internal transaction.
     */
    public static InternalTransaction contractCreateTransaction(RejectedStatus status, AionAddress sender, BigInteger senderNonce, BigInteger value, byte[] data, long energyLimit, long energyPrice) {
        return new InternalTransaction(status, sender, null, senderNonce, value, data, energyLimit, energyPrice, true);
    }

    /**
     * Constructs a new internal transaction that will be a contract-call transaction (this is a
     * call to a contract or a balance transfer).
     *
     * This transaction will be marked as rejected only if {@code status == RejectedStatus.REJECTED}.
     *
     * @param status Whether this transaction it to be marked as rejected or not.
     * @param sender The sender of the transaction.
     * @param destination The contract to be called or account to have value transferred to.
     * @param senderNonce The nonce of the sender.
     * @param value The amount of value to be transferred from the sender to the destination.
     * @param data The transaction data.
     * @param energyLimit The maximum amount of energy to be used by the transaction.
     * @param energyPrice The price per unit of energy to be charged.
     * @return a new internal transaction.
     */
    public static InternalTransaction contractCallTransaction(RejectedStatus status, AionAddress sender, AionAddress destination, BigInteger senderNonce, BigInteger value, byte[] data, long energyLimit, long energyPrice) {
        if (destination == null) {
            throw new NullPointerException("Cannot create InternalTransaction with null destination!");
        }

        return new InternalTransaction(status, sender, destination, senderNonce, value, data, energyLimit, energyPrice, false);
    }

    /**
     * Returns a copy of the transaction data.
     *
     * @return the transaction data.
     */
    public byte[] copyOfData() {
        return copyOf(this.data);
    }

    @Override
    public boolean equals(Object other) {
        if (!(other instanceof InternalTransaction)) {
            return false;
        } else if (other == this) {
            return true;
        }

        InternalTransaction otherTransaction = (InternalTransaction) other;

        return this.sender.equals(otherTransaction.sender)
            && ((this.destination == null) ? (otherTransaction.destination == null) : this.destination.equals(otherTransaction.destination))
            && this.senderNonce.equals(otherTransaction.senderNonce)
            && this.value.equals(otherTransaction.value)
            && Arrays.equals(this.data, otherTransaction.data)
            && (this.energyLimit == otherTransaction.energyLimit)
            && (this.energyPrice == otherTransaction.energyPrice)
            && (this.isRejected == otherTransaction.isRejected)
            && (this.isCreate == otherTransaction.isCreate);
    }

    @Override
    public int hashCode() {
        return this.sender.hashCode()
            + ((this.destination == null) ? 0 : this.destination.hashCode())
            + this.senderNonce.intValue() * 17
            + this.value.intValue() * 71
            + Arrays.hashCode(this.data)
            + (int) this.energyLimit * 127
            + (int) this.energyPrice * 5
            + ((this.isRejected) ? 1 : 0)
            + ((this.isCreate) ? 1 : 0);
    }

    @Override
    public String toString() {
        String type = (this.isCreate) ? "CREATE" : "CALL";
        String destination = (this.isCreate) ? "" : ", destination = " + this.destination;

        return "InternalTransaction { " + type
            + ", sender = " + this.sender
            + destination
            + ", nonce = " + this.senderNonce
            + ", value = " + this.value
            + ", data = " + this.data
            + ", energy limit = " + this.energyLimit
            + ", energy price = " + this.energyPrice
            + ", rejected = " + this.isRejected
            + " }";
    }

    private static byte[] copyOf(byte[] bytes) {
        return Arrays.copyOf(bytes, bytes.length);
    }
}
