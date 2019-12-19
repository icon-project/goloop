package avm;

public interface VarDB<E> {
    void set(E value);
    E get();
    E getOrDefault(E defaultValue);
}
