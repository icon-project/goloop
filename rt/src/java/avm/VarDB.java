package avm;

public interface VarDB {
    void set(Value value);

    Value get(ValueBuffer out);

    Value get();
}
