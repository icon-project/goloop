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

package example.util;

import score.ArrayDB;
import score.Context;
import score.DictDB;

import java.math.BigInteger;

public class EnumerableIntMap<E> {
    private final ArrayDB<BigInteger> keys;
    private final DictDB<BigInteger, E> map;

    public EnumerableIntMap(String id, Class<E> valueClass) {
        this.keys = Context.newArrayDB(id, BigInteger.class);
        this.map = Context.newDictDB(id, valueClass);
    }

    public int length() {
        return keys.size();
    }

    public boolean contains(BigInteger key) {
        return map.get(key) != null;
    }

    public BigInteger get(int i) {
        Context.require(i < keys.size());
        return keys.get(i);
    }

    public E getOrDefault(BigInteger key, E defaultValue) {
        return map.getOrDefault(key, defaultValue);
    }

    public E getOrThrow(BigInteger key, String msg) {
        E v = map.get(key);
        if (v == null) {
            Context.revert(msg);
        }
        return v;
    }

    public void set(BigInteger key, E to) {
        if (!contains(key)) {
            keys.add(key);
        }
        map.set(key, to);
    }

    public void remove(BigInteger key) {
        Arrays.remove(keys, key);
        map.set(key, null);
    }
}
