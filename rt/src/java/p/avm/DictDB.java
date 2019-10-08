package p.avm;

public interface DictDB<K, V> {
    void avm_putValue(K key, V value);

    V avm_get(K key);

    V avm_getValue(K key);

    PrimitiveBuffer avm_getValue(K key, PrimitiveBuffer out);
}

