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
 * A variable DB holds one value.
 * @param <E> Variable type. It shall be readable and writable class.
 * @see ObjectReader
 * @see ObjectWriter
 */
public interface VarDB<E> {
    /**
     * Sets value.
     * @param value new value
     */
    void set(E value);

    /**
     * Returns the current value.
     * @return current value
     */
    E get();

    /**
     * Returns the current value or {@code defaultValue} if the current value
     * is {@code null}.
     * @param defaultValue default value
     * @return the current value or {@code defaultValue} if the current value
     * is {@code null}.
     */
    E getOrDefault(E defaultValue);
}
