package avm;

public interface ArrayDB<E> {
    void add(E value);

    void set(int index, E value);

    void removeLast();

    E get(int index);

    PrimitiveBuffer get(int index, PrimitiveBuffer out);

    int size();

    // Do not shrink if decoding fails.
    E pop();

    PrimitiveBuffer pop(PrimitiveBuffer out);

//    E at(int index);
//    E addCollection();
}
