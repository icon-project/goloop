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

package foundation.icon.ee.ipc;

import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Bytes;
import org.msgpack.core.MessageBufferPacker;
import org.msgpack.value.ArrayValue;
import org.msgpack.value.Value;

import java.io.IOException;
import java.math.BigInteger;
import java.util.LinkedHashMap;
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

    public static Object decodeAny(Value raw) throws IOException {
        ArrayValue data = raw.asArrayValue();
        int tag = data.get(0).asIntegerValue().asInt();
        Value val = data.get(1);
        if (tag == NIL) {
            return null;
        } else if (tag == DICT) {
            Map<String, Object> map = new LinkedHashMap<>();
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
        } else if (tag == BYTES) {
            if (val.isNilValue()) {
                return null;
            }
            return val.asRawValue().asByteArray();
        } else if (tag == STRING) {
            return val.asStringValue().asString();
        } else if (tag == BOOL) {
            byte[] ba = val.asRawValue().asByteArray();
            return ba[0] != 0;
        } else if (tag == ADDRESS) {
            return new Address(val.asRawValue().asByteArray());
        } else if (tag == INT) {
            return new BigInteger(val.asRawValue().asByteArray());
        } else {
            throw new IOException("not supported tag: " + tag);
        }
    }

    public static TypedObj encodeAny(Object obj) throws IOException {
        if (obj == null) {
            return new TypedObj(NIL, null);
        } else if (obj instanceof Map) {
            @SuppressWarnings("unchecked")
            Map<String, Object> map = (Map<String, Object>) obj;
            Map<String, Object> typedMap = new LinkedHashMap<>();
            for (Map.Entry<String, Object> pair : map.entrySet()) {
                typedMap.put(pair.getKey(), encodeAny(pair.getValue()));
            }
            return new TypedObj(DICT, typedMap);
        } else if (obj instanceof Object[]) {
            Object[] arr = (Object[]) obj;
            Object[] list = new Object[arr.length];
            int i = 0;
            for (Object o : arr) {
                list[i++] = encodeAny(o);
            }
            return new TypedObj(LIST, list);
        } else if (obj instanceof byte[]) {
            return new TypedObj(BYTES, obj);
        } else if (obj instanceof String) {
            return new TypedObj(STRING, obj);
        } else if (obj instanceof Boolean) {
            boolean o = (Boolean) obj;
            return new TypedObj(BOOL, o ? new byte[]{1} : new byte[]{0});
        } else if (obj instanceof Address) {
            return new TypedObj(ADDRESS, ((Address)obj).toByteArray());
        } else if (obj instanceof BigInteger) {
            return new TypedObj(INT, ((BigInteger)obj).toByteArray());
        } else if (obj instanceof Byte) {
            var o  = (Byte) obj;
            return new TypedObj(INT, BigInteger.valueOf(o).toByteArray());
        } else if (obj instanceof Short) {
            var o  = (Short) obj;
            return new TypedObj(INT, BigInteger.valueOf(o).toByteArray());
        } else if (obj instanceof Integer) {
            var o  = (Integer) obj;
            return new TypedObj(INT, BigInteger.valueOf(o).toByteArray());
        } else if (obj instanceof Long) {
            var o  = (Long) obj;
            return new TypedObj(INT, BigInteger.valueOf(o).toByteArray());
        } else if (obj instanceof Character) {
            var o  = (Character) obj;
            return new TypedObj(INT, BigInteger.valueOf(o).toByteArray());
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

    void writeTo(MessageBufferPacker packer) throws IOException {
        packer.packArrayHeader(2);
        packer.packInt(type);
        if (type == NIL) {
            packer.packNil();
        } else if (type == DICT) {
            @SuppressWarnings("unchecked")
            Map<String, Object> map = (Map<String, Object>) obj;
            packer.packMapHeader(map.size());
            for (Map.Entry<String, Object> pair : map.entrySet()) {
                packer.packString(pair.getKey());
                TypedObj to = (TypedObj) pair.getValue();
                to.writeTo(packer);
            }
        } else if (type == LIST) {
            Object[] arr = (Object[]) obj;
            if (arr.length == 0) {
                packer.packArrayHeader(0);
            } else {
                packer.packArrayHeader(arr.length);
                for (Object o : arr) {
                    TypedObj to = (TypedObj) o;
                    to.writeTo(packer);
                }
            }
        } else if (type == STRING) {
            packer.packString((String) obj);
        } else if (type == BOOL || type == BYTES || type == ADDRESS || type == INT) {
            byte[] ba = (byte[]) obj;
            packer.packBinaryHeader(ba.length);
            packer.writePayload(ba);
        } else {
            throw new IOException("not supported type: " + type);
        }
    }
}
