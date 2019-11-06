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

package foundation.icon.ee.utils;

import foundation.icon.ee.types.Method;
import org.msgpack.core.MessageBufferPacker;

import java.io.IOException;
import java.math.BigInteger;

public class MethodPacker {

    public static void writeTo(Method m, MessageBufferPacker packer) throws IOException {
        packer.packArrayHeader(6);
        packer.packInt(m.getType());
        packer.packString(m.getName());
        packer.packInt(m.getFlags());
        packer.packInt(m.getIndexed());
        if (m.getInputs() != null) {
            packer.packArrayHeader(m.getInputs().length);
            for (Method.Parameter p : m.getInputs()) {
                packer.packArrayHeader(3);
                packer.packString(p.getName());
                packer.packInt(p.getType());
                if (p.isOptional()) {
                    packDefaultValue(packer, p.getType());
                } else {
                    packer.packNil();
                }
            }
        } else {
            packer.packArrayHeader(0);
        }
        if (m.getOutput() != 0) {
            packer.packArrayHeader(1);
            packer.packInt(m.getOutput());
        } else {
            packer.packArrayHeader(0);
        }
    }

    private static void packDefaultValue(MessageBufferPacker packer, int type) throws IOException {
        if (type == Method.DataType.INTEGER) {
            packer.packBigInteger(BigInteger.ZERO);
        } else if (type == Method.DataType.BOOL) {
            packer.packBoolean(false);
        } else {
            // empty byte array
            packer.packBinaryHeader(0);
        }
    }
}
