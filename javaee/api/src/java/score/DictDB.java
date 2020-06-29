/*
 * Copyright 2019 ICON Foundation
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

package score;

/**
 * A dictionary DB is a hash from key to value.
 * Only values of the dictionary DB is recorded in the DB.
 * Keys are not recorded.
 * @param <K> Key type. It shall be String, byte array, Address,
 *           Byte, Short, Integer, Long, Character or BigInteger.
 * @param <V> Value type. It shall be readable and writable class.
 * @see ObjectReader
 * @see ObjectWriter
 */
public interface DictDB<K, V> {
    /**
     * Sets a value for a key
     * @param key key
     * @param value value for the key
     */
    void set(K key, V value);

    /**
     * Returns the value for a key
     * @param key key
     * @return the value for a key
     */
    V get(K key);

    /**
     * Returns the value for a key or {@code defaultValue} if the value is
     * {@code null}.
     * @param key key
     * @param defaultValue default value
     * @return the value for a key or {@code defaultValue} if the value is
     * {@code null}.
     */
    V getOrDefault(K key, V defaultValue);
}
