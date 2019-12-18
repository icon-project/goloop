package p.avm;

import i.DBImplBase;
import i.IDBStorage;
import i.IObject;
import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.lang.Class;
import s.java.lang.String;

public class VarDBImpl extends DBImplBase implements VarDB {
    public VarDBImpl(String id, Class<?> vc) {
        super(TYPE_VAR_DB, id, vc);
    }

    public VarDBImpl(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void _avm_set(IObject value) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.VarDB_avm_putValue);
        s.setTyped(getStorageKey(), value);
    }

    public void avm_set(IObject value) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.VarDB_avm_putValue);
        s.setBytes(getStorageKey(), encode(value));
    }

    public IObject _avm_getTyped() {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.VarDB_avm_getValue);
        return s.getTyped(getStorageKey());
    }

    public IObject avm_get() {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.VarDB_avm_getValue);
        return decode(s.getBytes(getStorageKey()));
    }

    public IObject avm_getOrDefault(IObject defaultValue) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.VarDB_avm_getValue);
        var out = decode(s.getBytes(getStorageKey()));
        return (out != null) ? out : defaultValue;
    }
}
