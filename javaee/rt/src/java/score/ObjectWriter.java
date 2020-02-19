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

// charge per byte cost
public interface ObjectWriter {
    void write(boolean v);
    void write(byte v);
    void write(short v);
    void write(char v);
    void write(int v);
    void write(float v);
    void write(long v);
    void write(double v);
    void write(s.java.math.BigInteger v);
    void write(String v);
    void write(byte[] v);
    void write(Address v);
    void write(Object v);
    void writeNullable(Object v);
    void write(Object... v);
    void writeNullable(Object... v);
    void writeNull();

    void beginList(int l);
    void beginNullableList(int l);
    void writeListOf(Object... v);
    void beginMap(int l);
    void beginNullableMap(int l);
    void end();
}
