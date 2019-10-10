package org.aion.avm.core;

import i.ValueCodec;
import org.aion.avm.RuntimeMethodFeeSchedule;
import org.aion.avm.StorageFees;
import org.aion.avm.embed.hash.HashUtils;
import p.avm.ArrayDB;
import p.avm.DictDB;
import p.avm.PrimitiveBuffer;
import s.java.lang.String;

public class CollectionDBImpl<K, V> extends DBImplBase implements DictDB<K, V>, ArrayDB<V> {
    public CollectionDBImpl(DBStorage storage, String id) {
        super(storage, id);
    }

    public CollectionDBImpl(DBStorage storage, byte[] id) {
        super(storage, id);
    }

    private byte[] getStorageKey(K key) {
        return HashUtils.sha3_256(DBImplBase.catEncodedKey(id, key));
    }

    private byte[] getStorageKey(int key) {
        return HashUtils.sha3_256(DBImplBase.catEncodedKey(id, key));
    }

    /**
     * @param key
     * @param value
     * @throws InvalidDBAccessException if key is null,
     */
    public void avm_putValue(K key, V value) {
        int w = 0;
        try {
            ensureStorage();
            assumeValidValue(value);
            var v = ValueCodec.encodeValue(value);
            w = v.length;
            storage.setValue(getStorageKey(key), v);
        } finally {
            charge(RuntimeMethodFeeSchedule.DictDB_avm_putValue
                    + StorageFees.WRITE_PRICE_PER_BYTE * w);
        }
    }

    public V avm_get(K key) {
        try {
            ensureStorage();
            return (V) new CollectionDBImpl(storage, DBImplBase.catEncodedKey(id, key));
        } finally {
            charge(RuntimeMethodFeeSchedule.DictDB_avm_get);
        }
    }

    public V avm_getValue(K key) {
        int r = 0;
        try {
            ensureStorage();
            var v = storage.getValue(getStorageKey(key));
            r = v.length;
            return (V) ValueCodec.decodeValue(v);

        } finally {
            charge(RuntimeMethodFeeSchedule.DictDB_avm_getValue
                    + StorageFees.READ_PRICE_PER_BYTE * r);
        }
    }

    public PrimitiveBuffer avm_getValue(K key, PrimitiveBuffer out) {
        int r = 0;
        try {
            ensureStorage();
            var v = storage.getValue(getStorageKey(key));
            r = v.length;
            out.set(v);
            return out;
        } finally {
            charge(RuntimeMethodFeeSchedule.DictDB_avm_getValue
                    + StorageFees.READ_PRICE_PER_BYTE * r);
        }
    }

    public void avm_addValue(V value) {
        int w = 0;
        try {
            ensureStorage();
            int sz = storage.getArrayLength(getIDHash());
            assumeValidValue(value);
            var v = ValueCodec.encodeValue(value);
            w = v.length;
            storage.setValue(getStorageKey(sz), v);
            storage.setArrayLength(getIDHash(), sz + 1);
        } finally {
            charge(RuntimeMethodFeeSchedule.ArrayDB_avm_addValue
                    + StorageFees.WRITE_PRICE_PER_BYTE * w);
        }
    }

    public void avm_setValue(int index, V value) {
        int w = 0;
        try {
            ensureStorage();
            assumeValidValue(value);
            var v = ValueCodec.encodeValue(value);
            w = v.length;
            int sz = storage.getArrayLength(getIDHash());
            if (index >= sz) {
                throw new InvalidDBAccessException();
            }
            storage.setValue(getStorageKey(sz), v);
        } finally {
            charge(RuntimeMethodFeeSchedule.ArrayDB_avm_addValue
                    + StorageFees.WRITE_PRICE_PER_BYTE * w);
        }
    }

    // pop, get, tryGet may return an instance different from the instance passed in addValue() or setValue() call.
    public void avm_removeLast() {
        try {
            ensureStorage();
            int sz = storage.getArrayLength(getIDHash());
            if (sz <= 0) {
                throw new InvalidDBAccessException();
            }
            storage.setValue(getStorageKey(sz), null);
            storage.setArrayLength(getIDHash(), sz - 1);
        } finally {
            charge(RuntimeMethodFeeSchedule.ArrayDB_avm_addValue);
        }
    }

    public V avm_popValue() {
        int r = 0;
        try {
            ensureStorage();
            int sz = storage.getArrayLength(getIDHash());
            if (sz <= 0) {
                throw new InvalidDBAccessException();
            }
            var v = storage.getValue(getStorageKey(sz - 1));
            r = v.length;
            V out = (V) ValueCodec.decodeValue(v);
            storage.setArrayLength(getIDHash(), sz - 1);
            return out;
        } finally {
            charge(RuntimeMethodFeeSchedule.ArrayDB_avm_addValue
                    + StorageFees.WRITE_PRICE_PER_BYTE * r);
        }
    }

    public PrimitiveBuffer avm_popValue(PrimitiveBuffer out) {
        int r = 0;
        try {
            ensureStorage();
            int sz = storage.getArrayLength(getIDHash());
            if (sz <= 0) {
                throw new InvalidDBAccessException();
            }
            var v = storage.getValue(getStorageKey(sz - 1));
            r = v.length;
            storage.setArrayLength(getIDHash(), sz - 1);
            if (out == null) {
                out = new PrimitiveBuffer();
            }
            out.set(v);
            return out;
        } finally {
            charge(RuntimeMethodFeeSchedule.ArrayDB_avm_addValue
                    + StorageFees.WRITE_PRICE_PER_BYTE * r);
        }
    }

    public V avm_getValue(int index) {
        int r = 0;
        try {
            ensureStorage();
            int sz = storage.getArrayLength(getIDHash());
            if (index >= sz || index < 0) {
                throw new InvalidDBAccessException();
            }
            var v = storage.getValue(getStorageKey(index));
            r = v.length;
            return (V) ValueCodec.decodeValue(v);
        } finally {
            charge(RuntimeMethodFeeSchedule.ArrayDB_avm_addValue
                    + StorageFees.WRITE_PRICE_PER_BYTE * r);
        }
    }

    public PrimitiveBuffer avm_getValue(int index, PrimitiveBuffer out) {
        int r = 0;
        try {
            ensureStorage();
            int sz = storage.getArrayLength(getIDHash());
            if (index >= sz || index < 0) {
                throw new InvalidDBAccessException();
            }
            var v = storage.getValue(getStorageKey(index));
            r = v.length;
            if (out == null) {
                out = new PrimitiveBuffer();
            }
            out.set(v);
            return out;
        } finally {
            charge(RuntimeMethodFeeSchedule.ArrayDB_avm_addValue
                    + StorageFees.WRITE_PRICE_PER_BYTE * r);
        }
    }

    public int avm_size() {
        ensureStorage();
        return storage.getArrayLength(getIDHash());
    }
}
