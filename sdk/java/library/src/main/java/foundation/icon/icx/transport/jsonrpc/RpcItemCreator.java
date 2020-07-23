/*
 * Copyright 2018 ICON Foundation
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

package foundation.icon.icx.transport.jsonrpc;

import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;

import java.lang.reflect.Array;
import java.lang.reflect.Field;
import java.math.BigInteger;

public class RpcItemCreator {

    public static <T> RpcItem create(T item) {
        return toRpcItem(item);
    }

    static <T> RpcItem toRpcItem(T item) {
        return item != null ? toRpcItem(item.getClass(), item) : null;
    }

    static <T> RpcItem toRpcItem(Class<?> type, T item) {
        RpcValue rpcValue = toRpcValue(item);
        if (rpcValue != null) {
            return rpcValue;
        }

        if (type.isArray()) {
            return toRpcArray(item);
        }

        if (!type.isPrimitive()) {
            return toRpcObject(item);
        }

        return null;
    }

    static RpcObject toRpcObject(Object object) {
        RpcObject.Builder builder = new RpcObject.Builder();
        addObjectFields(builder, object, object.getClass().getDeclaredFields());
        addObjectFields(builder, object, object.getClass().getFields());
        return builder.build();
    }

    static String getKeyFromObjectField(Field field) {
        return field.getName();
    }

    static void addObjectFields(RpcObject.Builder builder, Object parent, Field[] fields) {
        for (Field field : fields) {
            String key = getKeyFromObjectField(field);
            if (key.equals("this$0")) continue;

            Class<?> type = field.getType();
            Object fieldObject = null;
            try {
                field.setAccessible(true);
                fieldObject = field.get(parent);
            } catch (IllegalAccessException ignored) {
            }
            if (fieldObject != null || !type.isInstance(fieldObject)) {
                RpcItem rpcItem = toRpcItem(type, fieldObject);
                if (rpcItem != null && !rpcItem.isEmpty()) {
                    builder.put(key, rpcItem);
                }
            }
        }
    }

    static RpcArray toRpcArray(Object obj) {
        Class<?> componentType = obj.getClass().getComponentType();
        if (componentType == boolean.class || !componentType.isPrimitive()) {
            RpcArray.Builder builder = new RpcArray.Builder();

            int length = Array.getLength(obj);
            for (int i = 0; i < length; i++) {
                builder.add(toRpcItem(Array.get(obj, i)));
            }
            return builder.build();
        }
        return null;
    }

    static RpcValue toRpcValue(Object object) {
        if (object.getClass().isAssignableFrom(Boolean.class)) {
            return new RpcValue((Boolean) object);
        } else if (object.getClass().isAssignableFrom(String.class)) {
            return new RpcValue((String) object);
        } else if (object.getClass().isAssignableFrom(BigInteger.class)) {
            return new RpcValue((BigInteger) object);
        } else if (object.getClass().isAssignableFrom(byte[].class)) {
            return new RpcValue((byte[]) object);
        } else if (object.getClass().isAssignableFrom(Bytes.class)) {
            return new RpcValue((Bytes) object);
        } else if (object.getClass().isAssignableFrom(Address.class)) {
            return new RpcValue((Address) object);
        }
        return null;
    }
}
