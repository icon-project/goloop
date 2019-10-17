package p.avm;

import i.IObject;

public interface DictDB {
    void avm_set(IObject key, Value value);

    Value avm_get(IObject key, ValueBuffer out);

    Value avm_get(IObject key);
}

