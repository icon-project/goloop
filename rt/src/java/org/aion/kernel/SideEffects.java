package org.aion.kernel;

import java.util.ArrayList;
import java.util.Collection;
import java.util.List;

import org.aion.avm.core.InternalTransactionUtil;
import org.aion.types.InternalTransaction;
import org.aion.types.AionAddress;
import org.aion.types.Log;

/**
 * A class representing the side-effects that are caused by executing some external transaction.
 * These side-effects include the following data:
 *
 * 1. All of the logs generated during the execution of this transaction.
 * 2. All of the addressed that were marked to be deleted during the execution of this transaction.
 * 3. All of the internal transactions that were spawned as a result of executing this transaction.
 */
public class SideEffects {
    private List<Log> logs;
    private List<InternalTransaction> internalTransactions;

    /**
     * Constructs a new empty {@code SideEffects}.
     */
    public SideEffects() {
        this.logs = new ArrayList<>();
        this.internalTransactions = new ArrayList<>();
    }

    public void merge(SideEffects sideEffects) {
        addLogs(sideEffects.getExecutionLogs());
        addInternalTransactions(sideEffects.getInternalTransactions());
    }

    public void markAllInternalTransactionsAsRejected() {
        List<InternalTransaction> rejectedInternalTransactions = new ArrayList<>();
        for (InternalTransaction transaction : this.internalTransactions) {
            rejectedInternalTransactions.add(InternalTransactionUtil.createRejectedTransaction(transaction));
        }
        internalTransactions = rejectedInternalTransactions;
    }

    public void addInternalTransaction(InternalTransaction transaction) {
        this.internalTransactions.add(transaction);
    }

    public void addInternalTransactions(List<InternalTransaction> transactions) {
        this.internalTransactions.addAll(transactions);
    }

    public void addToDeletedAddresses(AionAddress address) {
        throw new AssertionError("We shouldn't be adding and deleted addresses in the AVM");
    }

    public void addAllToDeletedAddresses(Collection<AionAddress> addresses) {
        throw new AssertionError("We shouldn't be adding and deleted addresses in the AVM");
    }

    public void addLog(Log log) {
        this.logs.add(log);
    }

    public void addLogs(Collection<Log> logs) {
        this.logs.addAll(logs);
    }

    public List<InternalTransaction> getInternalTransactions() {
        return this.internalTransactions;
    }

    public List<AionAddress> getAddressesToBeDeleted() {
        return new ArrayList<>();
    }

    public List<Log> getExecutionLogs() {
        return this.logs;
    }

}
