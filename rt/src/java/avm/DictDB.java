package avm;

/**
 * Represents dict data structure.
 *
 * @param <K> Key type. It shall be integral wrapper type, BigInteger, String,
 *            Address or byte[].
 */
public interface DictDB<K> {
    /**
     * @param key
     * @param value
     */
    void set(K key, Value value);

    /**
     * @param key
     * @return
     */
    Value get(K key, ValueBuffer out);
    Value get(K key);
}
