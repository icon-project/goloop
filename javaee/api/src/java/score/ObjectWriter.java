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

package score;

import java.math.BigInteger;

/**
 * Interface for object write.
 *
 * <p>An object is writable if the object is a builtin object or its class
 * has the following method.
 * <blockquote><pre>
 *  public static void writeObject(ObjectWriter w, UserClass obj)
 * </pre></blockquote>
 *
 * <p>A builtin object is a simple object or a container object. A simple object
 * is
 * a {@link Boolean}, a {@link Byte}, a {@link Short}, a {@link Character},
 * a {@link Integer}, a {@link Float}, a {@link Long}, a {@link Double},
 * a {@link BigInteger}, a {@link String}, byte array or an {@link Address}.
 * They are written by corresponding {@code write} methods. The following is an
 * example of writing simple objects.
 * <blockquote><pre>
 *      objectWriter.write("a string");
 *      objectWriter.write(0);
 * </pre></blockquote>
 *
 * <p>A container object is a list or a map. A container may have another
 * container as its element.
 *
 * <p>A list has zero or more builtin objects as its elements. A list is written
 * by a {@link #beginList(int)} call followed by calls for writings of zero or
 * more its elements followed by a {@link #end} call. For example, the following
 * method writes a list of two elements.
 * <blockquote><pre>
 *      objectWriter.beginList(2);
 *          objectWriter.write(0);
 *          objectWriter.beginList(0);
 *          objectWriter.end();
 *      objectWriter.end();
 * </pre></blockquote>
 *
 * <p>A map has zero or more pairs of objects as its elements. A map is written
 * by a {@link #beginMap(int)} call followed by calls for writings of zero or
 * more its elements followed by a {@link #end} call. For example, the following
 * method writes a map of one element.
 * <blockquote><pre>
 *      objectWriter.beginMap(1);
 *          objectWriter.write("key");
 *          objectWriter.write("value");
 *      objectWriter.end();
 * </pre></blockquote>
 *
 * <p>You can write a custom object indirectly if its class has the following
 * method.
 * <blockquote><pre>
 *  public static void writeObject(ObjectWriter w, UserClass obj)
 * </pre></blockquote>
 * When you write a custom object, the
 * {@code writeObject} method is called. In the method, you must
 * write one equivalent builtin object. It is error to write no object or
 * two or more objects. The method may write one list with multiple elements.
 * For example, the following is error.
 * <blockquote><pre>
 *  public static void writeObject(ObjectWriter w, UserClass obj) {
 *      w.write(obj.name);
 *      w.write(obj.description);
 *  }
 * </pre></blockquote>
 * Instead, write a list with multiple elements if you want to write multiple
 * objects.
 * <blockquote><pre>
 *  public static void writeObject(ObjectWriter w, UserClass obj) {
 *      w.beginList(2);
 *          w.write(obj.name);
 *          w.write(obj.description);
 *      w.end();
 *  }
 * </pre></blockquote>
 *
 * <p>You can write an object as a non-nullable type object or nullable type
 * object. A non-nullable type object is always not null. A nullable object may
 * be null or non-null. A write method writes an object as non-nullable type
 * unless the document specifies the method writes a nullable type object.
 * For example, {@link #write(String)} writes non-nullable type object.
 * {@link #writeNullable(Object)} and {@link #writeNull()} methods
 * write a nullable type object. {@link #beginNullableList(int)} begins a
 * nullable list.
 *
 * <p>Nullable-ness shall be preserved during write and read. If a non-nullable
 * object is written, the object shall be read as a non-nullable object and
 * shall not be read as a nullable object later. If a nullable object is
 * written, the object shall be read as a nullable object and shall not be read
 * a non-nullable object later. For example, if an object is written as the
 * following:
 *
 * <blockquote><pre>
 *     Integer i = getInteger();
 *     objectWriter.writeNullable(i);
 * </pre></blockquote>
 * <p>
 * It is error to read the object as the following:
 * <blockquote><pre>
 *     int i = objectReader.readInt();
 * </pre></blockquote>
 * <p>
 * Instead, the object shall be read as the following:
 * <blockquote><pre>
 *     Integer i = objectReader.readNullable(Integer.class);
 * </pre></blockquote>
 *
 * <p>If an exception is thrown during any write call, the writer becomes
 * invalidated. An invalidated writer fails any method.
 *
 * @see ObjectReader
 */
public interface ObjectWriter {
    /**
     * Writes a boolean.
     *
     * @param v a boolean value.
     * @throws IllegalStateException if this writer is invalidated one.
     */
    void write(boolean v);

    /**
     * Writes a byte.
     *
     * @param v a byte value.
     * @throws IllegalStateException if this writer is invalidated one.
     */
    void write(byte v);

    /**
     * Writes a short.
     *
     * @param v a short value.
     * @throws IllegalStateException if this writer is invalidated one.
     */
    void write(short v);

    /**
     * Writes a character.
     *
     * @param v a character value.
     * @throws IllegalStateException if this writer is invalidated one.
     */
    void write(char v);

    /**
     * Writes a integer.
     *
     * @param v an integer value.
     * @throws IllegalStateException if this writer is invalidated one.
     */
    void write(int v);

    /**
     * Writes a float.
     *
     * @param v a float value.
     * @throws IllegalStateException if this writer is invalidated one.
     */
    void write(float v);

    /**
     * Writes a long.
     *
     * @param v a long value.
     * @throws IllegalStateException if this writer is invalidated one.
     */
    void write(long v);

    /**
     * Writes a double.
     *
     * @param v a double value.
     * @throws IllegalStateException if this writer is invalidated one.
     */
    void write(double v);

