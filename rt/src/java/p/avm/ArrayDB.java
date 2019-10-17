package p.avm;

import i.IObject;

public interface ArrayDB {
    void avm_add(IObject value);

    void avm_set(int index, IObject value);

    void avm_removeLast();

    IObject avm_get(int index);

    ValueBuffer avm_get(int index, ValueBuffer out);

    int avm_size();

    IObject avm_pop();

    ValueBuffer avm_pop(ValueBuffer out);
}
