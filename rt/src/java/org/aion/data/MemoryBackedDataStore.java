package org.aion.data;

import java.util.HashMap;
import java.util.Map;

import org.aion.avm.core.util.ByteArrayWrapper;


public class MemoryBackedDataStore implements IDataStore {
    private final Map<ByteArrayWrapper, MemoryBackedAccountStore> accounts = new HashMap<>();

    @Override
    public IAccountStore openAccount(byte[] address) {
        return this.accounts.get(new ByteArrayWrapper(address));
    }

    @Override
    public IAccountStore createAccount(byte[] address) {
        ByteArrayWrapper wrapper = new ByteArrayWrapper(address);
        MemoryBackedAccountStore existing = this.accounts.get(wrapper);
        MemoryBackedAccountStore created = null;
        if (null == existing) {
            created = new MemoryBackedAccountStore();
            this.accounts.put(wrapper, created);
        }
        return created;
    }

    @Override
    public void deleteAccount(byte[] address) {
        this.accounts.remove(new ByteArrayWrapper(address));
    }
}
