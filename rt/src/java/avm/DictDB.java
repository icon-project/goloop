package avm;

/**
 * Represents dict data structure.
 *
 * @param <K> Key type. It shall be integral wrapper type, BigInteger, String,
 *            Address or byte[].
 * @param <V> Value type.
 */
public interface DictDB<K, V> {
    /**
     * @param key
     * @param value
     */
    void putValue(K key, V value);

    /**
     * Returns Collection for the key. This method shall be called only if
     * type of V is DictDB or ArrayDB.
     *
     * @param key
     * @return
     */
    V get(K key);

    /**
     * @param key
     * @return
     */
    V getValue(K key);

    PrimitiveBuffer getValue(K key, PrimitiveBuffer out);
}
