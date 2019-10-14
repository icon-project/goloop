package p.avm;

import i.IObject;

public interface ArrayDB {
    void avm_addValue(IObject value);

    void avm_setValue(int index, IObject value);

    void avm_removeLast();

    IObject avm_getValue(int index);

    PrimitiveBuffer avm_getValue(int index, PrimitiveBuffer out);

    int avm_size();

    IObject avm_popValue();

    PrimitiveBuffer avm_popValue(PrimitiveBuffer out);
}
