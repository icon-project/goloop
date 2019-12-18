package avm;

public interface ArrayDB<E> {
    void add(E value);
    void set(int index, E value);
    void removeLast();
    E get(int index);
    int size();
    E pop();
}
