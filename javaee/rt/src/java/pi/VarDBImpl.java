package pi;

import i.DBImplBase;
import i.IObject;
import p.score.VarDB;
import s.java.lang.Class;
import s.java.lang.String;

public class VarDBImpl extends DBImplBase implements VarDB {
    public VarDBImpl(String id, Class<?> vc) {
        super(TYPE_VAR_DB, id, vc);
    }

    public VarDBImpl(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void avm_set(IObject value) {
        getDBStorage().setBytes(getStorageKey(), encode(value));
    }

    public IObject avm_get() {
        return decode(getDBStorage().getBytes(getStorageKey()));
    }

    public IObject avm_getOrDefault(IObject defaultValue) {
        var out = decode(getDBStorage().getBytes(getStorageKey()));
        return (out != null) ? out : defaultValue;
    }
}
