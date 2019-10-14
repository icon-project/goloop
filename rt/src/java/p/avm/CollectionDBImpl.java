package p.avm;

import i.*;
import org.aion.avm.RuntimeMethodFeeSchedule;
import org.aion.avm.embed.hash.HashUtils;
import s.java.lang.String;

public class CollectionDBImpl extends DBImplBase implements DictDB, ArrayDB {
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
    public void avm_putValue(IObject key, IObject value) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.DictDB_avm_putValue);
        s.setValue(getStorageKey(key), value);
    }

    public IObject avm_get(IObject key) {
        IInstrumentation.attachedThreadInstrumentation.get()
                .chargeEnergy(RuntimeMethodFeeSchedule.DictDB_avm_get);
        return (IObject) new CollectionDBImpl(getSubDBID(key));
    }

    public IObject avm_getValue(IObject key) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.DictDB_avm_getValue);
        return s.getValue(getStorageKey(key));
    }

    public PrimitiveBuffer avm_getValue(IObject key, PrimitiveBuffer out) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.DictDB_avm_getValue);
        return s.getValue(getStorageKey(key), out);
    }

    public void avm_addValue(IObject value) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_addValue);
        int sz = s.getArrayLength(getStorageKey());
        s.setValue(getStorageKey(sz), value);
        s.setArrayLength(getStorageKey(), sz + 1);
    }

    public void avm_setValue(int index, IObject value) {
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

    public IObject avm_popValue() {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_popValue);
        int sz = s.getArrayLength(getStorageKey());
        if (sz <= 0) {
            throw new InvalidDBAccessException();
        }
        var v = s.getValue(getStorageKey(sz - 1));
        s.setArrayLength(getStorageKey(), sz - 1);
        return v;
    }

    public PrimitiveBuffer avm_popValue(PrimitiveBuffer out) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_popValue);
        int sz = s.getArrayLength(getStorageKey());
        if (sz <= 0) {
            throw new InvalidDBAccessException();
        }
        var o = s.getValue(getStorageKey(sz - 1), out);
        s.setArrayLength(getStorageKey(), sz - 1);
        return o;
    }

    public IObject avm_getValue(int index) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_getValue);
        int sz = s.getArrayLength(getStorageKey());
        if (index >= sz || index < 0) {
            throw new InvalidDBAccessException();
        }
        return s.getValue(getStorageKey(index));
    }

    public PrimitiveBuffer avm_getValue(int index, PrimitiveBuffer out) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_getValue);
        int sz = s.getArrayLength(getStorageKey());
        if (index >= sz || index < 0) {
            throw new InvalidDBAccessException();
        }
        return s.getValue(getStorageKey(index), out);
    }

    public int avm_size() {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.ArrayDB_avm_size);
        return s.getArrayLength(getStorageKey());
    }
}
