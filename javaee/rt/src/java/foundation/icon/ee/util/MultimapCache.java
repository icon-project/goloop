package foundation.icon.ee.util;

import java.lang.ref.ReferenceQueue;
import java.lang.ref.SoftReference;
import java.lang.ref.WeakReference;
import java.util.function.Predicate;

public abstract class MultimapCache<K, V> {
    private interface Entry<T> extends Disposable {
        T get();
    }

    private class EntryMap extends LinkedHashMultimap<K, Entry<V>> {
        @Override
        protected boolean removeEldestEntry(K key, Entry<V> value) {
            return entryMap.size() > cap;
        }
    }

    private static final int DEFAULT_CAP = 256;

    private final DisposableReferenceQueue<V> refQueue =
            new DisposableReferenceQueue<>();
    protected final EntryMap entryMap = new EntryMap();
    private final int cap;

    public MultimapCache() {
        this(DEFAULT_CAP);
    }

    public MultimapCache(int cap) {
        this.cap = cap;
    }

    public V remove(K k, Predicate<V> selector) {
        synchronized (entryMap) {
            var entry = entryMap.remove(k, set-> {
                Entry<V> any = null;
                for (var ref : set) {
                    var da = (ref != null) ? ref.get() : null;
                    if (da != null) {
                        if (selector.test(da)) {
                            return ref;
                        }
                        any = ref;
                    } else {
                        if (any == null) {
                            any = ref;
                        }
                    }
                }
                return any;
            });
            return entry!=null ? entry.get() : null;
        }
    }

    public void put(K k, V v) {
        synchronized (entryMap) {
            entryMap.put(k, newEntry(k, v, refQueue));
        }
    }

    public int size() {
        synchronized (entryMap) {
            return entryMap.size();
        }
    }

    // cleans up unreachable referent
    public void gc() {
        refQueue.consumeAll();
    }

    protected Entry<V> newEntry(K k, V v, ReferenceQueue<V> q) {
        return null;
    }

    public static<K, V> MultimapCache<K, V> newWeakCache(int cap) {
        return new MultimapCache<>(cap) {
            class WeakEntry extends WeakReference<V> implements Entry<V> {
                private final K key;

                public WeakEntry(V referent, ReferenceQueue<? super V> q, K key) {
                    super(referent, q);
                    this.key = key;
                }

                public void dispose() {
                    synchronized (entryMap) {
                        entryMap.remove(key, this);
                    }
                }
            }

            @Override
            protected Entry<V> newEntry(K k, V v, ReferenceQueue<V> q) {
                return new WeakEntry(v, q, k);
            }
        };
    }

    public static<K, V> MultimapCache<K, V> newSoftCache(int cap) {
        return new MultimapCache<>(cap) {
            class SoftEntry extends SoftReference<V> implements Entry<V> {
                private final K key;

                public SoftEntry(V referent, ReferenceQueue<? super V> q, K key) {
                    super(referent, q);
                    this.key = key;
                }

                public void dispose() {
                    synchronized (entryMap) {
                        entryMap.remove(key, this);
                    }
                }
            }

            @Override
            protected Entry<V> newEntry(K k, V v, ReferenceQueue<V> q) {
                return new SoftEntry(v, q, k);
            }
        };
    }
}
