package s.java.util;

import i.IObject;

public interface Map<K extends IObject, V extends IObject> extends IObject {

    // Query Operations

    int avm_size();

    boolean avm_isEmpty();

    boolean avm_containsKey(IObject key);

    boolean avm_containsValue(IObject value);

    V avm_get(IObject key);

    V avm_put(K key, V value);

    V avm_remove(IObject key);

    void avm_putAll(Map<? extends K, ? extends V> m);

    void avm_clear();

    // Views

    Set<K> avm_keySet();

    Collection<V> avm_values();

    Set<Map.Entry<K, V>> avm_entrySet();

    interface Entry<K extends IObject, V extends IObject> extends IObject {
        K avm_getKey();

        V avm_getValue();

        V avm_setValue(V value);

        boolean avm_equals(IObject o);

        int avm_hashCode();
    }

    boolean avm_equals(IObject o);

    int avm_hashCode();

    // Default
}
