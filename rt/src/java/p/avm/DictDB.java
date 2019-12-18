package p.avm;

import i.IObject;

public interface DictDB {
    void avm_set(IObject key, IObject value);
    IObject avm_get(IObject key);
    IObject avm_getOrDefault(IObject key, IObject defaultValue);
}

