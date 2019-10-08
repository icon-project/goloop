package avm;

public interface ArrayDB<E> {
    void addValue(E value);

    void setValue(int index, E value);

    void removeLast();

    E getValue(int index);

    PrimitiveBuffer getValue(int index, PrimitiveBuffer out);

    int size();

    // Do not shrink if decoding fails.
    E popValue();

    PrimitiveBuffer popValue(PrimitiveBuffer out);

//    E get(int index);
//    E add();
}
