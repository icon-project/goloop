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
 * An array DB holds a sequence of values.
 * @param <E> Element type. It shall be readable and writable class.
 * @see ObjectReader
 * @see ObjectWriter
 */
public interface ArrayDB<E> {
    /**
     * Adds a value at the end of the array DB.
     * @param value new value
     */
    void add(E value);

    /**
     * Sets value of the specified index.
     * @param index index
     * @param value new value
     * @throws IllegalArgumentException if index is out of range.
     */
    void set(int index, E value);

    /**
     * Removes last element of the array DB.
     * @throws IllegalStateException if array DB has zero elements.
     */
    void removeLast();

    /**
     * Returns the element at the specified position in the array DB.
     * @param index index of element
     * @return the element at the specified position in the array DB.
     * @throws IllegalArgumentException if index is out of range.
     */
    E get(int index);

    /**
     * Returns number of elements in this array DB.
     * @return number of elements in this array DB.
     */
    int size();

    /**
     * Pops last element of the array DB.
     * @return last element of array DB
     * @throws IllegalStateException if array DB has zero elements.
     */
    E pop();
}
