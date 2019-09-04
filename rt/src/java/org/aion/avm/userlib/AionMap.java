package org.aion.avm.userlib;

import java.util.*;

/**
 * Hash table based implementation of the {@code Map} interface.
 *
 * <p>This implementation provides all of the optional map operations, and permits {@code null} values.
 * {@code null} key will be rejected for this map.
 *
 * <p>An instance of {@code AionMap} has two parameters that affect its performance: <i>initial capacity</i> and <i>load factor</i>.
 * <i>capacity</i> is the number of buckets in the hash table, and <i>load factor</i> is a measure of how full the hash table is
 * allowed to get before its capacity is automatically increased.  Threshold for initiating a <i>rehashing</i> operation is calculated
 * as the product of the load factor and the current capacity. When the number of hash table entries exceed this threshold, the capacity is
 * doubled and the internal structure is rebuilt by rehashing all the keys.
 */
public class AionMap<K, V> implements Map<K, V> {

    /**
     * The entry table, resized as necessary.
     * Entries are places in the table based on key's hashcode modulo table size
     */
    private AionMapEntry[] entryTable;

    /**
     * The current size of the map
     */
    private int size;

    /**
     * The load factor for the hash table. It indicates how full the table has to be, before increasing its size.
     * A loadFactor of 1.0 indicates a full table, where the entry count equals its capacity.
     * Note that this does not reflect how full buckets are; entries can be placed in any bucket.
     */
    private float loadFactor;

    /**
     * The next size value at which to resize (capacity * load factor)
     */
    private int threshold;

    /**
     * The number of times this AionMap has been structurally modified.
     * Structural modifications are those that change the number of mappings
     */
    private int modCount;

    /**
     * Constructs an empty {@code AionMap}.
     *
     * @param initialCapacity The initial initialCapacity of the {@code AionMap}
     * @param loadFactor      The load factor for the hash table.
     */
    public AionMap(int initialCapacity, float loadFactor) {
        this.loadFactor = loadFactor;
        threshold = (int) (initialCapacity * loadFactor);
        entryTable = new AionMapEntry[initialCapacity];
    }

    /**
     * Constructs an empty {@code AionMap} with the default capacity (16) and load factor (0.75).
     */
    public AionMap() {
        int initialCapacity = 16;
        loadFactor = 0.75f;
        threshold = (int) (initialCapacity * loadFactor);
        entryTable = new AionMapEntry[initialCapacity];
    }

    /**
     * Returns the number of key-value mappings in this map.
     *
     * @return the number of key-value mappings in this map
     */
    @Override
    public int size() {
        return size;
    }

    /**
     * Returns {@code true} if this map contains no key-value mappings.
     *
     * @return {@code true} if this map contains no key-value mappings
     */
    @Override
    public boolean isEmpty() {
        return this.size == 0;
    }

    /**
     * Returns {@code true} if this map contains a mapping for the
     * specified key.
     *
     * @param key The key whose presence in this map is to be tested
     * @return {@code true} if this map contains a mapping for the specified
     * key.
     * @throws NullPointerException if the specified key is null
     */
    @SuppressWarnings("unchecked")
    @Override
    public boolean containsKey(Object key) {
        keyNullCheck(key);
        int keyHashcode = key.hashCode();

        int hashValue = hashValue(keyHashcode, entryTable.length);
        if (entryTable[hashValue] == null) {
            return false;
        } else {
            AionMapEntry current = entryTable[hashValue];
            while (current != null) {
                if (current.keyHashcode == keyHashcode && current.key.equals(key)) {
                    return true;
                }
                current = current.next;
            }
        }
        return false;
    }

    /**
     * Returns {@code true} if this map maps one or more keys to the
     * specified value.
     *
     * @param value value whose presence in this map is to be tested
     * @return {@code true} if this map maps one or more keys to the
     * specified value
     */
    @Override
    public boolean containsValue(Object value) {
        AionMapEntry[] tab = entryTable;
        int length = tab.length;
        if (value == null) {
            for (int i = 0; i < length; i++) {
                for (AionMapEntry e = tab[i]; e != null; e = e.next) {
                    if (null == e.value) {
                        return true;
                    }
                }
            }
        } else {
            for (int i = 0; i < length; i++) {
                for (AionMapEntry e = tab[i]; e != null; e = e.next) {
                    if (value.equals(e.value)) {
                        return true;
                    }
                }
            }
        }
        return false;
    }

