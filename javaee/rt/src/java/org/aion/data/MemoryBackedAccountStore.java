package org.aion.data;

import java.math.BigInteger;
import java.util.HashMap;
import java.util.Map;

import org.aion.avm.core.util.ByteArrayWrapper;


public class MemoryBackedAccountStore implements IAccountStore {
    private BigInteger balance = BigInteger.ZERO;
    private long nonce = 0;
    private byte[] code = null;
    private byte[] transformedCode = null;
    private final Map<ByteArrayWrapper, byte[]> storage = new HashMap<>();
    private byte[] graph = null;

    @Override
    public byte[] getCode() {
        return this.code;
    }

    @Override
    public void setCode(byte[] code) {
        this.code = code;
    }

    @Override
    public byte[] getTransformedCode() {
        return this.transformedCode;
    }

    @Override
    public void setTransformedCode(byte[] code) {
        this.transformedCode = code;
    }

    @Override
    public BigInteger getBalance() {
        return this.balance;
    }

    @Override
    public void setBalance(BigInteger balance) {
        this.balance = balance;
    }

    @Override
    public long getNonce() {
        return this.nonce;
    }

    @Override
    public void setNonce(long nonce) {
        this.nonce = nonce;
    }

    @Override
    public byte[] getData(byte[] key) {
        return this.storage.get(new ByteArrayWrapper(key));
    }

    @Override
    public void setData(byte[] key, byte[] value) {
        this.storage.put(new ByteArrayWrapper(key), value);
    }

    @Override
    public void removeData(byte[] key) {
        this.storage.remove(new ByteArrayWrapper(key));
    }

    @Override
    public Map<ByteArrayWrapper, byte[]> getStorageEntries() {
        return this.storage;
    }

    @Override
    public void setObjectGraph(byte[] data) {
        this.graph = data;
    }

    @Override
    public byte[] getObjectGraph() {
        return this.graph;
    }
}
