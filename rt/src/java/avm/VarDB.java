package avm;

public interface VarDB<V> {
    void set(V value);

    V get();

    PrimitiveBuffer get(PrimitiveBuffer out);
}