    /**
     * Returns the value to which the specified key is mapped,
     * or {@code null} if this map contains no mapping for the key.
     *
     * <p>A return value of {@code null} does not <i>necessarily</i>
     * indicate that the map contains no mapping for the key; it's also
     * possible that the map explicitly maps the key to {@code null}.
     * The {@link #containsKey containsKey} operation may be used to
     * distinguish these two cases.
     *
     * @throws NullPointerException if the specified key is null
     */
    @SuppressWarnings("unchecked")
    @Override
    public V get(Object key) {
        keyNullCheck(key);
        int keyHashcode = key.hashCode();
        int hashValue = hashValue(keyHashcode, entryTable.length);

        if (entryTable[hashValue] == null) {
            return null;
        } else {
            AionMapEntry current = entryTable[hashValue];
            while (current != null) {
                if (current.keyHashcode == keyHashcode && current.key.equals(key)) {
                    return (V) current.value;
                }
                current = current.next;
            }
        }
        return null;
    }

    /**
     * Associates the specified value with the specified key in this map.
     * If the map previously contained a mapping for the key, the old
     * value is replaced.
     *
     * @param key   key with which the specified value is to be associated
     * @param value value to be associated with the specified key
     * @return the previous value associated with {@code key}, or
     * {@code null} if there was no mapping for {@code key}.
     * (A {@code null} return can also indicate that the map
     * previously associated {@code null} with {@code key}.)
     * @throws NullPointerException if the specified key is null
     */
    @SuppressWarnings("unchecked")
    @Override
    public V put(K key, V value) {
        keyNullCheck(key);
        int newKeyHashcode = key.hashCode();
        int hashValue = hashValue(newKeyHashcode, entryTable.length);

        if (entryTable[hashValue] == null) {
            entryTable[hashValue] = new AionMapEntry(key, value);
        } else {
            AionMapEntry previous = null;
            AionMapEntry current = entryTable[hashValue];
            while (current != null) {
                // need to traverse the whole list in case of an overwrite
                if (current.keyHashcode == newKeyHashcode && current.key.equals(key)) {
                    return (V) current.setValue(value);
                }
                previous = current;
                current = current.next;
            }
            previous.next = new AionMapEntry(key, value);
        }

        size++;
        modCount++;

        if (size >= threshold) {
            resize(2 * entryTable.length);
        }

        return null;
    }

    /**
     * Removes the mapping for the specified key from this map if present.
     *
     * @param key key whose mapping is to be removed from the map
     * @return the previous value associated with {@code key}, or
     * {@code null} if there was no mapping for {@code key}.
     * (A {@code null} return can also indicate that the map
     * previously associated {@code null} with {@code key}.)
     * @throws NullPointerException if the specified key is null
     */
    @SuppressWarnings("unchecked")
    @Override
    public V remove(Object key) {
        keyNullCheck(key);
        int keyHashcode = key.hashCode();
        int hashValue = hashValue(keyHashcode, entryTable.length);

        if (entryTable[hashValue] == null) {
            return null;
        } else {
            AionMapEntry previous = null;
            AionMapEntry current = entryTable[hashValue];
            while (current != null) {
                if (current.keyHashcode == keyHashcode && current.key.equals(key)) {
                    V currentValue = (V) current.value;
                    if (previous == null) {
                        entryTable[hashValue] = entryTable[hashValue].next;
                    } else {
                        previous.next = current.next;
                    }
                    modCount++;
                    size--;
                    return currentValue;
                }
                previous = current;
                current = current.next;
            }
            return null;
        }
    }

    /**
     * Copies all of the mappings from the specified map to this map.
     * These mappings will replace any mappings that this map had for
     * any of the keys currently in the specified map.
     *
     * @param m mappings to be stored in this map
     * @throws NullPointerException if the specified map is null
     */
    @SuppressWarnings("unchecked")
    @Override
    public void putAll(Map<? extends K, ? extends V> m) {
        for (Map.Entry<?, ?> e : m.entrySet()) {
            this.put((K) e.getKey(), (V) e.getValue());
        }
    }

