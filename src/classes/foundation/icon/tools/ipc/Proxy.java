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
import java.util.ArrayList;
import java.util.List;

import static org.msgpack.value.ValueType.ARRAY;

public class Proxy {

    public class MsgType {
        public static final int VERSION = 0;
        public static final int INVOKE = 1;
        public static final int RESULT = 2;
        public static final int GETVALUE = 3;
        public static final int SETVALUE = 4;
        public static final int CALL = 5;
        public static final int EVENT = 6;
        public static final int GETINFO = 7;
        public static final int GETBALANCE = 8;
        public static final int GETAPI = 9;
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

    class APIInfo {
        List<List<TypedObj>> methods = new ArrayList<>();

        void addFunction() {

        }

        void addFallback() {

        }

        void addEvent() {

        }

        List getData() {
            return methods;
        }
    }

    private final Client client;
    private final MessageUnpacker unpacker;

    public Proxy(Client client) {
        this.client = client;
        unpacker = MessagePack.newDefaultUnpacker(client.getInputStream());
    }

    public void sendMessage(int msgType, Object... args) throws IOException {
        MessageBufferPacker packer = MessagePack.newDefaultBufferPacker();
        packer.packArrayHeader(2);
        packer.packInt(msgType);
        packer.packArrayHeader(args.length);
        for (Object arg : args) {
            if (arg instanceof Integer) {
                packer.packInt((int) arg);
            } else if (arg instanceof String) {
                packer.packString((String) arg);
            } else {
                throw new IOException("not yet supported");
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

    private void handleGetApi(String path) throws IOException {
        // FIXME: invoke the real method
        APIInfo info = getApiInfo(path);
        if (info != null) {
            sendMessage(MsgType.GETAPI, info.getData());
        }
    }

    // DEBUG: dummy for test
    private APIInfo getApiInfo(String path) {
        return null;
    }
}
