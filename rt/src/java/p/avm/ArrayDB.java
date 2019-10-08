package p.avm;

public interface ArrayDB<E> {
    void avm_addValue(E value);

    void avm_setValue(int index, E value);

    void avm_removeLast();

    E avm_getValue(int index);

    PrimitiveBuffer avm_getValue(int index, PrimitiveBuffer out);

    int avm_size();

    E avm_popValue();

    PrimitiveBuffer avm_popValue(PrimitiveBuffer out);
}
