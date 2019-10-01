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
import org.msgpack.value.ArrayValue;
import org.msgpack.value.Value;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.math.BigInteger;

public class EEProxy extends Proxy {
    private static final Logger logger = LoggerFactory.getLogger(EEProxy.class);

    private OnGetApiListener mOnGetApiListener;
    private OnInvokeListener mOnInvokeListener;

    static class MsgType {
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
        static final int LOG = 10;
        static final int CLOSE = 11;
    }

    public static class Status {
        public static final int SUCCESS = 0;
        public static final int FAILURE = 1;
    }

    public static class Info {
        public static final String BLOCK_TIMESTAMP = "B.timestamp";
        public static final String BLOCK_HEIGHT = "B.height";
        public static final String TX_HASH = "T.hash";
        public static final String TX_INDEX = "T.index";
        public static final String TX_FROM = "T.from";
        public static final String TX_TIMESTAMP = "T.timestamp";
        public static final String TX_NONCE = "T.nonce";
        public static final String STEP_COSTS = "StepCosts";
        public static final String CONTRACT_OWNER = "C.owner";
    }

    public EEProxy(Client client) {
        super(client, logger);
    }

    public void connect(String uuid) throws IOException {
        sendMessage(MsgType.VERSION, 1, uuid, "java");
    }

    public void close() throws IOException {
        this.client.close();
    }

    public void handleMessages() throws IOException {
        while (true) {
            Message msg = getNextMessage();
            switch (msg.type) {
                case MsgType.GETAPI:
                    String path = msg.value.asStringValue().asString();
                    logger.debug("[GETAPI] path={}", path);
                    handleGetApi(path);
                    break;
                case MsgType.INVOKE:
                    logger.debug("[INVOKE]");
                    handleInvoke(msg.value);
                    break;
                case MsgType.CLOSE:
                    logger.debug("[CLOSE]");
                    return; // exit loop
            }
        }
    }

    public Object getInfo() throws IOException {
        sendMessage(MsgType.GETINFO, (Object)null);
        Message msg = getNextMessage();
        if (msg.type != MsgType.GETINFO) {
            throw new IOException("Invalid message: GETINFO expected.");
        }
        logger.debug("[GETINFO]");
        return TypedObj.decodeAny(msg.value);
    }

    public BigInteger getBalance(Address addr) throws IOException {
        logger.debug("sendMessage to get balance : " + addr);
        sendMessage(MsgType.GETBALANCE, addr);
        Message msg = getNextMessage();
        if (msg.type != MsgType.GETBALANCE) {
            throw new IOException("Invalid message: GETBALANCE expected.");
        }
        BigInteger balance = new BigInteger(getValueAsByteArray(msg.value));
        logger.debug("[GETBALANCE] {}", balance);
        return balance;
    }

    public byte[] getValue(byte[] key) throws IOException {
        sendMessage(MsgType.GETVALUE, (Object) key);
        Message msg = getNextMessage();
        if (msg.type != MsgType.GETVALUE) {
            throw new IOException("Invalid message: GETVALUE expected.");
        }
        ArrayValue data = msg.value.asArrayValue();
        if (data.get(0).asBooleanValue().getBoolean()) {
            return getValueAsByteArray(data.get(1));
        } else {
            return null;
        }
    }

    public void setValue(byte[] key, byte[] value) throws IOException {
        if (value == null) {
            sendMessage(MsgType.SETVALUE, key, true, null);
        } else {
            sendMessage(MsgType.SETVALUE, key, false, value);
        }
    }

    public interface OnGetApiListener {
        Method[] onGetApi(String path) throws IOException;
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
                              BigInteger value, BigInteger limit, String method, Object[] params) throws IOException;
    }

    public void setOnInvokeListener(OnInvokeListener listener) {
        mOnInvokeListener = listener;
    }

    protected byte[] getValueAsByteArray(Value value) {
        return value.asRawValue().asByteArray();
    }

    private void handleInvoke(Value raw) throws IOException {
        ArrayValue data = raw.asArrayValue();
        String code = data.get(0).asStringValue().asString();
        System.out.println("code = " + code);
        boolean isQuery = data.get(1).asBooleanValue().getBoolean();
        Address from = new Address(getValueAsByteArray(data.get(2)));
        Address to = new Address(getValueAsByteArray(data.get(3)));
        BigInteger value = new BigInteger(getValueAsByteArray(data.get(4)));
        BigInteger limit = new BigInteger(getValueAsByteArray(data.get(5)));
        String method = data.get(6).asStringValue().asString();
        Object[] params = (Object[]) TypedObj.decodeAny(data.get(7));

        if (mOnInvokeListener != null) {
            InvokeResult result = mOnInvokeListener.onInvoke(code, isQuery, from, to, value, limit, method, params);
            sendMessage(MsgType.RESULT, result.getStatus(), result.getStepUsed(), result.getResult());
        } else {
            throw new IOException("no invoke handler");
        }
    }
}
