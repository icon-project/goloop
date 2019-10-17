package p.avm;

import i.IObject;

public interface VarDB {
    void avm_set(IObject value);

    IObject avm_get();

    ValueBuffer avm_get(ValueBuffer out);
}
