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

import foundation.icon.ee.score.ValidationException;
import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Method;
import foundation.icon.ee.types.ObjectGraph;
import foundation.icon.ee.types.Result;
import foundation.icon.ee.types.Status;
import i.RuntimeAssertionError;
import org.aion.avm.core.IExternalState;
import org.msgpack.core.MessageTypeCastException;
import org.msgpack.value.ArrayValue;
import org.msgpack.value.Value;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Map;
import java.util.function.IntConsumer;
import java.util.zip.ZipException;

public class EEProxy extends Proxy {
    private static final Logger logger = LoggerFactory.getLogger(EEProxy.class);
    private static final ThreadLocal<EEProxy> threadLocal = new ThreadLocal<>();

    public static final int LOG_PANIC = 0;
    public static final int LOG_FATAL = 1;
    public static final int LOG_ERROR = 2;
    public static final int LOG_WARN = 3;
    public static final int LOG_INFO = 4;
    public static final int LOG_DEBUG = 5;
    public static final int LOG_TRACE = 6;

    private static final int MAX_PREV_SIZE_CB = 32;

    private OnGetApiListener mOnGetApiListener;
    private OnInvokeListener mOnInvokeListener;
    private final ArrayList<IntConsumer> mPrevSizeCBs = new ArrayList<>();
    private boolean isTrace = false;

    public static class MsgType {
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
        public static final int LOG = 10;
        public static final int CLOSE = 11;
        public static final int SETCODE = 12;
        public static final int GETOBJGRAPH = 13;
        public static final int SETOBJGRAPH = 14;
        public static final int SETFEEPCT = 15;
    }

    public static class SetValueFlag {
        public static final int DELETE = 1;
        public static final int OLDVALUE = 2;
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
        public static final String REVISION = "Revision";
    }

    public EEProxy(Connection client) {
        super(client);
        threadLocal.set(this);
    }

    public static EEProxy getProxy() {
        return threadLocal.get();
    }

    public void connect(String uuid) throws IOException {
        sendMessage(MsgType.VERSION, 1, uuid, "java");
    }

    public void close() throws IOException {
        super.close();
    }

    public void handleMessages() throws IOException {
        doHandleMessages();
    }

    private Value doHandleMessages() throws IOException {
        while (true) {
            Message msg = getNextMessage();
            switch (msg.type) {
                case MsgType.GETAPI:
                    String path = msg.value.asStringValue().asString();
                    logger.trace("[GETAPI] path={}", path);
                    handleGetApi(path);
                    break;
                case MsgType.INVOKE:
                    logger.trace("[INVOKE]");
                    handleInvoke(msg.value);
                    break;
                case MsgType.RESULT:
                    logger.trace("[RESULT]");
                    return msg.value;
                case MsgType.CLOSE:
                    // TODO: unwind stack
                    logger.trace("[CLOSE]");
                    return null; // exit loop
            }
        }
    }

    public BigInteger getBalance(Address addr) throws IOException {
        sendMessage(MsgType.GETBALANCE, addr);
        waitForCallbacks();
        Message msg = getNextMessage();
        if (msg.type != MsgType.GETBALANCE) {
            throw new IOException("Invalid message: GETBALANCE expected.");
        }
        BigInteger balance = new BigInteger(getValueAsByteArray(msg.value));
        logger.trace("[GETBALANCE] {}", balance);
        return balance;
    }

