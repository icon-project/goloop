package avm;

public interface ArrayDB<E> {
    void add(E value);

    void set(int index, E value);

    void removeLast();

    E get(int index);

    ValueBuffer get(int index, ValueBuffer out);

    int size();

    // Do not shrink if decoding fails.
    E pop();

    ValueBuffer pop(ValueBuffer out);

//    E at(int index);
//    E addCollection();
}
