package avm;

public interface DictDB<K, V> {
    void set(K key, V value);
    V get(K key);
    V getOrDefault(K key, V defaultValue);
}
