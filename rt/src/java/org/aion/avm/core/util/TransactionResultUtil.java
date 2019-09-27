package org.aion.avm.core.util;

import i.RuntimeAssertionError;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import org.aion.avm.core.IExternalState;
import org.aion.kernel.AvmWrappedTransactionResult;
import org.aion.kernel.AvmWrappedTransactionResult.AvmInternalError;
import org.aion.types.InternalTransaction;
import org.aion.types.Log;
import org.aion.types.TransactionResult;
import org.aion.types.TransactionStatus;

/**
 * A utility class for constructing {@link AvmWrappedTransactionResult} objects. Typically, inside
 * the Avm we interact with {@link TransactionResult} via this wrapper class, which contains
 * additional data we require for our own purposes.
 */
public final class TransactionResultUtil {

    /**
     * Returns a new transaction result whose status is successful, with no logs or internal transactions,
     * and with the specified energy used and output.
     *
     * No exception or {@link IExternalState} are contained in the returned result either.
     *
     * @param energyUsed The energy used.
     * @param output The output.
     * @return the new result.
     */
    public static AvmWrappedTransactionResult newSuccessfulResultWithEnergyUsedAndOutput(long energyUsed, byte[] output) {
        TransactionResult result = new TransactionResult(TransactionStatus.successful(), Collections.emptyList(), Collections.emptyList(), energyUsed, output);
        return new AvmWrappedTransactionResult(result, null, null, AvmInternalError.NONE);
    }

    /**
     * Returns a new transaction result whose status is failed (specifically this is a non-reverted
     * failure!) and the type of failure is given by {@code failureError}. This result has no logs
     * or internal transactions, a null output, and the specified energy used.
     *
     * No exception or {@link IExternalState} are contained in the returned result either.
     *
     * @param failureError The failure type.
     * @param energyUsed The energy used.
     * @return the new result.
     */
    public static AvmWrappedTransactionResult newResultWithNonRevertedFailureAndEnergyUsed(AvmInternalError failureError, long energyUsed) {
        TransactionResult result = new TransactionResult(TransactionStatus.nonRevertedFailure(failureError.error), Collections.emptyList(), Collections.emptyList(), energyUsed, null);
        return new AvmWrappedTransactionResult(result, null, null, failureError);
    }

    /**
     * Returns a new transaction result whose status is successful, with no logs or internal transactions,
     * a null output, and with the specified energy used.
     *
     * No exception or {@link IExternalState} are contained in the returned result either.
     *
     * @param energyUsed The energy used.
     * @return the new result.
     */
    public static AvmWrappedTransactionResult newSuccessfulResultWithEnergyUsed(long energyUsed) {
        TransactionResult result = new TransactionResult(TransactionStatus.successful(), Collections.emptyList(), Collections.emptyList(), energyUsed, null);
        return new AvmWrappedTransactionResult(result, null, null, AvmInternalError.NONE);
    }

    /**
     * Returns a new transaction result whose status is rejected and the type of rejection is given
     * by {@code rejectedError}. This result has no logs or internal transactions, a null output,
     * and the specified energy used.
     *
     * No exception or {@link IExternalState} are contained in the returned result either.
     *
     * @param rejectedError The rejection type.
     * @param energyUsed The energy used.
     * @return the new result.
     */
    public static AvmWrappedTransactionResult newRejectedResultWithEnergyUsed(AvmInternalError rejectedError, long energyUsed) {
        TransactionResult result = new TransactionResult(TransactionStatus.rejection(rejectedError.error), Collections.emptyList(), Collections.emptyList(), energyUsed, null);
        return new AvmWrappedTransactionResult(result, null, null, rejectedError);
    }

    /**
     * Returns a new transaction result whose status is aborted. This result has no logs or
     * internal transactions, a null output, and zero energy used.
     *
     * No exception or {@link IExternalState} are contained in the returned result either.
     *
     * @return the new result.
     */
    public static AvmWrappedTransactionResult newAbortedResultWithZeroEnergyUsed() {
        TransactionResult result = new TransactionResult(TransactionStatus.nonRevertedFailure(AvmInternalError.ABORTED.error), Collections.emptyList(), Collections.emptyList(), 0, null);
        return new AvmWrappedTransactionResult(result, null, null, AvmInternalError.ABORTED);
    }

    /**
     * Returns a transaction result equivalent to {@code result} except that it contains all of the
     * specified logs and internal transactions in addition to whatever logs and internal transactions
     * it already contained.
     *
     * @param result The result which the new result will be derived from.
     * @param logs The logs to add.
     * @param transactions The internal transactions to add.
     * @return the new result.
     */
    public static AvmWrappedTransactionResult addLogsAndInternalTransactions(AvmWrappedTransactionResult result, List<Log> logs, List<InternalTransaction> transactions) {
        RuntimeAssertionError.assertTrue(logs != null);
        RuntimeAssertionError.assertTrue(transactions != null);

        List<Log> newLogs = new ArrayList<>(result.logs());
        newLogs.addAll(logs);

        List<InternalTransaction> newInternalTransactions = new ArrayList<>(result.internalTransactions());
        newInternalTransactions.addAll(transactions);

        TransactionResult transactionResult = new TransactionResult(result.transactionStatus(), newLogs, newInternalTransactions, result.energyUsed(), result.output());

        return new AvmWrappedTransactionResult(transactionResult, result.exception, result.externalState, result.avmInternalError);
    }

