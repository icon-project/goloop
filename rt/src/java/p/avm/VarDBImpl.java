package p.avm;

import i.DBImplBase;
import i.IDBStorage;
import i.IObject;
import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.lang.String;

public class VarDBImpl extends DBImplBase implements VarDB {
    public VarDBImpl(String id) {
        super(TYPE_VAR_DB, id);
    }

    public VarDBImpl(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void _avm_set(IObject value) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.VarDB_avm_putValue);
        s.setTyped(getStorageKey(), value);
    }

    public void avm_set(Value value) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.VarDB_avm_putValue);
        s.setValue(getStorageKey(), value);
    }

    public IObject _avm_getTyped() {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.VarDB_avm_getValue);
        return s.getTyped(getStorageKey());
    }

    public Value avm_get(ValueBuffer out) {
        IDBStorage s = chargeAndGetDBStorage(RuntimeMethodFeeSchedule.VarDB_avm_getValue);
        return s.getValue(getStorageKey(), out);
    }

    public Value avm_get() {
        return avm_get(null);
    }
}
