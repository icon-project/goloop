package p.avm;

import i.IObject;

public interface DictDB {
    void avm_set(IObject key, IObject value);

    IObject avm_at(IObject key);

    IObject avm_get(IObject key);

    ValueBuffer avm_get(IObject key, ValueBuffer out);
}

