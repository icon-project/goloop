package p.avm;

import i.*;
import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.lang.String;

public class CollectionDBImpl extends DBImplBase implements CollectionDB {
    public CollectionDBImpl(String id) {
        super(id);
    }

    public CollectionDBImpl(byte[] id) {
        super(id);
    }

    public CollectionDBImpl(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    /**
     * @param key
     * @param value
     * @throws InvalidDBAccessException if key is null,
     */
    public void _avm_set(IObject key, IObject value) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.DictDB_avm_putValue);
        s.setTyped(getStorageKey(key), value);
    }

    public void avm_set(IObject key, Value value) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.DictDB_avm_putValue);
        s.setValue(getStorageKey(key), value);
    }

    public IObject avm_at(IObject key) {
        IInstrumentation.attachedThreadInstrumentation.get()
                .chargeEnergy(RuntimeMethodFeeSchedule.DictDB_avm_get);
        return (IObject) new CollectionDBImpl(getSubDBID(key));
    }

    public IObject _avm_getTyped(IObject key) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.DictDB_avm_getValue);
        return s.getTyped(getStorageKey(key));
    }

    public Value avm_get(IObject key) {
        return avm_get(key, null);
    }

    public Value avm_get(IObject key, ValueBuffer out) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.DictDB_avm_getValue);
        return s.getValue(getStorageKey(key), out);
    }

    public void _avm_add(IObject value) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_addValue);
        int sz = s.getArrayLength(getStorageKey());
        s.setTyped(getStorageKey(sz), value);
        s.setArrayLength(getStorageKey(), sz + 1);
    }

    public void avm_add(Value value) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_addValue);
        int sz = s.getArrayLength(getStorageKey());
        s.setValue(getStorageKey(sz), value);
        s.setArrayLength(getStorageKey(), sz + 1);
    }

    public void _avm_set(int index, IObject value) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_setValue);
        int sz = s.getArrayLength(getStorageKey());
        if (index >= sz) {
            throw new InvalidDBAccessException();
        }
        s.setTyped(getStorageKey(sz), value);
    }

    public void avm_set(int index, Value value) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_setValue);
        int sz = s.getArrayLength(getStorageKey());
        if (index >= sz) {
            throw new InvalidDBAccessException();
        }
        s.setValue(getStorageKey(sz), value);
    }

    public void avm_removeLast() {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_removeLast);
        int sz = s.getArrayLength(getStorageKey());
        if (sz <= 0) {
            throw new InvalidDBAccessException();
        }
        s.setValue(getStorageKey(sz), null);
        s.setArrayLength(getStorageKey(), sz - 1);
    }

    public IObject _avm_popTyped() {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_popValue);
        int sz = s.getArrayLength(getStorageKey());
        if (sz <= 0) {
            throw new InvalidDBAccessException();
        }
        var v = s.getTyped(getStorageKey(sz - 1));
        s.setArrayLength(getStorageKey(), sz - 1);
        return v;
    }

    public Value avm_pop(ValueBuffer out) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_popValue);
        int sz = s.getArrayLength(getStorageKey());
        if (sz <= 0) {
            throw new InvalidDBAccessException();
        }
        var o = s.getValue(getStorageKey(sz - 1), out);
        s.setArrayLength(getStorageKey(), sz - 1);
        return o;
    }

    public Value avm_pop() {
        return avm_pop(null);
    }

    public IObject _avm_getTyped(int index) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_getValue);
        int sz = s.getArrayLength(getStorageKey());
        if (index >= sz || index < 0) {
            throw new InvalidDBAccessException();
        }
        return s.getTyped(getStorageKey(index));
    }

    public Value avm_get(int index, ValueBuffer out) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_getValue);
        int sz = s.getArrayLength(getStorageKey());
        if (index >= sz || index < 0) {
            throw new InvalidDBAccessException();
        }
        return s.getValue(getStorageKey(index), out);
    }

    public Value avm_get(int index) {
        return avm_get(index, null);
    }

    public int avm_size() {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_size);
        return s.getArrayLength(getStorageKey());
    }
}
