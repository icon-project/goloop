/*
 * Copyright 2020 ICON Foundation
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

package foundation.icon.ee.util;

import java.util.AbstractMap;
import java.util.HashMap;
import java.util.HashSet;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Set;
import java.util.function.Function;

public class LinkedHashMultimap<K, V> {
    private class LHSet extends LinkedHashMap<
            AbstractMap.SimpleEntry<K, V>,
            AbstractMap.SimpleEntry<K, V>> {
        LHSet() {
            super(16, 0.75f, true);
        }

        @Override
        protected boolean removeEldestEntry(
                Map.Entry<AbstractMap.SimpleEntry<K, V>, AbstractMap.SimpleEntry<K, V>> eldest) {
            var e = eldest.getValue();
            var res = LinkedHashMultimap.this.removeEldestEntry(e.getKey(),
                    e.getValue());
            if (res) {
                removeFromMultimap(e.getKey(), e.getValue());
            }
            return res;
        }
    }

    private final LHSet lru = new LHSet();
    private final HashMap<K, Set<V>> map = new HashMap<>();

    protected boolean removeEldestEntry(K key, V value) {
        return false;
    }

    private V removeFromMultimap(K k, V v) {
        var set = map.get(k);
        if (set == null) {
            return null;
        }
        var res = set.remove(v);
        if (set.isEmpty()) {
            map.remove(k);
        }
        return res ? v : null;
    }

    public int size() {
        return lru.size();
    }

    public V remove(K k, V v) {
        if (removeFromMultimap(k, v) != null) {
            var w = new AbstractMap.SimpleEntry<>(k, v);
            lru.remove(w);
            return v;
        }
        return null;
    }

    public V remove(K k, Function<Set<V>, V> selector) {
        var set = map.get(k);
        if (set == null) {
            return null;
        }
        return remove(k, selector.apply(set));
    }

    public void put(K k, V v) {
        var set = map.get(k);
        if (set == null) {
            var newSet = new HashSet<V>();
            newSet.add(v);
            map.put(k, newSet);
        } else {
            set.add(v);
        }
        var w = new AbstractMap.SimpleEntry<>(k, v);
        lru.put(w, w);
    }

    public boolean contains(K k, V v) {
        var set = map.get(k);
        if (set == null) {
            return false;
        }
        return set.contains(v);
    }
}
