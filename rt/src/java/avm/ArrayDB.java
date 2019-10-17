package avm;

public interface ArrayDB {
    void add(Value value);

    void set(int index, Value value);

    void removeLast();

    Value get(int index, ValueBuffer out);

    Value get(int index);

    int size();

    // Do not shrink if decoding fails.
    Value pop(ValueBuffer out);

    Value pop();
}
