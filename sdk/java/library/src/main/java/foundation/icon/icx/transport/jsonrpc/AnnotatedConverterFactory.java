/*
 * Copyright 2018 ICON Foundation.
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

import java.lang.reflect.Field;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Modifier;

import static foundation.icon.icx.data.Converters.fromRpcItem;

public class AnnotatedConverterFactory implements RpcConverter.RpcConverterFactory {

    @Override
    public <T> RpcConverter<T> create(Class<T> type) {
        return new RpcConverter<T>() {
            @Override
            public T convertTo(RpcItem object) {
                try {
                    T result;
                    try {
                        result = getClassInstance(type);
                    } catch (ClassNotFoundException | NoSuchMethodException | InvocationTargetException e) {
                        throw new IllegalArgumentException(e);
                    }

                    RpcObject o = object.asObject();
                    Field[] fields = type.getDeclaredFields();
                    for (Field field : fields) {
                        field.setAccessible(true);
                        if (field.isAnnotationPresent(ConverterName.class)) {
                            ConverterName n = field.getAnnotation(ConverterName.class);
                            Object value = fromRpcItem(o.getItem(n.value()), field.getType());
                            if (value != null) field.set(result, value);
                        }
                    }
                    return result;
                } catch (InstantiationException | IllegalAccessException e) {
                    throw new IllegalArgumentException(e);
                }
            }

            @Override
            public RpcItem convertFrom(T object) {
                return RpcItemCreator.create(object);
            }
        };

    }

    private <T> T getClassInstance(Class<T> type) throws IllegalAccessException, InstantiationException, ClassNotFoundException, NoSuchMethodException, InvocationTargetException {
        if (isInnerClass(type)) {
            String className = type.getCanonicalName().subSequence(0, type.getCanonicalName().length() - type.getSimpleName().length() - 1).toString();
            Class m = Class.forName(className);
            return type.getConstructor(m).newInstance(m.newInstance());
        }
        return type.newInstance();
    }

    private boolean isInnerClass(Class<?> clazz) {
        return clazz.isMemberClass() && !Modifier.isStatic(clazz.getModifiers());
    }
}
