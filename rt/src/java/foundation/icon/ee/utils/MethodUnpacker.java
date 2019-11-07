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
import foundation.icon.ee.types.Method.MethodType;
import org.msgpack.core.MessagePack;
import org.msgpack.core.MessageUnpacker;

import java.io.IOException;

public class MethodUnpacker {

    public static Method[] readFrom(byte[] data) throws IOException {
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
                    params[j] = getParameter(unpacker, false);
                }
                for (int j = indexed; j < inputSize; j++) {
                    params[j] = getParameter(unpacker, type == MethodType.FUNCTION);
                }
            }
            int output = unpacker.unpackArrayHeader();
            String outputDescriptor = "V";
            if (output != 0) {
                output = unpacker.unpackInt();
                outputDescriptor = unpacker.unpackString();
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

    private static Method.Parameter getParameter(MessageUnpacker unpacker, boolean optional) throws IOException {
        unpacker.unpackArrayHeader(); // 4
        String paramName = unpacker.unpackString();
        String paramDescriptor = unpacker.unpackString();
        int paramType = unpacker.unpackInt();
        unpacker.unpackValue(); // value ignored
        return new Method.Parameter(paramName, paramDescriptor, paramType, optional);
    }
}
