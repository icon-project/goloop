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

public class EnumerableSet<V> {
    private final ArrayDB<V> entries;
    private final DictDB<V, Integer> indexes;

    public EnumerableSet(String id, Class<V> valueClass) {
        // array of valueClass
        this.entries = Context.newArrayDB(id, valueClass);
        // value => array index
        this.indexes = Context.newDictDB(id, Integer.class);
    }

    public int length() {
        return entries.size();
    }

    public V at(int index) {
        return entries.get(index);
    }

    public boolean contains(V value) {
        return indexes.get(value) != null;
    }

    public void add(V value) {
        if (!contains(value)) {
            // add new value
            entries.add(value);
            indexes.set(value, entries.size());
        }
    }

    public void remove(V value) {
        var valueIndex = indexes.get(value);
        if (valueIndex != null) {
            // pop and swap with the last entry
            int lastIndex = entries.size();
            V lastValue = entries.pop();
            indexes.set(value, null);
            if (lastIndex != valueIndex) {
                entries.set(valueIndex - 1, lastValue);
                indexes.set(lastValue, valueIndex);
            }
        }
    }
}
