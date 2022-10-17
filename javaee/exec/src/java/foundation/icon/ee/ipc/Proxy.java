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
import foundation.icon.ee.types.Method;
import foundation.icon.ee.util.MethodPacker;
import org.msgpack.core.MessageBufferPacker;
import org.msgpack.core.MessagePack;
import org.msgpack.core.MessageUnpacker;
import org.msgpack.value.ArrayValue;
import org.msgpack.value.Value;

import java.io.IOException;
import java.math.BigInteger;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import static org.msgpack.value.ValueType.ARRAY;

public abstract class Proxy {
    private final Connection client;
    private static final Logger logger = LoggerFactory.getLogger(Proxy.class);
    private final MessageUnpacker unpacker;

    public static class Message {
        public final int type;
        public final Value value;

        Message(int type, Value value) {
            this.type = type;
            this.value = value;
        }
    }

    protected Proxy(Connection client) {
        this.client = client;
        unpacker = MessagePack.newDefaultUnpacker(client.getInputStream());
    }

    public void close() throws IOException {
        this.client.close();
    }

    public Message getNextMessage() throws IOException {
        Value v = unpacker.unpackValue();
        if (v.getValueType() != ARRAY) {
            throw new IOException("should be array type");
        }
        ArrayValue a = v.asArrayValue();
        int type = a.get(0).asIntegerValue().toInt();
        Value value = a.get(1);
        return new Message(type, value);
    }

    public void sendMessage(int msgType, Object... args) throws IOException {
        MessageBufferPacker packer = MessagePack.newDefaultBufferPacker();
        try (packer) {
            packer.packArrayHeader(2);
            packer.packInt(msgType);
            if (args.length == 1) {
                packObject(args[0], packer);
            } else {
                packer.packArrayHeader(args.length);
                for (Object obj : args) {
                    packObject(obj, packer);
                }
            }
        }
        client.send(packer.toByteArray());
    }

    private void packObject(Object obj, MessageBufferPacker packer) throws IOException {
        if (obj == null) {
            packer.packNil();
        } else if (obj instanceof Boolean) {
            packer.packBoolean((boolean) obj);
        } else if (obj instanceof Integer) {
            packer.packInt((int) obj);
        } else if (obj instanceof String) {
            packer.packString((String) obj);
        } else if (obj instanceof byte[]) {
            packByteArray((byte[]) obj, packer);
        } else if (obj instanceof BigInteger) {
            packByteArray(((BigInteger) obj).toByteArray(), packer);
        } else if (obj instanceof Address) {
            packByteArray(((Address) obj).toByteArray(), packer);
        } else if (obj instanceof Method[]) {
            Method[] methods = (Method[]) obj;
            packer.packArrayHeader(methods.length);
            for (Method m : methods) {
                MethodPacker.writeTo(m, packer, false);
            }
        } else if (obj instanceof TypedObj) {
            TypedObj to = (TypedObj) obj;
            to.writeTo(packer);
        } else if (obj instanceof TypedObj[]) {
            TypedObj[] toa = (TypedObj[]) obj;
            packer.packArrayHeader(toa.length);
            for (TypedObj to : toa) {
                to.writeTo(packer);
            }
        } else if (obj instanceof byte[][]) {
            byte[][] bytesArray = (byte[][]) obj;
            packer.packArrayHeader(bytesArray.length);
            for (byte[] bytes : bytesArray) {
                packByteArray(bytes, packer);
            }
        } else if (obj instanceof Object[]) {
            Object[] oa = (Object[]) obj;
            packer.packArrayHeader(oa.length);
            for (Object o : oa) {
                packObject(o, packer);
            }
        } else {
            throw new IOException("not yet supported: " + obj.getClass());
        }
    }

    private void packByteArray(byte[] ba, MessageBufferPacker packer) throws IOException {
        packer.packBinaryHeader(ba.length);
        packer.writePayload(ba);
    }

    public abstract void handleMessages() throws IOException;
}
