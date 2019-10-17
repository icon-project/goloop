package p.avm;

import i.DBImplBase;
import i.IDBStorage;
import i.IObject;
import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.lang.String;

public class VarDBImpl extends DBImplBase implements VarDB {
    public VarDBImpl(String id) {
        super(id);
    }

    public VarDBImpl(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void avm_set(IObject value) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.VarDB_avm_putValue);
        s.setValue(getStorageKey(), value);
    }

    public IObject avm_get() {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.VarDB_avm_getValue);
        return s.getValue(getStorageKey());
    }

    public ValueBuffer avm_get(ValueBuffer out) {
        IDBStorage s = getDBStorage(RuntimeMethodFeeSchedule.VarDB_avm_getValue);
        return s.getValue(getStorageKey(), out);
    }
}
