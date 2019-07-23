/*
 * Copyright (c) 2019 ICON Foundation
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

import org.msgpack.core.MessageBufferPacker;
import org.msgpack.core.MessagePack;
import org.msgpack.core.MessageUnpacker;
import org.msgpack.value.ArrayValue;
import org.msgpack.value.Value;

import java.io.IOException;

import static org.msgpack.value.ValueType.ARRAY;

public class Proxy {

    class MsgType {
        static final int VERSION = 0;
        static final int INVOKE = 1;
        static final int RESULT = 2;
        static final int GETVALUE = 3;
        static final int SETVALUE = 4;
        static final int CALL = 5;
        static final int EVENT = 6;
        static final int GETINFO = 7;
        static final int GETBALANCE = 8;
        static final int GETAPI = 9;
    }

    class Message {
        final int type;
        final Value value;

        Message(int type, Value value) {
            this.type = type;
            this.value = value;
        }
    }

    class TypedObj {
        static final int NIL = 0;
        static final int DICT = 1;
        static final int LIST = 2;
        static final int BYTES = 3;
        static final int STRING = 4;
        static final int BOOL = 5;

        static final int CUSTOM = 10;
        static final int ADDRESS = CUSTOM;
        static final int INT = CUSTOM + 1;

        final int type;
        final Object value;

        TypedObj(int type, Object value) {
            this.type = type;
            this.value = value;
        }
    }

    private final Client client;
    private final MessageUnpacker unpacker;

    public Proxy(Client client) {
        this.client = client;
        unpacker = MessagePack.newDefaultUnpacker(client.getInputStream());
    }

    public void connect(String uuid) throws IOException {
        sendMessage(MsgType.VERSION, 1, uuid, "java");
    }

    public void handleMessages() throws IOException {
        while (true) {
            Message msg = getNextMessage();
            switch (msg.type) {
                case MsgType.GETAPI:
                    String path = msg.value.asStringValue().asString();
                    System.out.println("[GETAPI] path=" + path);
                    handleGetApi(path);
                    break;
                case MsgType.INVOKE:
                    System.out.println("[INVOKE]");
                    break;
            }
        }
    }

    private void sendMessage(int msgType, Object... args) throws IOException {
        MessageBufferPacker packer = MessagePack.newDefaultBufferPacker();
        packer.packArrayHeader(2);
        packer.packInt(msgType);
        packer.packArrayHeader(args.length);
        for (Object arg : args) {
            if (arg instanceof Integer) {
                packer.packInt((int) arg);
            } else if (arg instanceof String) {
                packer.packString((String) arg);
            } else if (arg instanceof Method[]) {
                Method[] methods = (Method[]) arg;
                packer.packArrayHeader(methods.length);
                for (Method m : methods) {
                    m.accept(packer);
                }
            } else {
                throw new IOException("not yet supported: " + arg);
            }
        }
        packer.close();
        client.send(packer.toByteArray());
    }

    private Message getNextMessage() throws IOException {
        Value v = unpacker.unpackValue();
        if (v.getValueType() != ARRAY) {
            throw new IOException("should be array type");
        }
        ArrayValue a = v.asArrayValue();
        int type = a.get(0).asIntegerValue().toInt();
        Value value = a.get(1);
        Message m = new Message(type, value);

        //for DEBUG
        System.out.println("Array size: " + a.size());
        System.out.println("[MsgType] " + m.type);
        for (Value e : a) {
            System.out.println("-- type: " + e.getValueType());
            System.out.println("   value: " + e);
        }
        //
        return m;
    }

    private void handleGetApi(String path) throws IOException {
        // FIXME: invoke the real method
        Method[] methods = dummyApiInfo(path);
        if (methods != null) {
            sendMessage(MsgType.GETAPI, 0, methods);
        }
    }

    // DEBUG: dummy for test
    private Method[] dummyApiInfo(String path) {
        return new Method[] {
            Method.newFunction(
                "balanceOf",
                Method.Flags.READONLY | Method.Flags.EXTERNAL,
                new Method.Parameter[] {
                    new Method.Parameter("_owner", Method.DataType.ADDRESS)
                },
                Method.DataType.INTEGER
            ),
            Method.newFunction(
                "name",
                Method.Flags.READONLY | Method.Flags.EXTERNAL,
                null,
                Method.DataType.STRING
            ),
            Method.newFunction(
                "transfer",
                Method.Flags.EXTERNAL,
                new Method.Parameter[] {
                    new Method.Parameter("_to", Method.DataType.ADDRESS),
                    new Method.Parameter("_value", Method.DataType.INTEGER),
                    new Method.Parameter("_data", Method.DataType.BYTES)
                },
                Method.DataType.NONE
            ),
            Method.newFallback(),
            Method.newEvent(
                "Transfer",
                3,
                new Method.Parameter[] {
                    new Method.Parameter("_from", Method.DataType.ADDRESS),
                    new Method.Parameter("_to", Method.DataType.ADDRESS),
                    new Method.Parameter("_value", Method.DataType.INTEGER),
                    new Method.Parameter("_data", Method.DataType.BYTES)
                }
            ),
        };
    }
}
