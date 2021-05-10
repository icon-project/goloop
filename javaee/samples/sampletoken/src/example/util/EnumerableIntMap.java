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

import score.Context;
import score.DictDB;

import java.math.BigInteger;

public class EnumerableIntMap<V> {
    private final EnumerableSet<BigInteger> keys;
    private final DictDB<BigInteger, V> values;

    public EnumerableIntMap(String id, Class<V> valueClass) {
        this.keys = new EnumerableSet<>(id + "_keys", BigInteger.class);
        this.values = Context.newDictDB(id, valueClass);
    }

    public int length() {
        return keys.length();
    }

    public boolean contains(BigInteger key) {
        return keys.contains(key);
    }

    public BigInteger getKey(int index) {
        return keys.at(index);
    }

    public V get(BigInteger key) {
        return values.get(key);
    }

    public V getOrThrow(BigInteger key, String msg) {
        var entry = this.get(key);
        if (entry != null) {
            return entry;
        }
        Context.revert(msg);
        return null; // should not reach here, but made compiler happy
    }

    public void set(BigInteger key, V value) {
        values.set(key, value);
        keys.add(key);
    }

    public void remove(BigInteger key) {
        values.set(key, null);
        keys.remove(key);
    }
}
