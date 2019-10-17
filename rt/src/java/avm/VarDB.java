package avm;

public interface VarDB<V> {
    void set(V value);

    V get();

    ValueBuffer get(ValueBuffer out);
}
