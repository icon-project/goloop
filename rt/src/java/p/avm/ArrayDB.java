package p.avm;

import i.IObject;

public interface ArrayDB {
    void avm_add(Value value);

    void avm_set(int index, Value value);

    void avm_removeLast();

    Value avm_get(int index, ValueBuffer out);

    Value avm_get(int index);

    int avm_size();

    Value avm_pop(ValueBuffer out);

    Value avm_pop();
}
