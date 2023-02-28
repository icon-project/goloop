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

package foundation.icon.ee.util;

import foundation.icon.ee.types.Method;
import org.msgpack.core.MessageBufferPacker;

import java.io.IOException;
import java.math.BigInteger;

public class MethodPacker {

    public static void writeTo(Method m, MessageBufferPacker packer, boolean longForm) throws IOException {
        packer.packArrayHeader(6);
        packer.packInt(m.getType());
        packer.packString(m.getName());
        packer.packInt(m.getFlags());
        packer.packInt(m.getIndexed());
        if (m.getInputs() != null) {
            packer.packArrayHeader(m.getInputs().length);
            for (Method.Parameter p : m.getInputs()) {
                boolean isStruct = Method.DataType.getElement(p.getType()) == Method.DataType.STRUCT;
                int additionalFields = isStruct ? 1 : 0;
                if (longForm) {
                    packer.packArrayHeader(4 + additionalFields);
                    packer.packString(p.getName());
                    packer.packString(p.getDescriptor());
                } else {
                    packer.packArrayHeader(3 + additionalFields);
                    packer.packString(p.getName());
                }
                packer.packInt(p.getType());
                if (p.isOptional()) {
                    packDefaultValue(packer, p.getType());
                } else {
                    packer.packNil();
                }
                if (isStruct) {
                    packStructFields(packer, p.getStructFields());
                }
            }
        } else {
            packer.packArrayHeader(0);
        }
        if (m.getOutput() != 0) {
            packer.packArrayHeader(1);
            packer.packInt(m.getOutput());
            if (longForm) {
                packer.packString(m.getOutputDescriptor());
            }
        } else {
            packer.packArrayHeader(0);
        }
    }

    private static void packDefaultValue(MessageBufferPacker packer, int type) throws IOException {
        if (type == Method.DataType.INTEGER || type == Method.DataType.BOOL) {
            byte[] ba = BigInteger.valueOf(0).toByteArray();
            packer.packBinaryHeader(ba.length);
            packer.writePayload(ba);
        } else {
            packer.packNil();
        }
    }

    private static void packStructFields(MessageBufferPacker packer,
            Method.Field[] fields) throws IOException {
        packer.packArrayHeader(fields.length);
        for (var f : fields) {
            packer.packArrayHeader(3);
            packer.packString(f.getName());
            var t = f.getType();
            packer.packInt(t);
            if (Method.DataType.getElement(t) == Method.DataType.STRUCT) {
                packStructFields(packer, f.getStructFields());
            } else {
                packer.packNil();
            }
        }
    }
}
