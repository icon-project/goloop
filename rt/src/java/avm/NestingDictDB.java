package avm;

public interface NestingDictDB<K, V> {
    /**
     * Returns Collection for the key. This method shall be called only if
     * type of V is DictDB or ArrayDB.
     *
     * @param key
     * @return
     */
    V at(K key);
}
