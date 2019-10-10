package p.avm;

public interface VarDB<V> {
    void avm_putValue(V value);

    V avm_getValue();

    PrimitiveBuffer avm_getValue(PrimitiveBuffer out);
}
