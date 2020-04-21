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
 *  Interface to read objects.
 *
 *  <p>There are two types of read/write operation - normal and nullable.
 *  An object written by normal write operation shall be read by normal read
 *  operation. Also, an object written by nullable write operation shall be
 *  read by nullable read operation.
 *
 *  <p>You can read objects of custom class if the class has to following
 *  method.
 *
 *  <p><code>
 *      public static UserClass readObject(ObjectReader r)
 *  </code>
 *
 *  <p>When you read custom class object, {@code readObject} method is called.
 *  If the readObject method throws {@link java.lang.RuntimeException}, the
 *  exception is rethrown. If the method throws other
 *  {@link java.lang.Throwable}, the exception is consumed and
 *  {@link java.lang.UnsupportedOperationException} is thrown instead.
 *
 *  <p>If an error occurs during any read call, the reader becomes invalidated.
 */
public interface ObjectReader {
    boolean readBoolean();
    byte readByte();
    short readShort();
    char readChar();
    int readInt();
    float readFloat();
    long readLong();
    double readDouble();
    BigInteger readBigInteger();
    String readString();
    byte[] readByteArray();
    Address readAddress();
    <T> T read(Class<T> c);
    <T> T readOrDefault(Class<T> c, T def);
    <T> T readNullable(Class<T> c);
    <T> T readNullableOrDefault(Class<T> c, T def);

    void beginList();
    /**
     *  Begins a nullable list.
     *
     *  <p>If the nullable list is not null, the function returns {@code true}
     *  and the caller may read elements of the list. After reading zero or more
     *  elements, the caller must call {@link #end} to end list. It is not
     *  required to read all elements before an {@link #end} call.
     *
     *  <p>If the nullable list is null, the function return {@code false} and
     *  the caller may not read element. The caller shall not call {@link #end}
     *  for this list.
     *
     *  @return true if list is non-null and successfully begins a new
     *  list.
     */
    boolean beginNullableList();
    void beginMap();
    boolean beginNullableMap();
    boolean hasNext();
    void end();

    void skip();
    void skip(int count);
}
