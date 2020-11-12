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

import java.util.ArrayList;
import java.util.Collection;
import java.util.List;
import java.util.Map;

public class Multimap {
    public static <K, V> void addAll(Map<K, List<V>> map, K key,
            Collection<V> values) {
        var e = map.get(key);
        if (e == null) {
            map.put(key, new ArrayList<>(values));
            return;
        }
        e.addAll(values);
    }

    public static <K, V> void add(Map<K, List<V>> map, K key, V value) {
        var e = map.get(key);
        if (e == null) {
            var al = new ArrayList<V>();
            al.add(value);
            map.put(key, al);
            return;
        }
        e.add(value);
    }

    public static <K, V> List<V> getAllValues(Map<K, List<V>> map) {
        var al = new ArrayList<V>();
        for (var e : map.entrySet()) {
            al.addAll(e.getValue());
        }
        return al;
    }
}
