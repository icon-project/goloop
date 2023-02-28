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
    private static final int MAX_METHODS = 256;
    private static final int MAX_INPUTS = 64;
    private static final int MAX_FIELDS = 64;
    private static final int MAX_STRUCT_DEPTH = 4;

    private static int ensureMaxSize(int max, int size) throws IOException {
        if (size < 0 || max <= size) {
            throw new IOException("Invalid size");
        }
        return size;
    }

    private static void ensureSize(int exp, int size) throws IOException {
        if (exp != size) {
            throw new IOException("Size mismatch");
        }
    }

    private static int ensureFlags(int flags) throws IOException {
        if ((flags & (Method.Flags.MAX_FLAG - 1)) != flags) {
            throw new IOException("Invalid flags");
        }
        return flags;
    }

    public static Method[] readFrom(byte[] data) throws IOException {
        return readFrom(data, true);
    }

    public static Method[] readFrom(byte[] data, boolean longForm) throws IOException {
        try {
            if (data == null) {
                throw new IOException("data is null");
            }
            return readFromImpl(data, longForm);
        } catch (MessagePackException e) {
            throw new IOException(e);
        }
    }

    private static Method[] readFromImpl(byte[] data, boolean longForm) throws IOException {
        var unpackerConfig = MessagePack.DEFAULT_UNPACKER_CONFIG.withStringSizeLimit(255);
        MessageUnpacker unpacker = unpackerConfig.newUnpacker(data);
        int size = ensureMaxSize(MAX_METHODS, unpacker.unpackArrayHeader());
        Method[] methods = new Method[size];
        for (int i = 0; i < size; i++) {
            ensureSize(6, unpacker.unpackArrayHeader());
            int type = unpacker.unpackInt();
            String name = unpacker.unpackString();
            int flags = ensureFlags(unpacker.unpackInt());
            int indexed = ensureMaxSize(MAX_INPUTS, unpacker.unpackInt());
            int inputSize = ensureMaxSize(MAX_INPUTS, unpacker.unpackArrayHeader());
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
                ensureSize(1, output);
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
        int size = unpacker.unpackArrayHeader();
        String paramName = unpacker.unpackString(); size--;
        String paramDescriptor = "";
        if (longForm) {
            paramDescriptor = unpacker.unpackString(); size--;
        }
        int paramType = unpacker.unpackInt(); size--;
        if (!unpacker.tryUnpackNil()) {
            ensureSize(1, unpacker.unpackBinaryHeader());
            unpacker.readPayload(1); // value ignored
        }
        size--;
        Method.Field[] sf = null;
        if (Method.DataType.getElement(paramType) == Method.DataType.STRUCT) {
            sf = unpackStructFields(unpacker, 0); size--;
        }
        if (size != 0) {
            throw new IOException("Invalid param size");
        }
        return new Method.Parameter(paramName, paramDescriptor, paramType, sf,
                optional);
    }

    private static Method.Field[] unpackStructFields(MessageUnpacker unpacker, int depth)
        throws IOException {
        depth = ensureMaxSize(MAX_STRUCT_DEPTH, depth + 1);
        int n = ensureMaxSize(MAX_FIELDS, unpacker.unpackArrayHeader());
        var res = new Method.Field[n];
        for (int i=0; i<n; i++) {
            ensureSize(3, unpacker.unpackArrayHeader());
            String name = unpacker.unpackString();
            int t = unpacker.unpackInt();
            Method.Field[] sf = null;
            if (Method.DataType.getElement(t) == Method.DataType.STRUCT) {
                sf = unpackStructFields(unpacker, depth);
            } else {
                unpacker.unpackNil();
            }
            res[i] = new Method.Field(name, t, sf);
        }
        return res;
    }
}
