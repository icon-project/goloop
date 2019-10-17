package p.avm;

import i.IObject;

public interface VarDB {
    void avm_set(Value value);

    Value avm_get(ValueBuffer out);

    Value avm_get();
}
