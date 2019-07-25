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

import foundation.icon.common.Bytes;
import org.msgpack.core.MessageBufferPacker;
import org.msgpack.core.MessagePack;
import org.msgpack.core.MessageUnpacker;
import org.msgpack.value.ArrayValue;
import org.msgpack.value.Value;

import java.io.IOException;

import static org.msgpack.value.ValueType.ARRAY;

public class Proxy {

    private static final boolean DEBUG = true;

    private OnGetApiListener mOnGetApiListener;

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

    class Status {
        static final int SUCCESS = 0;
        static final int FAILURE = 1;
    }

    static class TypedObj {
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

        static TypedObj[] decodeList(ArrayValue data) throws IOException {
            int tag = data.get(0).asIntegerValue().asInt();
            if (tag == LIST) {
                ArrayValue arr = data.get(1).asArrayValue();
                TypedObj[] typed = new TypedObj[arr.size()];
                int i = 0;
                for (Value v : arr) {
                    System.out.println(" -- " + v);
                    ArrayValue v2 = v.asArrayValue();
                    tag = v2.get(0).asIntegerValue().asInt();
                    typed[i++] = new TypedObj(tag, decode(tag, v2.get(1)));
                }
                return typed;
            } else {
                throw new IOException("not supported tag: " + tag);
            }
        }

        static Object decode(int tag, Value value) {
            if (tag == STRING) {
                return value.asStringValue().asString();
            } else {
                return value.asRawValue().asByteArray();
            }
        }

        public String toString() {
            String str;
            if (type == STRING) {
                str = (String) value;
            } else {
                str = Bytes.toHexString(((byte[]) value));
            }
            if (str.equals("")) {
                str = "NIL";
            }
            return type + "@" + str;
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
                    handleGetInvoke(msg.value);
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
            if (arg == null) {
                packer.packNil();
            } else if (arg instanceof Integer) {
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

        if (DEBUG) {
            System.out.println("Array size: " + a.size());
            System.out.println("[MsgType] " + m.type);
            for (Value e : a) {
                System.out.println("-- type: " + e.getValueType());
                System.out.println("   value: " + e);
            }
        }
        return m;
    }

    public interface OnGetApiListener {
        Method[] onGetApi(String path);
    }

    public void setOnGetApiListener(OnGetApiListener listener) {
        mOnGetApiListener = listener;
    }

    private void handleGetApi(String path) throws IOException {
        if (mOnGetApiListener != null) {
            Method[] methods = mOnGetApiListener.onGetApi(path);
            if (methods != null) {
                sendMessage(MsgType.GETAPI, Status.SUCCESS, methods);
                return;
            }
        }
        sendMessage(MsgType.GETAPI, Status.FAILURE, null);
    }

    private void handleGetInvoke(Value raw) throws IOException {
        ArrayValue data = raw.asArrayValue();
//        if (DEBUG) {
//            int i = 0;
//            for (Value v : data) {
//                System.out.println("v[" + i++ + "]=" + v + " type=" + v.getValueType());
//            }
//        }
        String code = data.get(0).asStringValue().asString();
        boolean isQuery = data.get(1).asBooleanValue().getBoolean();
        byte[] from = data.get(2).asRawValue().asByteArray();
        byte[] to = data.get(3).asRawValue().asByteArray();
        byte[] value = data.get(4).asRawValue().asByteArray();
        byte[] limit = data.get(5).asRawValue().asByteArray();
        String method = data.get(6).asStringValue().asString();
        TypedObj[] params = TypedObj.decodeList(data.get(7).asArrayValue());

        if (DEBUG) {
            System.out.println(">>> code=" + code);
            System.out.println("    isQuery=" + isQuery);
            System.out.println("    from=" + Bytes.toHexString(from));
            System.out.println("      to=" + Bytes.toHexString(to));
            System.out.println("    value=" + Bytes.toHexString(value));
            System.out.println("    limit=" + Bytes.toHexString(limit));
            System.out.println("    method=" + method);
            System.out.println("    params={");
            int i = 0;
            for (TypedObj p : params) {
                System.out.printf("       [%d]=%s\n", i++, p);
            }
            System.out.println("    }");
        }
    }
}
