package p.avm;

import i.IObject;

public interface DictDB {
    void avm_putValue(IObject key, IObject value);

    IObject avm_get(IObject key);

    IObject avm_getValue(IObject key);

    PrimitiveBuffer avm_getValue(IObject key, PrimitiveBuffer out);
}