    /**
     * Returns a transaction result equivalent to {@code result} except that its external state will
     * be set to the specified {@code externalState}.
     *
     * @param result The result which the new result will be derived from.
     * @param externalState The new external state.
     * @return the new result.
     */
    public static AvmWrappedTransactionResult setExternalState(AvmWrappedTransactionResult result, IExternalState externalState) {
        RuntimeAssertionError.assertTrue(externalState != null);
        TransactionResult transactionResult = new TransactionResult(result.transactionStatus(), result.logs(), result.internalTransactions(), result.energyUsed(), result.output());
        return new AvmWrappedTransactionResult(transactionResult, result.exception, externalState, result.avmInternalError);
    }

    /**
     * Returns a transaction result equivalent to {@code result} except that its status will be set
     * to be a non-reverted failure, and the failure type will be the specified {@code failureError},
     * and it will also have the specified energy used.
     *
     * @param result The result which the new result will be derived from.
     * @param failureError The failure type.
     * @param energyUsed The new energy used.
     * @return the new result.
     */
    public static AvmWrappedTransactionResult setNonRevertedFailureAndEnergyUsed(AvmWrappedTransactionResult result, AvmInternalError failureError, long energyUsed) {
        RuntimeAssertionError.assertTrue(failureError != null);
        TransactionResult transactionResult = new TransactionResult(TransactionStatus.nonRevertedFailure(failureError.error), result.logs(), result.internalTransactions(), energyUsed, result.output());
        return new AvmWrappedTransactionResult(transactionResult, result.exception,  result.externalState, failureError);
    }

    /**
     * Returns a transaction result equivalent to {@code result} except that its status will be set
     * to be a reverted failure, and it will have the specified energy used.
     *
     * @param result The result which the new result will be derived from.
     * @param energyUsed The new energy used.
     * @return the new result.
     */
    public static AvmWrappedTransactionResult setRevertedFailureAndEnergyUsed(AvmWrappedTransactionResult result, long energyUsed) {
        TransactionResult transactionResult = new TransactionResult(TransactionStatus.revertedFailure(), result.logs(), result.internalTransactions(), energyUsed, result.output());
        return new AvmWrappedTransactionResult(transactionResult, result.exception,  result.externalState, AvmInternalError.FAILED_REVERTED);
    }

    /**
     * Returns a transaction result equivalent to {@code result} except that it will have the
     * specified output.
     *
     * Note that the output is allowed to be {@code null}.
     *
     * @param result The result which the new result will be derived from.
     * @param output The new output.
     * @return the new result.
     */
    public static AvmWrappedTransactionResult setSuccessfulOutput(AvmWrappedTransactionResult result, byte[] output) {
        TransactionResult transactionResult = new TransactionResult(TransactionStatus.successful(), result.logs(), result.internalTransactions(), result.energyUsed(), output);
        return new AvmWrappedTransactionResult(transactionResult, result.exception,  result.externalState, AvmInternalError.NONE);
    }

    /**
     * Returns a transaction result equivalent to {@code result} except that it will have the
     * specified energy used.
     *
     * @param result The result which the new result will be derived from.
     * @param energyUsed The new energy used.
     * @return the new result.
     */
    public static AvmWrappedTransactionResult setEnergyUsed(AvmWrappedTransactionResult result, long energyUsed) {
        TransactionResult transactionResult = new TransactionResult(result.transactionStatus(), result.logs(), result.internalTransactions(), energyUsed, result.output());
        return new AvmWrappedTransactionResult(transactionResult, result.exception,  result.externalState, result.avmInternalError);
    }

    /**
     * Returns a transaction result equivalent to {@code result} except that its status will be set
     * to be a non-reverted failure and the failure type will be {@link AvmInternalError#FAILED_EXCEPTION},
     * and it will have the specified exception and energy used.
     *
     * @param result The result which the new result will be derived from.
     * @param exception The exception.
     * @param energyUsed The new energy used.
     * @return the new result.
     */
    public static AvmWrappedTransactionResult setFailedException(AvmWrappedTransactionResult result, Throwable exception, long energyUsed) {
        RuntimeAssertionError.assertTrue(exception != null);
        TransactionResult transactionResult = new TransactionResult(TransactionStatus.nonRevertedFailure(AvmInternalError.FAILED_EXCEPTION.error), result.logs(), result.internalTransactions(), energyUsed, result.output());
        return new AvmWrappedTransactionResult(transactionResult, exception,  result.externalState, AvmInternalError.FAILED_EXCEPTION);
    }

    /**
     * Returns a transaction result equivalent to {@code result} except that its status will be set
     * to be a non-reverted failure and the failure type will be {@link AvmInternalError#FAILED_UNEXPECTED},
     * and it will have the specified exception and energy used.
     *
     * @param result The result which the new result will be derived from.
     * @param exception The exception.
     * @param energyUsed The new energy used.
     * @return the new result.
     */
    public static AvmWrappedTransactionResult setFailedUnexpected(AvmWrappedTransactionResult result, Throwable exception, long energyUsed) {
        RuntimeAssertionError.assertTrue(exception != null);
        TransactionResult transactionResult = new TransactionResult(TransactionStatus.nonRevertedFailure(AvmInternalError.FAILED_UNEXPECTED.error), result.logs(), result.internalTransactions(), energyUsed, result.output());
        return new AvmWrappedTransactionResult(transactionResult, exception,  result.externalState, AvmInternalError.FAILED_UNEXPECTED);
    }
}
