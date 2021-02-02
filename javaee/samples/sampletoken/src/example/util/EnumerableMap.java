/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package example.util;

import score.ArrayDB;
import score.Context;
import score.DictDB;

public class EnumerableMap<K, V> {
    private final ArrayDB<V> entries;
    private final DictDB<K, Integer> indexes;

    public EnumerableMap(String id, Class<V> valueClass) {
        // array of valueClass
        this.entries = Context.newArrayDB(id, valueClass);
        // key => array index
        this.indexes = Context.newDictDB(id, Integer.class);
    }

    public int length() {
        return entries.size();
    }

    public Integer getIndex(K key) {
        return indexes.get(key);
    }

    public boolean contains(K key) {
        return getIndex(key) != null;
    }

    public V get(int i) {
        Context.require(i < entries.size());
        return entries.get(i);
    }

    public V get(K key) {
        Integer i = getIndex(key);
        return (i != null) ? entries.get(i) : null;
    }

    public void set(K key, V value) {
        Integer i = getIndex(key);
        if (i != null) {
            entries.set(i, value);
        } else {
            entries.add(value);
            indexes.set(key, entries.size() - 1);
        }
    }

    public void popAndSwap(K key, K lastKey) {
        Integer keyIndex = getIndex(key);
        Integer lastIndex = getIndex(lastKey);
        if (keyIndex != null
                && lastIndex != null
                && lastIndex == length() - 1) {
            var lastEntry = entries.pop();
            indexes.set(key, null);
            if (!keyIndex.equals(lastIndex)) {
                entries.set(keyIndex, lastEntry);
                indexes.set(lastKey, keyIndex);
            }
        }
    }
}
