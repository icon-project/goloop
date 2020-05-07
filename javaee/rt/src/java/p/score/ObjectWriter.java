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
import i.IObjectArray;

public interface ObjectWriter {
    void avm_write(boolean v);
    void avm_write(byte v);
    void avm_write(short v);
    void avm_write(char v);
    void avm_write(int v);
    void avm_write(float v);
    void avm_write(long v);
    void avm_write(double v);
    void avm_write(s.java.math.BigInteger v);
    void avm_write(s.java.lang.String v);
    void avm_write(ByteArray v);
    void avm_write(Address v);
    void avm_write(IObject v);
    void avm_writeNullable(IObject v);
    void avm_write(IObjectArray v);
    void avm_writeNullable(IObjectArray v);
    void avm_writeNull();

    void avm_beginList(int l);
    void avm_beginNullableList(int l);
    void avm_writeListOf(IObjectArray v);
    void avm_writeListOfNullable(IObjectArray v);
    void avm_beginMap(int l);
    void avm_beginNullableMap(int l);
    void avm_end();
}
