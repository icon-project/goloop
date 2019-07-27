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

package foundation.icon.tools.ipc;

import foundation.icon.common.Address;
import foundation.icon.common.Bytes;
import org.msgpack.core.MessageBufferPacker;
import org.msgpack.value.ArrayValue;
import org.msgpack.value.Value;

import java.io.IOException;
import java.math.BigInteger;
import java.util.HashMap;
import java.util.Map;

public class TypedObj {
    private static final int NIL = 0;
    private static final int DICT = 1;
    private static final int LIST = 2;
    private static final int BYTES = 3;
    private static final int STRING = 4;
    private static final int BOOL = 5;

    private static final int CUSTOM = 10;
    private static final int ADDRESS = CUSTOM;
    private static final int INT = CUSTOM + 1;

    private final int type;
    private final Object obj;

    private TypedObj(int type, Object value) {
        this.type = type;
        this.obj = value;
    }

    static Object decodeAny(Value raw) throws IOException {
        ArrayValue data = raw.asArrayValue();
        int tag = data.get(0).asIntegerValue().asInt();
        Value val = data.get(1);
        if (tag == DICT) {
            Map<String, Object> map = new HashMap<>();
            for (Map.Entry<Value, Value> pair : val.asMapValue().entrySet()) {
                map.put(pair.getKey().asStringValue().asString(),
                        decodeAny(pair.getValue()));
            }
            return map;
        } else if (tag == LIST) {
            ArrayValue arr = val.asArrayValue();
            Object[] list = new Object[arr.size()];
            int i = 0;
            for (Value v : arr) {
                list[i++] = decodeAny(v);
            }
            return list;
        } else {
            return decode(tag, val);
        }
    }

    public static Object decode(int tag, Value value) throws IOException {
        if (tag == NIL) {
            return null;
        } else if (tag == BYTES) {
            return value.asRawValue().asByteArray();
        } else if (tag == STRING) {
            return value.asStringValue().asString();
        } else if (tag == BOOL) {
            byte[] ba = value.asRawValue().asByteArray();
            return ba[0] != 0;
        } else if (tag == ADDRESS) {
            return new Address(value.asRawValue().asByteArray());
        } else if (tag == INT) {
            return new BigInteger(value.asRawValue().asByteArray());
        } else {
            throw new IOException("not supported tag: " + tag);
        }
    }

    public static TypedObj encodeAny(Object obj) throws IOException {
        if (obj == null) {
            return new TypedObj(NIL, null);
        } else if (obj instanceof byte[]) {
            return new TypedObj(BYTES, obj);
        } else if (obj instanceof String) {
            return new TypedObj(STRING, obj);
        } else if (obj instanceof Boolean) {
            return new TypedObj(BOOL, obj);
        } else if (obj instanceof Address) {
            return new TypedObj(ADDRESS, ((Address)obj).toByteArray());
        } else if (obj instanceof BigInteger) {
            return new TypedObj(INT, ((BigInteger)obj).toByteArray());
        }
        throw new IOException("not supported type: " + obj);
    }

    public String toString() {
        if (type == NIL) {
            return "nil";
        } else if (type == LIST) {
            Object[] arr = (Object[]) obj;
            if (arr.length == 0) {
                return "[]";
            } else {
                StringBuilder sb = new StringBuilder();
                sb.append("[");
                sb.append((arr[0] != null) ? arr[0].toString() : null);
                for (int i = 1; i < arr.length; i++) {
                    sb.append(", ");
                    sb.append((arr[i] != null) ? arr[i].toString() : null);
                }
                sb.append("]");
                return sb.toString();
            }
        } else if (type == STRING) {
            return (String) obj;
        } else {
            return Bytes.toHexString(((byte[]) obj));
        }
    }

    void accept(MessageBufferPacker packer) throws IOException {
        System.out.println("=== TypedObj.accept() ===");
        System.out.println("  type: " + type);
        System.out.println("  obj: " + obj);
        packer.packArrayHeader(2);
        packer.packInt(type);
        if (type == NIL) {
            packer.packNil();
        } else if (type == STRING) {
            packer.packString((String) obj);
        } else if (type == BOOL) {
            packer.packBoolean((Boolean) obj);
        } else if (type == BYTES || type == ADDRESS || type == INT) {
            byte[] ba = (byte[]) obj;
            packer.packBinaryHeader(ba.length);
            packer.writePayload(ba);
        } else {
            throw new IOException("not supported type: " + type);
        }
    }
}
