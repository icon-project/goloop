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
 * Interface for object read.
 *
 * <p>Common specification for object reader and object writer is specified in
 * {@link ObjectWriter}. It is recommended to read object writer documentation
 * first before you read this documentation.
 *
 * <p>An object is readable if the object is a builtin object or its class
 * has the following method.
 * <blockquote><pre>
 *  public static UserClass readObject(ObjectReader r)
 * </pre></blockquote>
 *
 * <p>A simple object is read by corresponding read method.
 * The following is an example of reading simple objects.
 * <blockquote><pre>
 *      str = objectReader.readString();
 *      i = objectReader.readInt();
 * </pre></blockquote>
 *
 * <p>A list is read by a {@link #beginList} call followed by calls for reading
 * zero or more its elements followed by a {@link #end} call.
 *
 * <p>A map is read by a {@link #beginMap} call followed by calls for reading
 * zero or more its elements followed by a {@link #end} call.
 *
 * <p>You can read a custom object indirectly if its class has the following
 * method.
 * <blockquote><pre>
 *  public static UserClass readObject(ObjectReader r)
 * </pre></blockquote>
 * When you read a custom object of this class, the {@code readObject} method
 * is called. In the method, you must read one equivalent builtin object.
 * It is error to read no object or two or more objects. The method may
 * read on list with multiple elements.
 *
 * <p>If an exception is thrown during any read call, the reader becomes
 * invalidated. An invalidated reader fails any method.
 *
 * @see ObjectWriter
 */
public interface ObjectReader {
    /**
     * Reads a boolean.
     *
     * @return the boolean read.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     */
    boolean readBoolean();

    /**
     * Reads a byte.
     *
     * @return the byte read.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     */
    byte readByte();

    /**
     * Reads a short.
     *
     * @return the short read.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     */
    short readShort();

    /**
     * Reads a character.
     *
     * @return the char read.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     */
    char readChar();

    /**
     * Reads an integer.
     *
     * @return the int read.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     */
    int readInt();

    /**
     * Reads a float.
     *
     * @return the float read.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     */
    float readFloat();

    /**
     * Reads a long.
     *
     * @return the long read.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     */
    long readLong();

    /**
     * Reads a double.
     *
     * @return the double read.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     */
    double readDouble();

    /**
     * Reads a big integer.
     *
     * @return the big integer read.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     */
    BigInteger readBigInteger();

    /**
     * Reads a string.
     *
     * @return the string read.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     */
    String readString();

    /**
     * Reads a byte array.
     *
     * @return the byte array read.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     */
    byte[] readByteArray();

    /**
     * Reads an address.
     *
     * @return the address read.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     */
    Address readAddress();

    /**
     * Reads an object of the class {@code c}.
     *
     * @param <T> type of object to be read
     * @param c   Class of object to be read. It shall be one of
     *          {@link Boolean}, {@link Byte}, {@link Short}, {@link Character},
     *          {@link Integer}, {@link Float}, {@link Long}, {@link Double},
     *          {@link BigInteger}, {@link String}, byte array, {@link Address},
     *          or a custom class with the following method:
     * <blockquote><pre>
     * public static UserClass readObject(ObjectReader r)
     * </pre></blockquote>
     * @return the object read.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws IllegalArgumentException If the object is not a simple object
     *          and correct {@code readObject} method is not available or
     *          the method threw {@link Throwable} which is not an
     *          {@link RuntimeException}.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     */
    <T> T read(Class<T> c);

    /**
     * Reads an object or returns default object if there is no next object.
     *
     * @param <T> type of object to be read
     * @param c   class of object to be read.
     * @param def the default object.
     * @return the object read or default object.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of type mismatch, corrupted stream
     *          or invalidated reader.
     * @throws IllegalArgumentException If the object is not a simple object
     *          and correct {@code readObject} method is not available or
     *          the method threw {@link Throwable} which is not an
     *          {@link RuntimeException}.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     * @see #hasNext
     */
    <T> T readOrDefault(Class<T> c, T def);

    /**
     * Reads a nullable object.
     *
     * @param <T> type of object to be read
     * @param c   class of object to be read.
     * @return read object or null.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws IllegalArgumentException If the object is not a simple object
     *          and correct {@code readObject} method is not available or
     *          the method threw {@link Throwable} which is not an
     *          {@link RuntimeException}.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     */
    <T> T readNullable(Class<T> c);

    /**
     * Reads a nullable object or returns default object if there is no next
     * object.
     *
     * @param <T> type of object to be read
     * @param c   class of object to be read.
     * @param def the default object.
     * @return read object or null if an object or null is read. Default object
     *          if there is no more item in current list or map.
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of type mismatch, corrupted stream
     *          or invalidated reader.
     * @throws IllegalArgumentException If the object is not a simple object
     *          and correct {@code readObject} method is not available or
     *          the method threw {@link Throwable} which is not an
     *          {@link RuntimeException}.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #read(Class)
     * @see #hasNext
     */
    <T> T readNullableOrDefault(Class<T> c, T def);

    /**
     * Reads a list header and begins a list.
     *
     * <p>If a list was successfully begun, a read operation reads an element
     * of the list in writing order. After reading zero or more elements, the
     * caller must call {@link #end} to end list. It is not required to read all
     * elements before an {@link #end} call.
     *
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     */
    void beginList();

    /**
     * Reads a nullable list header. If the reader reads a list header, a list
     * is begun and {@code true} is returned. If the reader reads null, no list
     * is begun and {@code false} is returned.
     *
     * <p>If a list was successfully begun, a read operation reads an element
     * of the list in writing order. After reading zero or more elements, the
     * caller must call {@link #end} to end list. It is not required to read all
     * elements before an {@link #end} call.
     *
     * @return true if a list header is read or false if null is read
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #beginList
     */
    boolean beginNullableList();

    /**
     * Reads a map header and begins a map.
     *
     * <p>If a map was successfully begun, elements of map can be read.
     * A map element consist of a key and its value and can be read by
     * two separate reads and elements are read in writing order. For example,
     * The first read operation after beginning of a map reads the key of the
     * first map element and the next read operation reads it value and the next
     * read operation reads the key of the second element, and so on.
     * After reading keys and values, the caller must call {@link #end} to end
     * map. It is not required to read all elements before an {@link #end} call.
     *
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     */
    void beginMap();

    /**
     * Reads a nullable map header. If the reader reads a map header, a map is
     * begun and {@code true} is returned. If the reader reads null, no map is
     * begun and {@code false} is returned.
     *
     * <p>If a map was successfully begun, elements of map can be read.
     * A map element consist of a key and its value and can be read by
     * two separate reads and elements are read in writing order. For example,
     * The first read operation after beginning of a map reads the key of the
     * first map element and the next read operation reads it value and the next
     * read operation reads the key of the second element, and so on.
     * After reading keys and values, the caller must call {@link #end} to end
     * map. It is not required to read all elements before an {@link #end} call.
     *
     * @return true if a map header is read or false if null is read
     *
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     * @see #beginMap
     */
    boolean beginNullableMap();

    /**
     * Returns true if this reader has a next object to read.
     * If the reader is reading a container, this method returns true if the top
     * most container has more object to read,
     * returns false otherwise.
     *
     * @return true if this reader has a next object to read.
     * @throws IllegalStateException if this reader was already invalidated.
     */
    boolean hasNext();

    /**
     * Ends the current container. Unread elements of the current container
     * are all skipped.
     *
     * @throws IllegalStateException if this end is imbalanced.
     */
    void end();

    /**
     * Skips an element of the current container.
     *
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     */
    void skip();

    /**
     * Skips elements of the current container.
     *
     * @param count the count.
     *
     * @throws IllegalStateException If this reader cannot read the
     *          given type of object because of end of stream, end of list,
     *          end of map, type mismatch, corrupted stream or invalidated
     *          reader.
     * @throws UnsupportedOperationException If this reader cannot read an
     *          object because the object is too long (for example, 2^32 bytes
     *          or longer byte array).
     */
    void skip(int count);
}
