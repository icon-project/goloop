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

package p.score;

import a.ByteArray;
import i.IObject;

// charges 0 cost
public interface ObjectReader {
    // UnsupportedOperationException : invalid or unsupported format
    // IllegalStateException : programming error or unexpected stream
    // NoSuchElementException : unexpected end of container
    boolean avm_readBoolean();
    byte avm_readByte();
    short avm_readShort();
    char avm_readChar();
    int avm_readInt();
    float avm_readFloat();
    long avm_readLong();
    double avm_readDouble();
    s.java.math.BigInteger avm_readBigInteger();
    s.java.lang.String avm_readString();
    ByteArray avm_readByteArray();
    Address avm_readAddress();
    <T extends IObject> T avm_read(s.java.lang.Class<T> c);
    <T extends IObject> T avm_readOrDefault(s.java.lang.Class<T> c, T def);
    <T extends IObject> T avm_readNullable(s.java.lang.Class<T> c);
    <T extends IObject> T avm_readNullableOrDefault(s.java.lang.Class<T> c, T def);

    // returns length of list or -1 if unknown
    void avm_beginList();
    boolean avm_beginNullableList();
    void avm_beginMap();
    boolean avm_beginNullableMap();
    boolean avm_hasNext();
    void avm_end();

    void avm_skip();
    void avm_skip(int count);

}
