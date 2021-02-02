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

import score.Address;
import score.Context;
import score.ObjectReader;
import score.ObjectWriter;

import java.math.BigInteger;

public class IntToAddressMap {

    public static class IntToAddress {
        private final BigInteger key;
        private final Address value;

        public IntToAddress(BigInteger key, Address value) {
            this.key = key;
            this.value = value;
        }

        // for serialize
        public static void writeObject(ObjectWriter w, IntToAddress e) {
            w.write(e.key);
            w.write(e.value);
        }

        // for de-serialize
        public static IntToAddress readObject(ObjectReader r) {
            return new IntToAddress(
                    r.readBigInteger(),
                    r.readAddress()
            );
        }
    }

    private final EnumerableMap<BigInteger, IntToAddress> map;

    public IntToAddressMap(String id) {
        map = new EnumerableMap<>(id, IntToAddress.class);
    }

    public int length() {
        return map.length();
    }

    public boolean contains(BigInteger key) {
        return map.contains(key);
    }

    public BigInteger getKey(int index) {
        var entry = map.get(index);
        return entry.key;
    }

    public Address getOrThrow(BigInteger key, String msg) {
        var entry = map.get(key);
        if (entry != null) {
            return entry.value;
        }
        Context.revert(msg);
        return null; // should not reach here, but made compiler happy
    }

    public void set(BigInteger key, Address value) {
        var entry = new IntToAddress(key, value);
        map.set(key, entry);
    }

    public void remove(BigInteger key) {
        var lastKey = getKey(length() - 1);
        map.popAndSwap(key, lastKey);
    }
}
