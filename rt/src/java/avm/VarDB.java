package avm;

public interface VarDB<V> {
    void putValue(V value);

    V getValue();

    PrimitiveBuffer getValue(PrimitiveBuffer out);
}
