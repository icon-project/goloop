package org.aion.types;

/**
 * A class that represents the status of executing a transaction.
 *
 * A transaction either executes successfully or else, if unsuccessful, is either rejected, failed,
 * reverted, or fatal.
 *
 * Note that reverted is a type of failure, but it is a failure type with special behaviour
 * associated with it and thus why it is distinguished from other failures. Therefore, if
 * {@code isReverted() == true} then {@code isFailed() == true}.
 *
 * A rejected transaction is a transaction that violates some rule prior to executing any code.
 *
 * A failure/revert is an error encountered while executing code.
 *
 * A fatal error is an error that cannot be recovered from and indicates that the virtual machine
 * can no longer be used.
 *
 * Exactly one of the following will be true: isSuccess(), isRejected(), isFailed(), isFatal().
 *
 * It is guaranteed that {@code causeOfError} is always non-null! In cases where there is no error
 * to report, such as success, this will be the empty string.
 *
 * A transaction status object is immutable.
 */
public final class TransactionStatus {
    private final InternalStatus status;
    public final String causeOfError;

    private enum InternalStatus { SUCCESS, REJECTED, REVERTED_FAILURE, NON_REVERTED_FAILURE, FATAL }

    private TransactionStatus(InternalStatus status, String causeOfError) {
        if (status == null) {
            throw new NullPointerException("Cannot construct ResultStatus with null status!");
        }
        if (causeOfError == null) {
            throw new NullPointerException("Cannot construct ResultStatus with null causeOfError!");
        }

        this.status = status;
        this.causeOfError = causeOfError;
    }

    /**
     * Constructs a new successful transaction status.
     */
    public static TransactionStatus successful() {
        return new TransactionStatus(InternalStatus.SUCCESS, "");
    }

    /**
     * Constructs a new rejected transaction status with the specified causeOfError.
     */
    public static TransactionStatus rejection(String causeOfError) {
        return new TransactionStatus(InternalStatus.REJECTED, causeOfError);
    }

    /**
     * Constructs a new reverted failed transaction status.
     */
    public static TransactionStatus revertedFailure() {
        return new TransactionStatus(InternalStatus.REVERTED_FAILURE, "reverted");
    }

    /**
     * Constructs a new non-reverted failed transaction status with the specified causeOfError.
     */
    public static TransactionStatus nonRevertedFailure(String causeOfError) {
        return new TransactionStatus(InternalStatus.NON_REVERTED_FAILURE, causeOfError);
    }

    /**
     * Constructs a new fatal transaction status.
     */
    public static TransactionStatus fatal(String causeOfError) {
        return new TransactionStatus(InternalStatus.FATAL, causeOfError);
    }

    /**
     * Returns {@code true} only if the transaction was successful.
     *
     * @return whether the transaction was successful or not.
     */
    public boolean isSuccess() {
        return this.status == InternalStatus.SUCCESS;
    }

    /**
     * Returns {@code true} only if the transaction was rejected.
     *
     * @return whether the transaction was rejected or not.
     */
    public boolean isRejected() {
        return this.status == InternalStatus.REJECTED;
    }

    /**
     * Returns {@code true} only if the transaction failed.
     *
     * @return whether the transaction failed or not.
     */
    public boolean isFailed() {
        return this.status == InternalStatus.REVERTED_FAILURE || this.status == InternalStatus.NON_REVERTED_FAILURE;
    }

    /**
     * Returns {@code true} only if the transaction was reverted.
     *
     * @return whether the transaction was reverted or not.
     */
    public boolean isReverted() {
        return this.status == InternalStatus.REVERTED_FAILURE;
    }

    /**
     * Returns {@code true} only if a fatal error was encountered.
     *
     * @return whether a fatal error was encountered or not.
     */
    public boolean isFatal() {
        return this.status == InternalStatus.FATAL;
    }

    /**
     * Returns {@code true} only if other is a transaction status object and if it is the same type
     * of status object (success/rejected/failure/reverted/fatal) and if the causeOfError of both
     * the statuses are the same.
     *
     * @param other The other object whose equality with this is to be tested.
     * @return whether other is equal to this.
     */
    @Override
    public boolean equals(Object other) {
        if (!(other instanceof TransactionStatus)) {
            return false;
        } else if (other == this) {
            return true;
        }

        TransactionStatus otherStatus = (TransactionStatus) other;
        return (this.status == otherStatus.status) && this.causeOfError.equals(otherStatus.causeOfError);
    }

    @Override
    public int hashCode() {
        return this.status.hashCode() + this.causeOfError.hashCode();
    }

    @Override
    public String toString() {
        if (isSuccess()) {
            return "TransactionStatus { successful }";
        } else if (isRejected()) {
            return "TransactionStatus { rejected due to: " + this.causeOfError + " }";
        } else if (isFailed()) {
            return "TransactionStatus { failed due to: " + this.causeOfError + " }";
        } else {
            return "TransactionStatus { a fatal error occurred: " + this.causeOfError + " }";
        }
    }
}
