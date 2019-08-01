package org.aion.types;

import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.Optional;

/**
 * A class that represents the end result of executing a transaction.
 *
 * This result includes the status of executing the transaction, the logs fired off during its
 * execution, the list of internal transactions spawned as a result of executing the transaction,
 * the amount of energy used by the transaction and the output of the transaction.
 *
 * A transaction may or may not have any output.
 *
 * A transaction result is immutable.
 */
public final class TransactionResult {
    public final TransactionStatus transactionStatus;
    public final List<InternalTransaction> internalTransactions;
    public final List<Log> logs;
    public final long energyUsed;
    private final byte[] output;

    /**
     * Constructs a new transaction result.
     *
     * @param transactionStatus The status of executing the transaction.
     * @param logs The logs fired off during execution of the transaction.
     * @param internalTransactions The internal transactions spawned during the execution of the transaction.
     * @param energyUsed The amount of energy used during the executin of the transaction.
     * @param output The output of the transaction.
     */
    public TransactionResult(TransactionStatus transactionStatus, List<Log> logs, List<InternalTransaction> internalTransactions, long energyUsed, byte[] output) {
        if (transactionStatus == null) {
            throw new NullPointerException("Cannot construct TransactionResult with null transactionStatus!");
        }
        if (logs == null) {
            throw new NullPointerException("Cannot construct TransactionResult with null logs!");
        }
        if (internalTransactions == null) {
            throw new NullPointerException("Cannot construct TransactionResult with null internalTransactions!");
        }
        if (energyUsed < 0) {
            throw new IllegalArgumentException("Cannot construct TransactionResult with negative energyUsed!");
        }

        this.transactionStatus = transactionStatus;
        this.logs = Collections.unmodifiableList(logs);
        this.internalTransactions = Collections.unmodifiableList(internalTransactions);
        this.energyUsed = energyUsed;
        this.output = (output == null) ? null : copyOf(output);
    }

    /**
     * Returns a copy of the transaction output if the transaction had any output.
     *
     * @return the transaction output.
     */
    public Optional<byte[]> copyOfTransactionOutput() {
        return (this.output == null) ? Optional.empty() : Optional.of(copyOf(this.output));
    }

    /**
     * Returns {@code true} only if other is a transaction result object and if that result has the
     * an equal transaction status, list of logs, list of internal transactions, energy used, and
     * transaction output.
     *
     * @param other The other object whose equality is to be tested.
     * @return whether other is equal to this.
     */
    @Override
    public boolean equals(Object other) {
        if (!(other instanceof TransactionResult)) {
            return false;
        } else if (other == this) {
            return true;
        }

        TransactionResult otherResult = (TransactionResult) other;
        return this.transactionStatus.equals(otherResult.transactionStatus)
            && this.logs.equals(otherResult.logs)
            && this.internalTransactions.equals(otherResult.internalTransactions)
            && (this.energyUsed == otherResult.energyUsed)
            && Arrays.equals(this.output, otherResult.output);
    }

    @Override
    public int hashCode() {
        return this.transactionStatus.hashCode()
            + this.logs.hashCode()
            + this.internalTransactions.hashCode()
            + ((int) this.energyUsed)
            + Arrays.hashCode(this.output);
    }

    @Override
    public String toString() {
        return "TransactionResult { "
            + "status = " + this.transactionStatus
            + ", energy used = " + this.energyUsed
            + ", output = " + ((this.output == null) ? "null" : this.output)
            + ", logs = " + this.logs
            + ", internal transactions = " + this.internalTransactions
            + " }";
    }

    private static byte[] copyOf(byte[] bytes) {
        return Arrays.copyOf(bytes, bytes.length);
    }
}
