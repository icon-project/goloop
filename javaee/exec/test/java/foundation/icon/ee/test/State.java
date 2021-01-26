package foundation.icon.ee.test;

import foundation.icon.ee.types.Address;
import org.aion.avm.core.util.ByteArrayWrapper;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class State {
    private static class AccountImpl implements Account {
        private final Address address;
        private final Map<ByteArrayWrapper, byte[]> storage = new HashMap<>();
        private BigInteger balance = BigInteger.ZERO;
        private Contract contract;

        public AccountImpl(Address address) {
            this.address = address;
        }

        public AccountImpl(AccountImpl other, Contract contract) {
            this.address = other.address;
            storage.putAll(other.storage);
            this.balance = other.balance;
            this.contract = contract;
        }

        public Address getAddress() {
            return address;
        }

        public byte[] getStorage(byte[] key) {
            return storage.get(new ByteArrayWrapper(key));
        }

        public byte[] setStorage(byte[] key, byte[] value) {
            return storage.put(new ByteArrayWrapper(key), value);
        }

        public byte[] removeStorage(byte[] key) {
            return storage.remove(new ByteArrayWrapper(key));
        }

        public BigInteger getBalance() {
            return balance;
        }

        public void setBalance(BigInteger balance) {
            this.balance = balance;
        }

        public Contract getContract() {
            return contract;
        }

        public byte[] getContractID() {
            return this.contract.getID();
        }
    }

    private final Map<ByteArrayWrapper, Contract> contracts = new HashMap<>();
    private final Map<Address, AccountImpl> accounts = new HashMap<>();
    private final List<Contract> garbage = new ArrayList<>();

    public State() {
    }

    public State(State src) {
        for (var entry : src.contracts.entrySet()) {
            contracts.put(entry.getKey(), new Contract(entry.getValue()));
        }
        for (var entry : src.accounts.entrySet()) {
            var c = contracts.get(new ByteArrayWrapper(entry.getValue().getContractID()));
            accounts.put(entry.getKey(), new AccountImpl(entry.getValue(), c));
        }
        garbage.addAll(src.garbage);
    }


    public Account getAccount(Address address) {
        return doGetAccount(address);
    }

    private AccountImpl doGetAccount(Address address) {
        var account = accounts.get(address);
        if (account==null) {
            account = new AccountImpl(address);
            accounts.put(address, account);
        }
        return account;
    }

    public Contract getContract(byte[] contractID) {
        return contracts.get(new ByteArrayWrapper(contractID));
    }

    public void deploy(Address address, Contract contract) {
        var account = doGetAccount(address);
        if (account.contract != null) {
            garbage.add(account.contract);
        }
        contracts.put(new ByteArrayWrapper(contract.getID()), contract);
        account.contract = contract;
    }

    public void gc() {
        for (var c: garbage) {
            contracts.remove(new ByteArrayWrapper(c.getID()));
        }
    }

    public void clearEID() {
        for (var e: contracts.entrySet()) {
            e.getValue().setEID(0);
        }
    }
}