    /**
     * Removes all of the mappings from this map.
     * The map will be empty after this call returns.
     * The capacity of this map will remain the same.
     */
    @Override
    public void clear() {
        this.size = 0;
        modCount++;
        entryTable = new AionMapEntry[entryTable.length];
    }

    /**
     * Returns a {@link Set} view of the keys contained in this map.
     * The key set is based on a snapshot of the map.
     * Modifications of the map after key set generation will not be reflected back to existing key set, and vice-versa.
     *
     * @return a set view of the keys contained in this map
     */
    @Override
    public Set<K> keySet() {
        return new KeySet();
    }

    /**
     * Returns a {@link Collection} view of the values contained in this map.
     * Values is based on a snapshot of the map.
     * Modifications of the map after values generation will not be reflected back to existing values, and vice-versa.
     *
     * @return a view of the values contained in this map
     */
    @Override
    public Collection<V> values() {
        return new Values();
    }

    /**
     * Returns a {@link Set} view of the mappings contained in this map.
     * The entry set is based on a snapshot of the map.
     * Modifications of the map after entry set generation will not be reflected back to existing entry set, and vice-versa.
     *
     * @return a set view of the mappings contained in this map
     */
    @Override
    public Set<Map.Entry<K, V>> entrySet() {
        return new EntrySet();
    }

    /**
     * Returns the value to which the specified key is mapped, or
     * {@code defaultValue} if this map contains no mapping for the key.
     *
     * @param key          the key whose associated value is to be returned
     * @param defaultValue the default mapping of the key
     * @return the value to which the specified key is mapped, or
     * {@code defaultValue} if this map contains no mapping for the key
     */
    public V getOrDefault(Object key, V defaultValue) {
        return containsKey(key) ? get(key) : defaultValue;
    }

    private final class EntrySet extends AionAbstractCollection<Entry<K, V>> implements Set<Entry<K, V>> {

        public Iterator<Map.Entry<K, V>> iterator() {
            return new EntryIterator();
        }

        public boolean contains(Object o) {
            K key = ((Entry<K, V>) o).getKey();
            V value = ((Entry<K, V>) o).getValue();

            return AionMap.this.containsKey(key) && (AionMap.this.get(key) == value || (value != null && AionMap.this.get(key).equals(value)));
        }

        public int size() {
            return size;
        }


        public boolean remove(Object o) {
            Entry<K, V> entry = (Entry<K, V>) o;
            return null != AionMap.this.remove(entry.getKey());
        }

        public void clear() {
            AionMap.this.clear();
        }
    }

    private final class KeySet extends AionAbstractCollection<K> implements Set<K> {

        public int size() {
            return size;
        }

        public Iterator<K> iterator() {
            return new KeyIterator();
        }

        public boolean contains(Object o) {
            return AionMap.this.containsKey(o);
        }

        public boolean remove(Object key) {
            return null != AionMap.this.remove(key);
        }

        public void clear() {
            AionMap.this.clear();
        }
    }

    private final class Values extends AionAbstractCollection<V> {
        public int size() {
            return size;
        }

        public Iterator<V> iterator() {
            return new ValueIterator();
        }

        public boolean contains(Object o) {
            return AionMap.this.containsValue(o);
        }

        public void clear() {
            AionMap.this.clear();
        }

        public boolean remove(Object o) {
            Iterator<V> it = iterator();
            if (o == null) {
                while (it.hasNext()) {
                    if (it.next() == null) {
                        it.remove();
                        return true;
                    }
                }
            } else {
                while (it.hasNext()) {
                    if (o.equals(it.next())) {
                        it.remove();
                        return true;
                    }
                }
            }
            return false;
        }
    }

    private final class EntryIterator extends HashIterator<Entry<K, V>> {
        public AionMapEntry<K, V> next() {
            return nextEntry();
        }
    }

    private final class ValueIterator extends HashIterator<V> {
        public V next() {
            return nextEntry().value;
        }
    }

    private final class KeyIterator extends HashIterator<K> {
        public K next() {
            return nextEntry().key;
        }
    }

    @SuppressWarnings("unchecked")
    private abstract class HashIterator<E> implements Iterator<E> {
        AionMapEntry<K, V> next;        // next entry to return
        AionMapEntry<K, V> current;     // current entry
        int expectedModCount;  // for fast-fail
        int index;             // current slot