    /**
     * Writes a big integer.
     *
     * @param v a big integer.
     * @throws IllegalStateException if this writer is invalidated one.
     * @throws NullPointerException If {@code v} is {@code null}.
     */
    void write(BigInteger v);

    /**
     * Writes a string.
     *
     * @param v a string.
     * @throws IllegalStateException if this writer is invalidated one.
     * @throws NullPointerException If {@code v} is {@code null}.
     */
    void write(String v);

    /**
     * Writes a byte array.
     *
     * @param v a byte array.
     * @throws IllegalStateException if this writer is invalidated one.
     * @throws NullPointerException If {@code v} is {@code null}.
     */
    void write(byte[] v);

    /**
     * Writes an address.
     *
     * @param v an address.
     * @throws IllegalStateException if this writer is invalidated one.
     * @throws NullPointerException If {@code v} is {@code null}.
     */
    void write(Address v);

    /**
     * Writes an object. The class of the object shall be
     * {@link Boolean}, {@link Byte}, {@link Short}, {@link Character},
     * {@link Integer}, {@link Float}, {@link Long}, {@link Double},
     * {@link BigInteger}, {@link String}, byte array, {@link Address},
     * or a custom class with the following method.
     *
     * <blockquote><pre>
     * public static void writeObject(ObjectWriter w, UserClass obj)
     * </pre></blockquote>
     *
     * @param v object to be written.
     * @throws IllegalStateException if this writer is invalidated one.
     * @throws IllegalArgumentException If the object is not a simple object
     *          and correct {@code writeObject} method is not available or
     *          the method threw {@link Throwable} which is not an
     *          {@link RuntimeException}.
     * @throws NullPointerException If {@code v} is {@code null}.
     */
    void write(Object v);

    /**
     * Writes a nullable object.
     *
     * @param v object to be written.
     * @throws IllegalStateException if this writer is invalidated one.
     * @throws IllegalArgumentException If the object is not a simple object
     *          and correct {@code writeObject} method is not available or
     *          the method threw {@link Throwable} which is not an
     *          {@link RuntimeException}.
     * @see #write(Object)
     */
    void writeNullable(Object v);

    /**
     * Writes objects.
     *
     * @param v objects to be written.
     * @throws IllegalStateException if this writer is invalidated one.
     * @throws IllegalArgumentException If the object is not a simple object
     *          and correct {@code writeObject} method is not available or
     *          the method threw {@link Throwable} which is not an
     *          {@link RuntimeException}.
     * @throws NullPointerException If {@code v} is {@code null}.
     * @see #write(Object)
     */
    void write(Object... v);

    /**
     * Writes nullable objects.
     *
     * @param v objects to be written.
     * @throws IllegalStateException if this writer is invalidated one.
     * @throws IllegalArgumentException If the object is not a simple object
     *          and correct {@code writeObject} method is not available or
     *          the method threw {@link Throwable} which is not an
     *          {@link RuntimeException}.
     * @see #write(Object)
     */
    void writeNullable(Object... v);

    /**
     * Writes a null.
     *
     * @throws IllegalStateException if this writer is invalidated one.
     */
    void writeNull();

    /**
     * Writes a list header and begins a list.
     * <p>
     * A following write operation writes an element of list. When all elements
     * are written you must call {@link #end} to end the current list.
     *
     * @param l number of elements.
     * @throws IllegalStateException if this writer is invalidated one.
     * @see ObjectWriter
     */
    void beginList(int l);

    /**
     * Writes a nullable list header and begins a list.
     * <p>
     * A following write operation writes an element of list. When all elements
     * are written you must call {@link #end} to end the current list.
     *
     * @param l number of elements.
     * @throws IllegalStateException if this writer is invalidated one.
     * @see ObjectWriter
     */
    void beginNullableList(int l);

    /**
     * Writes a list. The operation writes a list header and writes elements
     * and ends the list.
     *
     * @param v list elements
     * @throws IllegalStateException if this writer is invalidated one.
     * @throws IllegalArgumentException If the object is not a simple object
     *          and correct {@code writeObject} method is not available or
     *          the method threw {@link Throwable} which is not an
     *          {@link RuntimeException}.
     * @see ObjectWriter
     */
    void writeListOf(Object... v);

    /**
     * Writes a list of nullable. The operation writes a list header and writes
     * elements and ends the list.
     *
     * @param v list elements
     * @throws IllegalStateException if this writer is invalidated one.
     * @throws IllegalArgumentException If the object is not a simple object
     *          and correct {@code writeObject} method is not available or
     *          the method threw {@link Throwable} which is not an
     *          {@link RuntimeException}.
     * @see ObjectWriter
     */
    void writeListOfNullable(Object... v);

    /**
     * Writes a map header and begins a map.
     * <p>
     * A following write operation writes an element of map. When all elements
     * are written you must call {@link #end} to end the current map.
     *
     * @param l number of elements.
     * @throws IllegalStateException if this writer is invalidated one.
     * @see ObjectWriter
     */
    void beginMap(int l);

    /**
     * Writes a nullable map header and begins a map.
     * <p>
     * A following write operation writes an element of map. When all elements
     * are written you must call {@link #end} to end the current map.
     *
     * @param l number of elements.
     * @throws IllegalStateException if this writer is invalidated one.
     * @see ObjectWriter
     */
    void beginNullableMap(int l);

    /**
     * Ends the current container.
     *
     * @throws IllegalStateException if this writer is invalidated one or
     *          if this end is imbalanced.
     * @see ObjectWriter
     */
    void end();
}
