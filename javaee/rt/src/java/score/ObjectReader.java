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

    // returns length of list or -1 if unknown
    void beginList();
    void beginNullableList();
    boolean tryBeginNullableList();
    void beginMap();
    void beginNullableMap();
    boolean tryBeginNullableMap();
    boolean hasNext();
    void end();
    boolean tryReadNull();

    void skip();
    void skip(int count);
}