    public byte[] getValue(byte[] key) throws IOException {
        sendMessage(MsgType.GETVALUE, (Object) key);
        waitForCallbacks();
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

    public void setValue(byte[] key, byte[] value, IntConsumer prevSizeCB) throws IOException {
        int flag = 0;
        if (value == null) {
            flag |= SetValueFlag.DELETE;
        }
        if (prevSizeCB != null) {
            flag |= SetValueFlag.OLDVALUE;
            mPrevSizeCBs.add(prevSizeCB);
        }
        sendMessage(MsgType.SETVALUE, key, flag, value);
    }

    public boolean waitForCallback() throws IOException {
        if (mPrevSizeCBs.isEmpty()) {
            return false;
        }
        Message msg = getNextMessage();
        if (msg.type != MsgType.SETVALUE) {
            throw new IOException("Invalid message: SETVALUE expected.");
        }
        logger.trace("[SETVALUE]");
        handleSetValue(msg.value);
        return true;
    }

    @SuppressWarnings("StatementWithEmptyBody")
    public void waitForCallbacks() throws IOException {
        while(waitForCallback());
    }

    public void limitPendingCallbackLength() throws IOException {
        while (mPrevSizeCBs.size() > MAX_PREV_SIZE_CB) {
            waitForCallback();
        }
    }

    public void setCode(byte[] code) throws IOException {
        sendMessage(MsgType.SETCODE, (Object) code);
    }

    public ObjectGraph getObjGraph(boolean flag) throws IOException {
        sendMessage(MsgType.GETOBJGRAPH, flag ? 1 : 0);
        waitForCallbacks();
        Message msg = getNextMessage();
        if (msg.type != MsgType.GETOBJGRAPH) {
            throw new IOException("Invalid message: GETOBJGRAPH expected.");
        }
        ArrayValue data = msg.value.asArrayValue();
        int nextHash = data.get(0).asIntegerValue().asInt();
        byte[] graphHash = getValueAsByteArray(data.get(1));
        byte[] graphData = flag ? getValueAsByteArray(data.get(2)) : null;
        ObjectGraph objGraph = new ObjectGraph(nextHash, graphHash, graphData);
        logger.trace("[GETOBJGRAPH] {}", objGraph);
        return objGraph;
    }

    public void setObjGraph(boolean flags, ObjectGraph objectGraph) throws IOException {
        logger.trace("[SETOBJGRAPH] {}, {}", flags, objectGraph);
        sendMessage(MsgType.SETOBJGRAPH,
                flags ? 1 : 0,
                objectGraph.getNextHash(),
                flags ? objectGraph.getGraphData() : null);
    }

    public void log(int level, int flag, String msg) throws IOException {
        sendMessage(MsgType.LOG, level, flag, msg);
    }

    public void event(byte[][]indexed, byte[][] data) throws IOException {
        logger.trace("[LOGEVENT] {}, {}", indexed, data);
        sendMessage(MsgType.EVENT, indexed, data);
    }

    public void setFeeSharingProportion(int proportion) throws IOException {
        logger.trace("[SETFEEPCT] {}", proportion);
        sendMessage(MsgType.SETFEEPCT, proportion);
    }

    public interface OnGetApiListener {
        Method[] onGetApi(String path) throws IOException,
                ValidationException;
    }

    public void setOnGetApiListener(OnGetApiListener listener) {
        mOnGetApiListener = listener;
    }

    private void handleGetApi(String path) throws IOException {
        if (mOnGetApiListener == null) {
            RuntimeAssertionError.unreachable("no getAPI handler");
        }
        try {
            Method[] methods = mOnGetApiListener.onGetApi(path);
            sendMessage(MsgType.GETAPI, Status.Success, methods);
        } catch (ZipException e) {
            e.printStackTrace();
            sendMessage(MsgType.GETAPI, Status.PackageError, null);
        } catch (ValidationException e) {
            e.printStackTrace();
            sendMessage(MsgType.GETAPI, Status.IllegalFormat, null);
        }
    }

    public interface OnInvokeListener {
        InvokeResult onInvoke(String code, int option, Address from, Address to,
                              BigInteger value, BigInteger limit,
                              String method, Object[] params,
                              Map<String, Object> info,
                              byte[] contractID, int eid, int nextHash,
                              byte[] graphHash, int prevEID) throws IOException;
    }

    public void setOnInvokeListener(OnInvokeListener listener) {
        mOnInvokeListener = listener;
    }

    private byte[] getValueAsByteArray(Value value) {
        return value.asRawValue().asByteArray();
    }

    private Address getValueAsAddress(Value value) {
        if (!value.isNilValue()) {
            return new Address(getValueAsByteArray(value));
        }
        return null;
    }

    private void handleInvoke(Value raw) throws IOException {
        try {
            ArrayValue data = raw.asArrayValue();
            String code = data.get(0).asStringValue().asString();
            int option = data.get(1).asIntegerValue().asInt();
            Address from = getValueAsAddress(data.get(2));
            Address to = getValueAsAddress(data.get(3));
            BigInteger value = new BigInteger(getValueAsByteArray(data.get(4)));
            BigInteger limit = new BigInteger(getValueAsByteArray(data.get(5)));
            String method = data.get(6).asStringValue().asString();
            Object[] params = (Object[]) TypedObj.decodeAny(data.get(7));
            @SuppressWarnings("unchecked")
            var info = (Map<String, Object>) TypedObj.decodeAny(data.get(8));
            var contractID = getValueAsByteArray(data.get(9));
            int eid = data.get(10).asIntegerValue().asInt();
            int nextHash = 0;
            byte[] graphHash = null;
            int prevEID = 0;
            Value state_ = data.get(11);
            if (state_.isArrayValue()) {
                var state = state_.asArrayValue();
                nextHash = state.get(0).asIntegerValue().asInt();
                graphHash = state.get(1).asRawValue().asByteArray();
                prevEID = state.get(2).asIntegerValue().asInt();
            }

            if (mOnInvokeListener != null) {
                boolean oldIsTrace = isTrace;
                isTrace = (option & IExternalState.OPTION_TRACE) != 0;
                InvokeResult result = mOnInvokeListener.onInvoke(
                        code, option, from, to, value, limit, method, params,
                        info, contractID, eid, nextHash, graphHash, prevEID);
                sendMessage(MsgType.RESULT, result.getStatus(), result.getStepUsed(), result.getResult());
                isTrace = oldIsTrace;
            } else {
                throw new IOException("no invoke handler");
            }
        } catch (MessageTypeCastException e) {
            String errMsg = "MessagePack casting error";
            logger.warn(errMsg, e);
            sendMessage(MsgType.RESULT, Status.UnknownFailure, BigInteger.ZERO, TypedObj.encodeAny(errMsg));
        }
    }

    public Result call(Address addr, BigInteger value, long stepLimit,
                       String dataType, Object dataObj) throws IOException {
        // send message first
        var limit = BigInteger.valueOf(stepLimit);
        var typedObj = TypedObj.encodeAny(dataObj);
        sendMessage(MsgType.CALL, addr, value, limit, dataType, typedObj);

        // handle result
        Value raw = doHandleMessages();
        if (raw==null) {
            throw new IOException("close message");
        }
        ArrayValue data = raw.asArrayValue();
        int status = data.get(0).asIntegerValue().asInt();
        BigInteger stepUsed = new BigInteger(getValueAsByteArray(data.get(1)));
        Object res = TypedObj.decodeAny(data.get(2));
        int eid = data.get(3).asIntegerValue().asInt();
        int prevEID = data.get(4).asIntegerValue().asInt();
        return new Result(status, stepUsed, res, eid, prevEID);
    }

    private void handleSetValue(Value raw) throws IOException {
        try {
            ArrayValue data = raw.asArrayValue();
            boolean hasOld = data.get(0).asBooleanValue().getBoolean();
            int prevSize = data.get(1).asIntegerValue().asInt();
            var cb = mPrevSizeCBs.remove(0);
            if (!hasOld) {
                cb.accept(-1);
            } else {
                cb.accept(prevSize);
            }
        } catch (MessageTypeCastException e) {
            String errMsg = "MessagePack casting error";
            logger.warn(errMsg, e);
            throw new IOException("no invoke handler");
        }
    }

    public boolean isTrace() {
        return isTrace;
    }
}