        HashIterator() {
            expectedModCount = modCount;
            AionMapEntry[] t = entryTable;
            if (size > 0) {
                // advance to first entry
                while (index < t.length && next == null) {
                    next = t[index];
                    index++;
                }
            }
        }

        public boolean hasNext() {
            return next != null;
        }

        public AionMapEntry<K, V> nextEntry() {
            AionMapEntry<K, V> e = next;
            if (modCount != expectedModCount) {
                // ConcurrentModificationException is not supported in the AVM
                throw new RuntimeException();
            }
            if (e == null) {
                throw new NoSuchElementException();
            }

            next = e.next;
            if (next == null) {
                AionMapEntry<K, V>[] t = entryTable;
                // advance to next entry
                while (index < t.length && next == null) {
                    next = t[index];
                    index++;
                }
            }
            current = e;
            return e;
        }

        public void remove() {
            AionMapEntry<K, V> p = current;
            if (modCount != expectedModCount) {
                // ConcurrentModificationException is not supported in the AVM
                throw new RuntimeException();
            }

            current = null;
            K key = p.key;
            AionMap.this.remove(key);
            expectedModCount = modCount;
        }
    }

    private abstract class AionAbstractCollection<E> implements Collection<E> {

        public boolean isEmpty() {
            return size() == 0;
        }

        public boolean containsAll(Collection<?> c) {
            for (Object e : c) {
                if (!contains(e)) {
                    return false;
                }
            }
            return true;
        }

        public Object[] toArray() {
            throw new UnsupportedOperationException();
        }

        public <T> T[] toArray(T[] a) {
            throw new UnsupportedOperationException();
        }

        public boolean add(E e) {
            throw new UnsupportedOperationException();
        }

        public boolean addAll(Collection<? extends E> c) {
            throw new UnsupportedOperationException();
        }

        public boolean removeAll(Collection<?> c) {
            boolean modified = false;
            Iterator<?> it = iterator();
            while (it.hasNext()) {
                if (c.contains(it.next())) {
                    it.remove();
                    modified = true;
                }
            }
            return modified;
        }

        public boolean retainAll(Collection<?> c) {
            boolean modified = false;
            Iterator<E> it = iterator();
            while (it.hasNext()) {
                if (!c.contains(it.next())) {
                    it.remove();
                    modified = true;
                }
            }
            return modified;
        }
    }

    @SuppressWarnings("unchecked")
    private void resize(int newCapacity) {
        threshold = (int) (newCapacity * loadFactor);
        entryTable = rehashAndTransfer(newCapacity);
    }

    @SuppressWarnings("unchecked")
    private AionMapEntry<K, V>[] rehashAndTransfer(int capacity) {
        AionMapEntry<K, V>[] newTable = new AionMapEntry[capacity];

        AionMapEntry<K, V>[] src = entryTable;
        int length = src.length;

        for (int i = 0; i < length; i++) {
            AionMapEntry e = src[i];
            if (e != null) {
                src[i] = null;
                do {
                    AionMapEntry next = e.next;
                    //compute the new hash and add to new table
                    int newHash = hashValue(e.keyHashcode, capacity);
                    e.next = newTable[newHash];
                    newTable[newHash] = e;
                    e = next;
                } while (e != null);
            }
        }
        return newTable;
    }

    private void keyNullCheck(Object key) {
        if (null == key) {
            throw new NullPointerException();
        }
    }

    private int hashValue(int hashcode, int length) {
        if (hashcode < 0) {
            hashcode = -hashcode;
        }
        return hashcode % length;
    }

    static class AionMapEntry<K, V> implements Map.Entry<K, V> {

        // key is immutable and declared as public mainly to preserve energy cost
        public final K key;
        public final int keyHashcode;
        //declared as public mainly to reduce energy consumption
        public V value;
        // modified directly, indicating the next entry in the current bucket
        public AionMapEntry next;

        AionMapEntry(K key, V value) {
            this.key = key;
            this.value = value;
            this.next = null;
            this.keyHashcode = key.hashCode();
        }

        @Override
        public K getKey() {
            return key;
        }

        @Override
        public V getValue() {
            return value;
        }

        @Override
        public V setValue(V value) {
            V ret = this.value;
            this.value = value;
            return ret;
        }

        @Override
        public int hashCode() {
            return keyHashcode;
        }
    }
}