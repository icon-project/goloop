package foundation.icon.ee.test;

import foundation.icon.ee.types.Address;
import org.aion.avm.core.util.ByteArrayWrapper;

import java.math.BigInteger;
import java.util.HashMap;
import java.util.Map;

public class State {
    static class Account {
        public Address address;
        public BigInteger balance = BigInteger.ZERO;
        public int nextHash = 0;
        public byte[] objectGraph = null;
        public byte[] objectGraphHash = null;
        public Map<ByteArrayWrapper, byte[]> storage = new HashMap<>();
        public byte[] optimized = null;
        public byte[] transformed = null;
        public Contract contract = null;
        public int exid = 0;
        public int eid = 0;

        Account(byte[] addr) {
            address = new Address(addr);
        }

        Account(Account src) {
            address = src.address;
            balance = src.balance;
            nextHash = src.nextHash;
            objectGraph = src.objectGraph;
            objectGraphHash = src.objectGraphHash;
            storage.putAll(src.storage);
            optimized = src.optimized;
            transformed = src.transformed;
            contract = src.contract;
            exid = src.exid;
            eid = src.eid;
        }
    }

    private Map<ByteArrayWrapper, Account> accounts = new HashMap<>();
    private Map<String, byte[]> files = new HashMap<>();

    public State() {
    }

    public State(State src) {
        for (var entry : src.accounts.entrySet()) {
            accounts.put(entry.getKey(), new Account(entry.getValue()));
        }
        files.putAll(src.files);
    }

    public Account getAccount(Address addr) {
        var ba = addr.toByteArray();
        var baw = new ByteArrayWrapper(ba);
        var account = accounts.get(baw);
        if (account==null) {
            account = new Account(ba);
            accounts.put(baw, account);
        }
        return account;
    }

    public void writeFile(String path, byte[] data) {
        files.put(path, data.clone());
    }

    public byte[] readFile(String path) {
        var data = files.get(path);
        if (data!=null) {
            data = data.clone();
        }
        return data;
    }
}
