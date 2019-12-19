package org.aion.types;

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
    public final long energyUsed;
    private final Object output;

    /**
     * Constructs a new transaction result.
     *
     * @param transactionStatus The status of executing the transaction.
     * @param energyUsed The amount of energy used during the execution of the transaction.
     * @param output The output of the transaction.
     */
    public TransactionResult(TransactionStatus transactionStatus, long energyUsed, Object output) {
        if (transactionStatus == null) {
            throw new NullPointerException("Cannot construct TransactionResult with null transactionStatus!");
        }
        if (energyUsed < 0) {
            throw new IllegalArgumentException("Cannot construct TransactionResult with negative energyUsed!");
        }
        this.transactionStatus = transactionStatus;
        this.energyUsed = energyUsed;
        this.output = copyOfOutput(output);
    }

    /**
     * Returns a copy of the transaction output if the transaction had any output.
     *
     * @return the transaction output.
     */
    public Object copyOfTransactionOutput() {
        return copyOfOutput(output);
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
            && (this.energyUsed == otherResult.energyUsed)
            && equalOutput(this.output, otherResult.output);
    }

    private static boolean equalOutput(Object a, Object b) {
        if (a==null) {
            return b==null;
        }
        return a.equals(b);
    }

    @Override
    public int hashCode() {
        return this.transactionStatus.hashCode()
            + ((int) this.energyUsed)
            + (this.output != null ? this.output.hashCode() : 7);
    }

    @Override
    public String toString() {
        return "TransactionResult { "
            + "status = " + this.transactionStatus
            + ", energy used = " + this.energyUsed
            + ", output = " + ((this.output == null) ? "null" : this.output)
            + " }";
    }

    private static Object copyOfOutput(Object output) {
        if (output instanceof byte[]) {
            return ((byte[])output).clone();
        }
        return output;
    }
}
