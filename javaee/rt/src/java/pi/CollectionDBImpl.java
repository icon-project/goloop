package pi;

import i.*;
import org.aion.avm.RuntimeMethodFeeSchedule;
import p.avm.CollectionDB;
import s.java.lang.Class;
import s.java.lang.String;

public class CollectionDBImpl extends DBImplBase implements CollectionDB {
    public CollectionDBImpl(int type, String id, Class<?> vc) {
        super(type, id, vc);
    }

    public CollectionDBImpl(byte[] id, Class<?> vc) {
        super(id, vc);
    }

    public CollectionDBImpl(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    /**
     * @param key
     * @param value
     */
    public void _avm_set(IObject key, IObject value) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.DictDB_avm_putValue);
        s.setTyped(getStorageKey(key), value);
    }

    public void avm_set(IObject key, IObject value) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.DictDB_avm_putValue);
        s.setBytes(getStorageKey(key), encode(value));
    }

    public IObject avm_at(IObject key) {
        IInstrumentation.attachedThreadInstrumentation.get()
                .chargeEnergy(RuntimeMethodFeeSchedule.DictDB_avm_get);
        return (IObject) new CollectionDBImpl(getSubDBID(key), leafValue);
    }

    public IObject _avm_getTyped(IObject key) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.DictDB_avm_getValue);
        return s.getTyped(getStorageKey(key));
    }

    public IObject avm_get(IObject key) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.DictDB_avm_getValue);
        return decode(s.getBytes(getStorageKey(key)));
    }

    public IObject avm_getOrDefault(IObject key, IObject defaultValue) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.DictDB_avm_getValue);
        var out = decode(s.getBytes(getStorageKey(key)));
        return (out != null) ? out : defaultValue;
    }

    public void _avm_add(IObject value) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_addValue);
        int sz = s.getArrayLength(getStorageKey());
        s.setTyped(getStorageKey(sz), value);
        s.setArrayLength(getStorageKey(), sz + 1);
    }

    public void avm_add(IObject value) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_addValue);
        int sz = s.getArrayLength(getStorageKey());
        s.setBytes(getStorageKey(sz), encode(value));
        s.setArrayLength(getStorageKey(), sz + 1);
    }

    public void _avm_set(int index, IObject value) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_setValue);
        int sz = s.getArrayLength(getStorageKey());
        if (index >= sz) {
            throw new IllegalArgumentException();
        }
        s.setTyped(getStorageKey(sz), value);
    }

    public void avm_set(int index, IObject value) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_setValue);
        int sz = s.getArrayLength(getStorageKey());
        if (index >= sz) {
            throw new IllegalArgumentException();
        }
        s.setBytes(getStorageKey(sz), encode(value));
    }

    public void avm_removeLast() {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_removeLast);
        int sz = s.getArrayLength(getStorageKey());
        if (sz <= 0) {
            throw new IllegalArgumentException();
        }
        s.setBytes(getStorageKey(sz), null);
        s.setArrayLength(getStorageKey(), sz - 1);
    }

    public IObject _avm_popTyped() {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_popValue);
        int sz = s.getArrayLength(getStorageKey());
        if (sz <= 0) {
            throw new IllegalArgumentException();
        }
        var v = s.getTyped(getStorageKey(sz - 1));
        s.setArrayLength(getStorageKey(), sz - 1);
        return v;
    }

    public IObject avm_pop() {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_popValue);
        int sz = s.getArrayLength(getStorageKey());
        if (sz <= 0) {
            throw new IllegalArgumentException();
        }
        var o = decode(s.getBytes(getStorageKey(sz - 1)));
        s.setArrayLength(getStorageKey(), sz - 1);
        return o;
    }

    public IObject _avm_getTyped(int index) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_getValue);
        int sz = s.getArrayLength(getStorageKey());
        if (index >= sz || index < 0) {
            throw new IllegalArgumentException();
        }
        return s.getTyped(getStorageKey(index));
    }

    public IObject avm_get(int index) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_getValue);
        int sz = s.getArrayLength(getStorageKey());
        if (index >= sz || index < 0) {
            throw new IllegalArgumentException();
        }
        return decode(s.getBytes(getStorageKey(index)));
    }

    public int avm_size() {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_size);
        return s.getArrayLength(getStorageKey());
    }
}
