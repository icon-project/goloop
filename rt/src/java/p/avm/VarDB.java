package p.avm;

import i.IObject;

public interface VarDB {
    void avm_set(IObject value);

    IObject avm_get();

    PrimitiveBuffer avm_get(PrimitiveBuffer out);
}
