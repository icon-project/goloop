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
import foundation.icon.ee.types.Method.MethodType;
import org.msgpack.core.MessagePack;
import org.msgpack.core.MessagePackException;
import org.msgpack.core.MessageUnpacker;

import java.io.IOException;

public class MethodUnpacker {
    public static Method[] readFrom(byte[] data) throws IOException {
        return readFrom(data, true);
    }

    public static Method[] readFrom(byte[] data, boolean longForm) throws IOException {
        try {
            return readFromImpl(data, longForm);
        } catch (MessagePackException e) {
            throw new IOException(e);
        }
    }

    private static Method[] readFromImpl(byte[] data, boolean longForm) throws IOException {
        MessageUnpacker unpacker = MessagePack.newDefaultUnpacker(data);
        int size = unpacker.unpackArrayHeader();
        Method[] methods = new Method[size];
        for (int i = 0; i < size; i++) {
            unpacker.unpackArrayHeader(); // 6
            int type = unpacker.unpackInt();
            String name = unpacker.unpackString();
            int flags = unpacker.unpackInt();
            int indexed = unpacker.unpackInt();
            int inputSize = unpacker.unpackArrayHeader();
            if (indexed > inputSize) {
                throw new IOException("Invalid indexed: " + indexed);
            }
            Method.Parameter[] params = new Method.Parameter[inputSize];
            if (inputSize > 0) {
                for (int j = 0; j < indexed; j++) {
                    params[j] = getParameter(unpacker, false, longForm);
                }
                for (int j = indexed; j < inputSize; j++) {
                    params[j] = getParameter(unpacker,
                            type == MethodType.FUNCTION, longForm);
                }
            }
            int output = unpacker.unpackArrayHeader();
            String outputDescriptor = "V";
            if (output != 0) {
                output = unpacker.unpackInt();
                if (longForm) {
                    outputDescriptor = unpacker.unpackString();
                }
            }
            if (!longForm) {
                outputDescriptor = "";
            }
            if (type == MethodType.FUNCTION) {
                methods[i] = Method.newFunction(name, flags, inputSize - indexed, params, output, outputDescriptor);
            } else if (type == MethodType.FALLBACK) {
                if ("fallback".equals(name)) {
                    methods[i] = Method.newFallback();
                } else {
                    throw new IOException("Invalid fallback: " + name);
                }
            } else if (type == MethodType.EVENT) {
                methods[i] = Method.newEvent(name, indexed, params);
            } else {
                throw new IOException("Unknown method type: " + type);
            }
        }
        return methods;
    }

    private static Method.Parameter getParameter(MessageUnpacker unpacker,
            boolean optional, boolean longForm) throws IOException {
        unpacker.unpackArrayHeader(); // 3 or 4 (longForm)
        String paramName = unpacker.unpackString();
        String paramDescriptor = "";
        if (longForm) {
            paramDescriptor = unpacker.unpackString();
        }
        int paramType = unpacker.unpackInt();
        unpacker.unpackValue(); // value ignored
        Method.Field[] sf = null;
        if ((paramType&Method.DataType.ELEMENT_MASK) == Method.DataType.STRUCT) {
            sf = unpackStructFields(unpacker);
        }
        return new Method.Parameter(paramName, paramDescriptor, paramType, sf,
                optional);
    }

    private static Method.Field[] unpackStructFields(MessageUnpacker unpacker)
        throws IOException {
        int n = unpacker.unpackArrayHeader();
        var res = new Method.Field[n];
        for (int i=0; i<n; i++) {
            unpacker.unpackArrayHeader(); // 3
            String name = unpacker.unpackString();
            int t = unpacker.unpackInt();
            Method.Field[] sf = null;
            if ((t & Method.DataType.ELEMENT_MASK) == Method.DataType.STRUCT) {
                sf = unpackStructFields(unpacker);
            } else {
                unpacker.unpackNil();
            }
            res[i] = new Method.Field(name, t, sf);
        }
        return res;
    }
}
