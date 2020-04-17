package pi;

import i.*;
import org.aion.avm.RuntimeMethodFeeSchedule;
import p.score.CollectionDB;
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

    public void avm_set(IObject key, IObject value) {
        getDBStorage().setBytes(getStorageKey(key), encode(value));
    }

    public IObject avm_at(IObject key) {
        IInstrumentation.attachedThreadInstrumentation.get()
                .chargeEnergy(RuntimeMethodFeeSchedule.DictDB_avm_at);
        return new CollectionDBImpl(getSubDBID(key), leafValue);
    }

    public IObject avm_get(IObject key) {
        return decode(getDBStorage().getBytes(getStorageKey(key)));
    }

    public IObject avm_getOrDefault(IObject key, IObject defaultValue) {
        var out = decode(getDBStorage().getBytes(getStorageKey(key)));
        return (out != null) ? out : defaultValue;
    }

    public void avm_add(IObject value) {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey());
        s.setBytes(getStorageKey(sz), encode(value));
        s.setArrayLength(getStorageKey(), sz + 1);
    }

    public void avm_set(int index, IObject value) {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey());
        if (index >= sz) {
            throw new IllegalArgumentException();
        }
        s.setBytes(getStorageKey(index), encode(value));
    }

    public void avm_removeLast() {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey());
        if (sz <= 0) {
            throw new IllegalArgumentException();
        }
        s.setBytes(getStorageKey(sz - 1), null);
        s.setArrayLength(getStorageKey(), sz - 1);
    }

    public IObject avm_pop() {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey());
        if (sz <= 0) {
            throw new IllegalArgumentException();
        }
        var o = decode(s.getBytes(getStorageKey(sz - 1)));
        s.setBytes(getStorageKey(sz - 1), null);
        s.setArrayLength(getStorageKey(), sz - 1);
        return o;
    }

    public IObject avm_get(int index) {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey());
        if (index >= sz || index < 0) {
            throw new IllegalArgumentException();
        }
        return decode(s.getBytes(getStorageKey(index)));
    }

    public int avm_size() {
        return getDBStorage().getArrayLength(getStorageKey());
    }
}
