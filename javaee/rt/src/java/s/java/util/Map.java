package s.java.util;

import i.IObject;

public interface Map<K, V> extends IObject {

    // Query Operations

    int avm_size();

    boolean avm_isEmpty();

    boolean avm_containsKey(IObject key);

    boolean avm_containsValue(IObject value);

    IObject avm_get(IObject key);

    IObject avm_put(IObject key, IObject value);

    IObject avm_remove(IObject key);

    void avm_putAll(Map<? extends K, ? extends V> m);

    void avm_clear();

    // Views

    Set<K> avm_keySet();

    Collection<V> avm_values();

    Set<Map.Entry<K, V>> avm_entrySet();

    interface Entry<K, V> extends IObject {
        IObject avm_getKey();

        IObject avm_getValue();

        IObject avm_setValue(IObject value);

        boolean avm_equals(IObject o);

        int avm_hashCode();
    }

    boolean avm_equals(IObject o);

    int avm_hashCode();

    // Default
}
