package p.avm;

import i.IObject;

public interface VarDB {
    void avm_putValue(IObject value);

    IObject avm_getValue();

    PrimitiveBuffer avm_getValue(PrimitiveBuffer out);
}
