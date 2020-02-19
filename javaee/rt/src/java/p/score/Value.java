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

package p.score;

import s.java.math.BigInteger;
import s.java.lang.String;
import a.ByteArray;

public interface Value {
    byte avm_asByte();
    short avm_asShort();
    int avm_asInt();
    long avm_asLong();
    float avm_asFloat();
    double avm_asDouble();
    char avm_asChar();
    boolean avm_asBoolean();
    BigInteger avm_asBigInteger();
    Address avm_asAddress();
    String avm_asString();
    ByteArray avm_asByteArray();
}
