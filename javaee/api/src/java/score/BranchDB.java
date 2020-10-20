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
 * A branch DB is a hash from keys to sub-DBs.
 * @param <K> Key type. K shall be String, byte array, Address,
 *           Byte, Short, Integer, Long, Character or BigInteger.
 * @param <V> Value type. V shall be VarDB, DictDB, ArrayDB or BranchDB.
 */
public interface BranchDB<K, V> {
    /**
     * Returns sub-DB for the key.
     *
     * @param key key for sub-DB.
     * @return sub-DB.
     */
    V at(K key);
}
