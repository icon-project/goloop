/*
 * Copyright 2020 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package example;

import score.Address;
import score.ArrayDB;
import score.BranchDB;
import score.Context;
import score.DictDB;
import score.VarDB;
import score.annotation.EventLog;
import score.annotation.External;
import score.annotation.Optional;
import score.annotation.Payable;

import java.math.BigInteger;
import java.util.List;
import java.util.Map;

public class MultiSigWallet
{
    private static final int MAX_OWNER_COUNT = 50;

    private final ArrayDB<Address> owners = Context.newArrayDB("owners", Address.class);
    private final VarDB<BigInteger> required = Context.newVarDB("required", BigInteger.class);
    private final DictDB<BigInteger, Transaction> transactions = Context.newDictDB("transactions", Transaction.class);
    private final VarDB<BigInteger> transactionCount = Context.newVarDB("transactionCount", BigInteger.class);
    private final BranchDB<BigInteger, DictDB<Address, Boolean>>
            confirmations = Context.newBranchDB("confirmations", Boolean.class);

    /**
     * Contract constructor sets initial owners and required number of confirmations.
     */
    public MultiSigWallet(String _walletOwners, BigInteger _required) {
        assert(this.owners != null);
        StringTokenizer st = new StringTokenizer(_walletOwners, ", ");
        while (st.hasMoreTokens()) {
            this.owners.add(Address.fromString(st.nextToken()));
        }
        checkRequirement(this.owners.size(), _required);
        this.required.set(_required);
    }

    @Payable
    public void fallback() {
        BigInteger value = Context.getValue();
        if (value.signum() > 0) {
            Deposit(Context.getCaller(), value);
        }
    }

    @External
    public void tokenFallback(Address _from, BigInteger _value, byte[] _data) {
        if (_value.signum() > 0) {
            DepositToken(_from, _value, _data);
        }
    }

    @External
    public void addWalletOwner(Address _walletOwner) {
        onlyFromWallet();
        checkOwnerDoesNotExist(_walletOwner);
        checkRequirement(this.owners.size() + 1, this.required.get());
        // now we can add the owner
        this.owners.add(_walletOwner);
        WalletOwnerAddition(_walletOwner);
    }

    @External
    public void removeWalletOwner(Address _walletOwner) {
        onlyFromWallet();
        checkOwnerExist(_walletOwner);
        checkRequirement(this.owners.size() - 1, this.required.get());
        // get the topmost value
        Address top = this.owners.pop();
        if (!top.equals(_walletOwner)) {
            for (int i = 0; i < this.owners.size(); i++) {
                if (_walletOwner.equals(this.owners.get(i))) {
                    this.owners.set(i, top);
                    break;
                }
            }
        }
        WalletOwnerRemoval(_walletOwner);
    }

    @External
    public void replaceWalletOwner(Address _walletOwner, Address _newWalletOwner) {
        onlyFromWallet();
        checkOwnerExist(_walletOwner);
        checkOwnerDoesNotExist(_newWalletOwner);
        // now we can replace the owner
        for (int i = 0; i < this.owners.size(); i++) {
            if (_walletOwner.equals(this.owners.get(i))) {
                this.owners.set(i, _newWalletOwner);
                break;
            }
        }
        WalletOwnerRemoval(_walletOwner);
        WalletOwnerAddition(_newWalletOwner);
    }

    @External
    public void changeRequirement(BigInteger _required) {
        onlyFromWallet();
        checkRequirement(this.owners.size(), _required);
        this.required.set(_required);
        RequirementChange(_required);
    }

    /**
     * Allows an owner to submit and confirm a transaction.
     */
    @External
    public void submitTransaction(Address _destination,
                                  @Optional String _method,
                                  @Optional String _params,
                                  @Optional BigInteger _value,
                                  @Optional String _description) {
        checkOwnerExist(Context.getCaller());
        Context.require(_value == null || _value.signum() >= 0);
        BigInteger transactionId = addTransaction(_destination, _method, _params, _value, _description);
        confirmTransaction(transactionId);
    }

    /**
     * Allows an owner to confirm a transaction.
     */
    @External
    public void confirmTransaction(BigInteger _transactionId) {
        Address sender = Context.getCaller();
        checkOwnerExist(sender);
        checkTransactionExist(_transactionId);
        checkNotConfirmed(_transactionId, sender);
        // set confirmation true for the sender
        this.confirmations.at(_transactionId).set(sender, true);
        Confirmation(sender, _transactionId);
        executeTransaction(_transactionId);
    }

    /**
     * Allows an owner to revoke a confirmation for a transaction.
     */
    @External
    public void revokeTransaction(BigInteger _transactionId) {
        Address sender = Context.getCaller();
        checkOwnerExist(sender);
        checkTransactionExist(_transactionId);
        checkNotExecuted(_transactionId);
        checkConfirmed(_transactionId, sender);
        // set confirmation false for the sender
        this.confirmations.at(_transactionId).set(sender, false);
        Revocation(sender, _transactionId);
    }

    /*
     * Read-only methods
     */
    @External(readonly=true)
    public BigInteger getRequirement() {
        return this.required.get();
    }

    @External(readonly=true)
    public List<Address> getWalletOwners() {
        int len = this.owners.size();
        Address[] array = new Address[len];
        for (int i = 0; i < len; i++) {
            array[i] = this.owners.get(i);
        }
        return List.of(array);
    }

    @External(readonly=true)
    public int getConfirmationCount(BigInteger _transactionId) {
        int count = 0;
        for (int i = 0; i < this.owners.size(); i++) {
            if (this.confirmations.at(_transactionId).getOrDefault(this.owners.get(i), false)) {
                count++;
            }
        }
        return count;
    }

    @External(readonly=true)
    public List<Address> getConfirmations(BigInteger _transactionId) {
        int len = this.owners.size();
        Address[] array = new Address[len];
        int count = 0;
        for (int i = 0; i < len; i++) {
            Address owner = this.owners.get(i);
            if (this.confirmations.at(_transactionId).getOrDefault(owner, false)) {
                array[count++] = owner;
            }
        }
        Address[] confirmations = new Address[count];
        System.arraycopy(array, 0, confirmations, 0, count);
        return List.of(confirmations);
    }

    @External(readonly=true)
    public BigInteger getTransactionCount(boolean _pending, boolean _executed) {
        BigInteger count = BigInteger.ZERO;
        BigInteger total = this.transactionCount.getOrDefault(BigInteger.ZERO);
        while (total.signum() > 0) {
            total = total.subtract(BigInteger.ONE);
            Transaction transaction = this.transactions.get(total);
            if (_pending && !transaction.executed() || _executed && transaction.executed()) {
                count = count.add(BigInteger.ONE);
            }
        }
        return count;
    }

    @External(readonly=true)
    public Map<String, String> getTransactionInfo(BigInteger _transactionId) {
        Transaction transaction = this.transactions.get(_transactionId);
        if (transaction == null) {
            return Map.of();
        }
        return transaction.toMap(_transactionId);
    }

    @External(readonly=true)
    public List<BigInteger> getTransactionIds(BigInteger _offset, BigInteger _count,
                                              boolean _pending, boolean _executed) {
        Context.require(_offset.signum() >= 0 && _count.signum() >= 0);
        BigInteger total = this.transactionCount.getOrDefault(BigInteger.ZERO);
        if (_offset.add(_count).compareTo(total) > 0) {
            _count = total.subtract(_offset);
        }
        if (_count.signum() <= 0) {
            return List.of();
        }
        BigInteger[] entries = new BigInteger[_count.intValue()];
        int index = 0;
        for (int i = 0; _count.signum() > 0; i++) {
            _count = _count.subtract(BigInteger.ONE);
            BigInteger transactionId = _offset.add(BigInteger.valueOf(i));
            Transaction transaction = this.transactions.get(transactionId);
            if (_pending && !transaction.executed() || _executed && transaction.executed()) {
                entries[index++] = transactionId;
            }
        }
        if (index < entries.length) {
            BigInteger[] tmp = new BigInteger[index];
            System.arraycopy(entries, 0, tmp, 0, index);
            entries = tmp;
        }
        return List.of(entries);
    }

    @External(readonly=true)
    public List<Map<String, String>> getTransactionList(BigInteger _offset, BigInteger _count,
                                                        boolean _pending, boolean _executed) {
        Context.require(_offset.signum() >= 0 && _count.signum() >= 0);
        BigInteger total = this.transactionCount.getOrDefault(BigInteger.ZERO);
        if (_offset.add(_count).compareTo(total) > 0) {
            _count = total.subtract(_offset);
        }
        if (_count.signum() <= 0) {
            return List.of();
        }
        @SuppressWarnings("unchecked")
        Map<String, String>[] entries = new Map[_count.intValue()];
        int index = 0;
        for (int i = 0; _count.signum() > 0; i++) {
            _count = _count.subtract(BigInteger.ONE);
            BigInteger transactionId = _offset.add(BigInteger.valueOf(i));
            Transaction transaction = this.transactions.get(transactionId);
            if (_pending && !transaction.executed() || _executed && transaction.executed()) {
                entries[index++] = transaction.toMap(transactionId);
            }
        }
        if (index < entries.length) {
            @SuppressWarnings("unchecked")
            Map<String, String>[] tmp = new Map[index];
            System.arraycopy(entries, 0, tmp, 0, index);
            entries = tmp;
        }
        return List.of(entries);
    }

    /*
     * Assertion methods
     */
    private void onlyFromWallet() {
        assert(Context.getAddress() != null);
        Context.require(Context.getAddress().equals(Context.getCaller()));
    }

    private void checkRequirement(int ownerCount, BigInteger required) {
        int _required = required.intValue();
        Context.require((ownerCount <= MAX_OWNER_COUNT)
                && (_required <= ownerCount)
                && (_required > 0));
    }

    private void checkOwnerExist(Address owner) {
        //TODO: iteration is not efficient. Consider to use a Map.
        for (int i = 0; i < this.owners.size(); i++) {
            if (owner.equals(this.owners.get(i))) {
                return;
            }
        }
        Context.revert(100, "Owner not exist");
    }

    private void checkOwnerDoesNotExist(Address owner) {
        //TODO: iteration is not efficient. Consider to use a Map.
        for (int i = 0; i < this.owners.size(); i++) {
            if (owner.equals(this.owners.get(i))) {
                Context.revert(101, "Owner already exists");
            }
        }
    }

    private void checkTransactionExist(BigInteger transactionId) {
        Context.require(this.transactions.get(transactionId) != null);
    }

    private void checkConfirmed(BigInteger transactionId, Address sender) {
        Context.require(this.confirmations.at(transactionId).getOrDefault(sender, false));
    }

    private void checkNotConfirmed(BigInteger transactionId, Address sender) {
        Context.require(!this.confirmations.at(transactionId).getOrDefault(sender, false));
    }

    private void checkNotExecuted(BigInteger transactionId) {
        Transaction transaction = this.transactions.get(transactionId);
        Context.require(!transaction.executed());
    }

    /*
     * Internal methods
     */
    private BigInteger addTransaction(Address destination, String method, String params,
                                      BigInteger value, String description) {
        BigInteger transactionId = this.transactionCount.getOrDefault(BigInteger.ZERO);
        this.transactions.set(transactionId,
                new Transaction(destination, method, params, value, description));
        this.transactionCount.set(transactionId.add(BigInteger.ONE));
        Submission(transactionId);
        return transactionId;
    }

    private void executeTransaction(BigInteger transactionId) {
        if (isConfirmed(transactionId)) {
            Transaction transaction = this.transactions.get(transactionId);
            Context.require(!transaction.executed());
            if (externalCall(transaction)) {
                transaction.setExecuted(true);
                // we need to set the transaction again since we changed the executed status
                this.transactions.set(transactionId, transaction);
                Execution(transactionId);
            } else {
                ExecutionFailure(transactionId);
            }
        }
    }

    private boolean isConfirmed(BigInteger transactionId) {
        int count = 0;
        int required = this.required.get().intValue();
        for (int i = 0; i < this.owners.size(); i++) {
            if (this.confirmations.at(transactionId).getOrDefault(this.owners.get(i), false)) {
                count++;
            }
        }
        // execute the transaction only if the confirmed count exactly matches the required
        return count == required;
    }

    private boolean externalCall(Transaction transaction) {
        try {
            Context.call(transaction.value(), transaction.destination(),
                    transaction.method(), transaction.getConvertedParams());
            return true;
        } catch (Exception e) {
            Context.println("[Exception] " + e.getMessage());
            return false;
        }
    }

    /*
     * Events
     */
    @EventLog(indexed=1)
    protected void WalletOwnerAddition(Address _walletOwner) {}

    @EventLog(indexed=1)
    protected void WalletOwnerRemoval(Address _walletOwner) {}

    @EventLog
    protected void RequirementChange(BigInteger _required) {}

    @EventLog(indexed=1)
    protected void Submission(BigInteger _transactionId) {}

    @EventLog(indexed=2)
    protected void Confirmation(Address _sender, BigInteger _transactionId) {}

    @EventLog(indexed=2)
    protected void Revocation(Address _sender, BigInteger _transactionId) {}

    @EventLog(indexed=1)
    protected void Execution(BigInteger _transactionId) {}

    @EventLog(indexed=1)
    protected void ExecutionFailure(BigInteger _transactionId) {}

    @EventLog(indexed=1)
    protected void Deposit(Address _sender, BigInteger _value) {}

    @EventLog(indexed=1)
    protected void DepositToken(Address _sender, BigInteger _value, byte[] _data) {}
}
