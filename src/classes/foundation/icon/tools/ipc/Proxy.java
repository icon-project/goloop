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

import foundation.icon.common.Address;
import foundation.icon.common.Bytes;
import org.msgpack.core.MessageBufferPacker;
import org.msgpack.core.MessagePack;
import org.msgpack.core.MessageUnpacker;
import org.msgpack.value.ArrayValue;
import org.msgpack.value.Value;

import java.io.IOException;
import java.math.BigInteger;

import static org.msgpack.value.ValueType.ARRAY;

public class Proxy {

    private static final boolean DEBUG = true;

    private OnGetApiListener mOnGetApiListener;
    private OnInvokeListener mOnInvokeListener;

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

    public static class TypedObj {
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

        void accept(MessageBufferPacker packer) throws IOException {
            System.out.println("=== TypedObj.accept() ===");
            System.out.println("  type: " + type);
            System.out.println("  value: " + value);
            packer.packArrayHeader(2);
            packer.packInt(type);
            if (type == NIL) {
                packer.packNil();
            } else if (type == STRING) {
                packer.packString((String) value);
            } else if (type == BOOL) {
                packer.packBoolean((Boolean) value);
            } else if (type == BYTES || type == ADDRESS || type == INT) {
                byte[] ba = (byte[]) value;
                packer.packBinaryHeader(ba.length);
                packer.writePayload(ba);
            } else {
                throw new IOException("not supported type: " + type);
            }
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
            } else if (arg instanceof byte[]) {
                byte[] ba = (byte[]) arg;
                packer.packBinaryHeader(ba.length);
                packer.writePayload(ba);
            } else if (arg instanceof BigInteger) {
                byte[] ba = ((BigInteger) arg).toByteArray();
                packer.packBinaryHeader(ba.length);
                packer.writePayload(ba);
            } else if (arg instanceof Method[]) {
                Method[] methods = (Method[]) arg;
                packer.packArrayHeader(methods.length);
                for (Method m : methods) {
                    m.accept(packer);
                }
            } else if (arg instanceof TypedObj) {
                TypedObj obj = (TypedObj) arg;
                obj.accept(packer);
            } else {
                throw new IOException("not yet supported: " + arg.getClass());
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

    public interface OnInvokeListener {
        InvokeResult onInvoke(String code, boolean isQuery, Address from, Address to,
                              BigInteger value, BigInteger limit, String method, TypedObj[] params) throws IOException;
    }

    public void setOnInvokeListener(OnInvokeListener listener) {
        mOnInvokeListener = listener;
    }

    private void handleGetInvoke(Value raw) throws IOException {
        ArrayValue data = raw.asArrayValue();
        String code = data.get(0).asStringValue().asString();
        boolean isQuery = data.get(1).asBooleanValue().getBoolean();
        Address from = new Address(data.get(2).asRawValue().asByteArray());
        Address to = new Address(data.get(3).asRawValue().asByteArray());
        BigInteger value = new BigInteger(data.get(4).asRawValue().asByteArray());
        BigInteger limit = new BigInteger(data.get(5).asRawValue().asByteArray());
        String method = data.get(6).asStringValue().asString();
        TypedObj[] params = TypedObj.decodeList(data.get(7).asArrayValue());

        if (DEBUG) {
            System.out.println(">>> code=" + code);
            System.out.println("    isQuery=" + isQuery);
            System.out.println("    from=" + from);
            System.out.println("      to=" + to);
            System.out.println("    value=" + value);
            System.out.println("    limit=" + limit);
            System.out.println("    method=" + method);
            System.out.println("    params={");
            int i = 0;
            for (TypedObj p : params) {
                System.out.printf("       [%d]=%s\n", i++, p);
            }
            System.out.println("    }");
        }

        if (mOnInvokeListener != null) {
            InvokeResult result = mOnInvokeListener.onInvoke(code, isQuery, from, to, value, limit, method, params);
            sendMessage(MsgType.RESULT, result.getStatus(), result.getStepUsed(), result.getResult());
        } else {
            throw new IOException("no invoke handler");
        }
    }
}
