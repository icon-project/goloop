package s.java.util;

import a.ObjectArray;
import pi.UnmodifiableArrayMap;
import pi.UnmodifiableMapEntry;
import foundation.icon.ee.util.IObjects;
import i.IObject;
import i.IObjectArray;

import java.util.Objects;

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

    private static IObject[] verify(IObject[] kv) {
        for (int i = 0; i < kv.length; i += 2) {
            IObject k = Objects.requireNonNull(kv[i]);
            Objects.requireNonNull(kv[i + 1]);
            for (int j = 0; j < i; j++) {
                if (IObjects.indexOf(kv, k, 0, i, 2) >= 0)
                    throw new IllegalArgumentException("duplicated key");
            }
        }
        return kv;
    }

    static <K extends IObject, V extends IObject> Map<K, V> avm_of() {
        return UnmodifiableArrayMap.emptyMap();
    }

    static <K extends IObject, V extends IObject> Map<K, V> avm_of(K k1, V v1) {
        return new UnmodifiableArrayMap<>(verify(new IObject[]{k1, v1}));
    }

    static <K extends IObject, V extends IObject> Map<K, V> avm_of(
            K k1, V v1, K k2, V v2) {
        return new UnmodifiableArrayMap<>(verify(new IObject[]{k1, v1, k2, v2}));
    }

    static <K extends IObject, V extends IObject> Map<K, V> avm_of(
            K k1, V v1, K k2, V v2, K k3, V v3) {
        return new UnmodifiableArrayMap<>(verify(new IObject[]{k1, v1, k2, v2,
                k3, v3}));
    }

    static <K extends IObject, V extends IObject> Map<K, V> avm_of(
            K k1, V v1, K k2, V v2, K k3, V v3, K k4, V v4) {
        return new UnmodifiableArrayMap<>(verify(new IObject[]{k1, v1, k2, v2,
                k3, v3, k4, v4}));
    }

    static <K extends IObject, V extends IObject> Map<K, V> avm_of(
            K k1, V v1, K k2, V v2, K k3, V v3, K k4, V v4, K k5, V v5) {
        return new UnmodifiableArrayMap<>(verify(new IObject[]{k1, v1, k2, v2,
                k3, v3, k4, v4, k5, v5}));
    }

    static <K extends IObject, V extends IObject> Map<K, V> avm_of(
            K k1, V v1, K k2, V v2, K k3, V v3, K k4, V v4, K k5, V v5,
            K k6, V v6) {
        return new UnmodifiableArrayMap<>(verify(new IObject[]{k1, v1, k2, v2,
                k3, v3, k4, v4, k5, v5, k6, v6}));
    }

    static <K extends IObject, V extends IObject> Map<K, V> avm_of(
            K k1, V v1, K k2, V v2, K k3, V v3, K k4, V v4, K k5, V v5,
            K k6, V v6, K k7, V v7) {
        return new UnmodifiableArrayMap<>(verify(new IObject[]{k1, v1, k2, v2,
                k3, v3, k4, v4, k5, v5, k6, v6, k7, v7}));
    }

    static <K extends IObject, V extends IObject> Map<K, V> avm_of(
            K k1, V v1, K k2, V v2, K k3, V v3, K k4, V v4, K k5, V v5,
            K k6, V v6, K k7, V v7, K k8, V v8) {
        return new UnmodifiableArrayMap<>(verify(new IObject[]{k1, v1, k2, v2,
                k3, v3, k4, v4, k5, v5, k6, v6, k7, v7, k8, v8}));
    }

    static <K extends IObject, V extends IObject> Map<K, V> avm_of(
            K k1, V v1, K k2, V v2, K k3, V v3, K k4, V v4, K k5, V v5,
            K k6, V v6, K k7, V v7, K k8, V v8, K k9, V v9) {
        return new UnmodifiableArrayMap<>(verify(new IObject[]{k1, v1, k2, v2,
                k3, v3, k4, v4, k5, v5, k6, v6, k7, v7, k8, v8, k9, v9}));
    }

    static <K extends IObject, V extends IObject> Map<K, V> avm_of(
            K k1, V v1, K k2, V v2, K k3, V v3, K k4, V v4, K k5, V v5,
            K k6, V v6, K k7, V v7, K k8, V v8, K k9, V v9, K k10, V v10) {
        return new UnmodifiableArrayMap<>(verify(new IObject[]{k1, v1, k2, v2,
                k3, v3, k4, v4, k5, v5, k6, v6, k7, v7, k8, v8, k9, v9,
                k10, v10}));
    }

    static <K extends IObject, V extends IObject> Map<K, V> avm_ofEntries(
            IObjectArray entries) {
        var oa = ((ObjectArray) entries).getUnderlying();
        var keyValue = new IObject[oa.length * 2];
        int dst = 0;
        for (Object o : oa) {
            var entry = (Entry<?, ?>) o;
            keyValue[dst++] = entry.avm_getKey();
            keyValue[dst++] = entry.avm_getValue();
        }
        return new UnmodifiableArrayMap<>(verify(keyValue));
    }

    static <K extends IObject, V extends IObject> Entry<K, V> avm_entry(K k, V v) {
        Objects.requireNonNull(k);
        Objects.requireNonNull(v);
        return new UnmodifiableMapEntry<>(k, v);
    }
}
