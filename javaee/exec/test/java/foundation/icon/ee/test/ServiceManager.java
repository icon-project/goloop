package foundation.icon.ee.test;

import foundation.icon.ee.Agent;
import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Method;
import foundation.icon.ee.types.Result;
import foundation.icon.ee.util.MethodUnpacker;
import foundation.icon.ee.ipc.Connection;
import foundation.icon.ee.ipc.EEProxy;
import foundation.icon.ee.ipc.Proxy;
import foundation.icon.ee.ipc.TypedObj;
import foundation.icon.ee.score.FileIO;
import foundation.icon.ee.tooling.deploy.OptimizedJarBuilder;
import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.StepCost;
import foundation.icon.ee.util.Crypto;
import org.aion.avm.core.util.ByteArrayWrapper;
import org.aion.avm.utilities.JarBuilder;
import org.msgpack.core.MessagePack;
import org.msgpack.value.ArrayValue;

import java.io.IOException;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;
import java.util.function.Consumer;

import static foundation.icon.ee.ipc.EEProxy.Info;

public class ServiceManager implements Agent {
    private final ArrayList<MyProxy> allProxies = new ArrayList<>();
    private MyProxy proxy;
    private State state = new State();
    private int nextScoreAddr = 1;
    private int nextExtAddr = 1;
    private BigInteger value = BigInteger.valueOf(0);
    private BigInteger stepLimit = BigInteger.valueOf(1_000_000_000);
    private State.Account current;
    private Address origin;
    private final Map<String, Object> info = new HashMap<>();
    private final StepCost stepCost;
    private boolean isReadOnly = false;
    private int eid = 0;
    private int exid = 0;
    private Indexer indexer;
    private Consumer<String> logger;

    private boolean isClassMeteringEnabled = true;

    public ServiceManager(Connection conn) {
        proxy = new MyProxy(conn);
        allProxies.add(proxy);
        origin = newExternalAddress();
        info.put(Info.BLOCK_TIMESTAMP, BigInteger.valueOf(1000000));
        info.put(Info.BLOCK_HEIGHT, BigInteger.valueOf(10));
        info.put(Info.TX_HASH, Arrays.copyOf(new byte[]{1, 2}, 32));
        info.put(Info.TX_INDEX, BigInteger.valueOf(1));
        info.put(Info.TX_FROM, origin);
        info.put(Info.TX_TIMESTAMP, BigInteger.valueOf(1000000));
        info.put(Info.TX_NONCE, BigInteger.valueOf(2));
        info.put(Info.CONTRACT_OWNER, origin);
        Map<String, BigInteger> stepCosts = new HashMap<>(Map.of(
                StepCost.GET, BigInteger.valueOf(40),
                StepCost.REPLACE, BigInteger.valueOf(80),
                StepCost.EVENT_LOG, BigInteger.valueOf(100),
                StepCost.DEFAULT_GET, BigInteger.valueOf(2000),
                StepCost.DEFAULT_SET, BigInteger.valueOf(20000),
                StepCost.REPLACE_BASE, BigInteger.valueOf(64),
                StepCost.DEFAULT_DELETE, BigInteger.valueOf(-10000),
                StepCost.EVENT_LOG_BASE, BigInteger.valueOf(64)
        ));
        info.put(Info.STEP_COSTS, stepCosts);
        stepCost = new StepCost(stepCosts);
        current = state.getAccount(origin);
    }

    public void setIndexer(Indexer indexer) {
        this.indexer = indexer;
    }

    public void setLogger(Consumer<String> logger) {
        this.logger = logger;
    }

    public void accept(Connection c) {
        allProxies.add(new MyProxy(c));
    }

    public static byte[] makeJar(Class<?> c) {
        return makeJar(c.getName(), new Class<?>[]{c});
    }

    public static byte[] makeJar(String name, Class<?>[] all) {
        byte[] preopt = JarBuilder.buildJarForExplicitMainAndClasses(name, all);
        return new OptimizedJarBuilder(true, preopt, true)
                .withUnreachableMethodRemover()
                .withRenamer().withLog(System.out).getOptimizedBytes();
    }

    public Address newScoreAddress() {
        var addr = new Address(Arrays.copyOf(new byte[]{
                1,
                (byte)(nextScoreAddr>>8),
                (byte)(nextScoreAddr)
        }, 21));
        nextScoreAddr++;
        return addr;
    }

    public Address newExternalAddress() {
        var addr = new Address(Arrays.copyOf(new byte[]{
                0,
                (byte) (nextExtAddr >> 8),
                (byte) (nextExtAddr)
        }, 21));
        nextExtAddr++;
        return addr;
    }

    public Contract deploy(Class<?> main, Object ... params) {
        ++exid;
        eid = 0;
        byte[] jar = makeJar(main);
        return doDeploy(jar, params);
    }

    public Address getOrigin() {
        return origin;
    }

    public Contract deploy(Class<?>[] all, Object ... params) {
        ++exid;
        eid = 0;
        byte[] jar = makeJar(all[0].getName(), all);
        return doDeploy(jar, params);
    }

    private Method[] getAPI(String path) throws IOException {
        printf("SEND getAPI %s%n", path);
        proxy.sendMessage(EEProxy.MsgType.GETAPI, path);
        var msg = waitFor(EEProxy.MsgType.GETAPI);
        var packer = MessagePack.newDefaultBufferPacker();
        var arr = msg.value.asArrayValue();
        var status = arr.get(0).asIntegerValue().asInt();
        if (status!=0) {
            printf("RECV getAPI status=%d%n", status);
            return null;
        }
        var methods = arr.get(1);
        methods.writeTo(packer);
        var res = MethodUnpacker.readFrom(packer.toByteArray(), false);
        printf("RECV getAPI status=%d methods=[%n", status);
        for (var m : res) {
            printf("    %s%n", m);
        }
        printf("]%n");
        return res;
    }

    private Contract doDeploy(byte[] jar, Object ... params) {
        Address scoreAddr = newScoreAddress();
        String path = getHexPrefix(scoreAddr);
        try {
            var prev = current;
            var prevState = new State(state);
            state.writeFile(path + "/code.jar", jar);
            Method[] methods = getAPI(path);
            if (methods==null) {
                throw new TransactionException(new Result(Status.IllegalFormat,
                        0, null));
            }
            current = state.getAccount(scoreAddr);
            info.put(Info.CONTRACT_OWNER, origin);
            var res = doInvoke(path, false, origin, scoreAddr, value, stepLimit, "<init>", params);
            if (res.getStatus()!=0) {
                state = prevState;
                current = state.getAccount(prev.address);
                throw new TransactionException(res);
            }
            var contract = new Contract(ServiceManager.this, scoreAddr, methods);
            current.contract = contract;
            current = state.getAccount(prev.address);
            return contract;
        } catch (IOException e) {
            throw new AssertionError(e);
        }
    }

    public BigInteger getValue() {
        return value;
    }

    public BigInteger getStepLimit() {
        return stepLimit;
    }

    public void setStepLimit(BigInteger sl) {
        stepLimit = sl;
    }

    public FileIO getFileIO() {
        return new FileIO() {
            public byte[] readFile(String path) {
                return state.readFile(path);
            }

            public void writeFile(String path, byte[] bytes) {
                state.writeFile(path, bytes);
            }
        };
    }

    private Object[] unpackByteArrayArray(ArrayValue arr) {
        var res = new Object[arr.size()];
        for (int i=0; i<res.length; i++) {
            res[i] = arr.get(i).asRawValue().asByteArray();
        }
        return res;
    }

    private Proxy.Message waitFor(int type) throws IOException {
        while (true) {
            Proxy.Message msg = proxy.getNextMessage();
            if (msg.type==type) {
                return msg;
            }
            switch(msg.type) {
                case EEProxy.MsgType.GETVALUE: {
                    var key = msg.value.asRawValue().asByteArray();
                    var value = current.storage.get(new ByteArrayWrapper(key));
                    printf("RECV getValue %s => %s%n", key, value);
                    proxy.sendMessage(EEProxy.MsgType.GETVALUE, value!=null, value);
                    break;
                }
                case EEProxy.MsgType.SETVALUE: {
                    var data = msg.value.asArrayValue();
                    var key = data.get(0).asRawValue().asByteArray();
                    var flag = data.get(1).asIntegerValue().toInt();
                    byte[] old;
                    if ((flag & EEProxy.SetValueFlag.DELETE) != 0) {
                        old = current.storage.remove(new ByteArrayWrapper(key));
                        printf("RECV setValue %s isDelete=%b%n", key, true);
                    } else {
                        var value = data.get(2).asRawValue().asByteArray();
                        old = current.storage.put(new ByteArrayWrapper(key), value);
                        printf("RECV setValue %s isDelete=%b %s%n", key, false, value);
                    }
                    if ((flag & EEProxy.SetValueFlag.OLDVALUE) != 0) {
                        if (old == null) {
                            proxy.sendMessage(EEProxy.MsgType.SETVALUE, false, 0);
                        } else {
                            proxy.sendMessage(EEProxy.MsgType.SETVALUE, true, old.length);
                        }
                    }
                    break;
                }
                case EEProxy.MsgType.CALL: {
                    var data = msg.value.asArrayValue();
                    var to = new Address(data.get(0).asRawValue().asByteArray());
                    var value = new BigInteger(data.get(1).asRawValue().asByteArray());
                    var stepLimit = new BigInteger(data.get(2).asRawValue().asByteArray());
                    String dataType = data.get(3).asStringValue().asString();
                    @SuppressWarnings("unchecked")
                    var dataObj = (Map<String, Object>) TypedObj.decodeAny(data.get(4));
                    assert "call".equals(dataType);
                    assert dataObj != null;
                    String method = (String) dataObj.get("method");
                    Object[] params = (Object[]) dataObj.get("params");
                    BigInteger stepsContractCall = BigInteger.valueOf(5000);
                    stepLimit = stepLimit.subtract(stepsContractCall);
                    printf("RECV call to=%s value=%d stepLimit=%d method=%s params=%s%n",
                            to, value, stepLimit, method, params);
                    current.eid = eid;
                    var res = invokeInner(to, value, stepLimit, method, params);
                    printf("SEND result status=%d stepUsed=%d ret=%s EID=%d prevEID=%d%n",
                            res.getStatus(), res.getStepUsed(), res.getRet(),
                            eid, current.eid);
                    proxy.sendMessage(EEProxy.MsgType.RESULT, res.getStatus(),
                            res.getStepUsed().add(stepsContractCall),
                            TypedObj.encodeAny(res.getRet()), eid, current.eid);
                    break;
                }
                case EEProxy.MsgType.EVENT: {
                    var data = msg.value.asArrayValue();
                    var indexed = unpackByteArrayArray(data.get(0).asArrayValue());
                    var nonIndexed = unpackByteArrayArray(data.get(1).asArrayValue());
                    printf("RECV event indxed=%s data=%s%n", indexed, nonIndexed);
                    break;
                }
                case EEProxy.MsgType.GETBALANCE: {
                    var addr = new Address(msg.value.asRawValue().asByteArray());
                    var balance = state.getAccount(addr).balance;
                    proxy.sendMessage(EEProxy.MsgType.GETBALANCE, (Object) balance.toByteArray());
                    printf("RECV getBalance %s => %d%n", addr, balance);
                    break;
                }
                case EEProxy.MsgType.LOG: {
                    var data = msg.value.asArrayValue();
                    var level = data.get(0).asIntegerValue().asInt();
                    var logMsg = data.get(1).asStringValue().asString();
                    if (logger != null) {
                        logger.accept(logMsg);
                    }
                    // filter only Context.println
                    if (logMsg.startsWith("org.aion.avm.core.BlockchainRuntimeImpl PRT|")
                            || logMsg.startsWith("s.java.lang.Throwable PRT|")) {
                        printf("RECV log level=%d %s%n", level, logMsg);
                    }
                    break;
                }
                case EEProxy.MsgType.SETCODE:{
                    var code = msg.value.asRawValue().asByteArray();
                    state.writeFile(getHexPrefix(current.address) + "/transformed.jar", code);
                    printf("RECV setCode hash=%s len=%d%n", Crypto.sha3_256(code), code.length);
                    break;
                }
                case EEProxy.MsgType.GETOBJGRAPH: {
                    var flag = msg.value.asIntegerValue().asInt();
                    var nextHash = current.nextHash;
                    var ogh = current.objectGraphHash;
                    if ((flag&1)!=0) {
                        var og = current.objectGraph;
                        printf("RECV getObjGraph flag=%d => next=%d hash=%s graphLen=%d graph=%s%n", flag, nextHash, ogh, og.length, beautifyObjectGraph(og));
                        proxy.sendMessage(EEProxy.MsgType.GETOBJGRAPH, nextHash, ogh, og);
                    } else {
                        printf("RECV getObjGraph flag=%d => next=%d hash=%s%n", flag, nextHash, ogh);
                        proxy.sendMessage(EEProxy.MsgType.GETOBJGRAPH, nextHash, ogh);
                    }
                    break;
                }
                case EEProxy.MsgType.SETOBJGRAPH: {
                    var data = msg.value.asArrayValue();
                    var flag = data.get(0).asIntegerValue().asInt();
                    var nextHash = data.get(1).asIntegerValue().asInt();
                    current.nextHash = nextHash;
                    if ((flag&1)!=0) {
                        var og = data.get(2).asRawValue().asByteArray();
                        current.objectGraphHash = Crypto.sha3_256(og);
                        current.objectGraph = og;
                        printf("RECV setObjGraph flag=%d next=%d hash=%s graphLen=%d graph=%s%n", flag, nextHash, current.objectGraphHash, og.length, beautifyObjectGraph(og));
                    } else {
                        printf("RECV setObjGraph flag=%d next=%d%n", flag, nextHash);
                    }
                    break;
                }
                case EEProxy.MsgType.SETFEEPCT: {
                    var proportion = msg.value.asIntegerValue().asInt();
                    if (0 <= proportion && proportion <= 100) {
                        printf("RECV setProportion %d%n", proportion);
                    } else {
                        printf("RECV setProportion OutOfRange p=%d%n", proportion);
                    }
                }
            }
        }
    }

    public Result invoke(Address to, BigInteger value, BigInteger stepLimit,
                         String method, Object[] params) throws IOException {
        ++exid;
        eid = 0;
        return doInvoke(false, to, value, stepLimit, method, params);
    }

    private Result invokeInner(Address to, BigInteger value, BigInteger stepLimit,
                         String method, Object[] params) throws IOException {
        return doInvoke(false, to, value, stepLimit, method, params);
    }

    public Result invoke(boolean query, Address to, BigInteger value, BigInteger stepLimit,
                         String method, Object[] params) throws IOException {
        ++exid;
        eid = 0;
        return doInvoke(query, to, value, stepLimit, method, params);
    }

    private Result doInvoke(boolean query, Address to, BigInteger value, BigInteger stepLimit,
                         String method, Object[] params) throws IOException {
        var prev = current;
        var prevState = new State(state);
        var from = current.address;
        current = state.getAccount(to);
        info.put(Info.CONTRACT_OWNER, from);
        var code = getHexPrefix(to) + "/transformed.jar";
        if (state.readFile(code) == null) {
            state = prevState;
            current = state.getAccount(prev.address);
            return new Result(
                    Status.ContractNotFound,
                    BigInteger.ZERO,
                    "Contract not found");
        }
        var res = doInvoke(getHexPrefix(to), query, from, to, value, stepLimit, method, params);
        if (res.getStatus()!=0) {
            state = prevState;
            current = state.getAccount(prev.address);
        } else {
            current = state.getAccount(prev.address);
        }
        return res;
    }

    private Result doInvoke(String code, boolean isQuery, Address from,
                            Address to, BigInteger value, BigInteger stepLimit,
                            String method, Object[] params) throws IOException {
        boolean readOnlyMethod = false;
        if (!method.equals("<init>")) {
            var m = current.contract.getMethod(method);
            readOnlyMethod = (m.getFlags()&Method.Flags.READONLY) != 0;
        }
        var prevIsReadOnly = isReadOnly;
        if (isQuery || isReadOnly || readOnlyMethod) {
            isReadOnly = true;
            info.remove(Info.TX_HASH);
        } else {
            info.put(Info.TX_HASH, Arrays.copyOf(new byte[]{1, 2}, 32));
        }
        try {
            Object[] codeState = null;
            ++eid;
            if (current.objectGraph != null) {
                if (current.exid != exid) {
                    current.exid = exid;
                    current.eid = 0;
                }
                codeState = new Object[]{
                        current.nextHash,
                        current.objectGraphHash,
                        current.eid
                };
            }
            var prevProxy = proxy;
            if (indexer != null) {
                var index = indexer.getIndex(to);
                proxy = allProxies.get(index);
                printf("SEND invoke EE=%d code=%s isQuery=%b from=%s to=%s value=%d stepLimit=%d method=%s params=%s EID=%d codeState=%s%n",
                        index, code, isReadOnly, from, to, value, stepLimit,
                        method, params, eid, codeState);
            } else {
                printf("SEND invoke code=%s isQuery=%b from=%s to=%s value=%d stepLimit=%d method=%s params=%s EID=%d codeState=%s%n",
                        code, isReadOnly, from, to, value, stepLimit, method,
                        params, eid, codeState);
            }
            // TODO need to get proper codeId
            byte[] codeId = null;
            proxy.sendMessage(EEProxy.MsgType.INVOKE, code, isReadOnly, from,
                    to, value, stepLimit, method, TypedObj.encodeAny(params),
                    TypedObj.encodeAny(info), codeId, eid, codeState);
            var msg = waitFor(EEProxy.MsgType.RESULT);
            proxy = prevProxy;
            if (msg.type != EEProxy.MsgType.RESULT) {
                throw new AssertionError(String.format("unexpected message type %d", msg.type));
            }
            var data = msg.value.asArrayValue();
            var status = data.get(0).asIntegerValue().asInt();
            var stepUsed = new BigInteger(data.get(1).asRawValue().asByteArray());
            var result = TypedObj.decodeAny(data.get(2));
            current.eid = eid++;
            printf("RECV result status=%d stepUsed=%d ret=%s%n", status, stepUsed, result);
            return new Result(status, stepUsed, result);
        } finally {
            isReadOnly = prevIsReadOnly;
        }
    }

    private static boolean isPrint(int ch) {
        return ch >= 0x20 && ch <= 0x7e;
    }

    private static final char[] HEX_ARRAY = "0123456789abcdef".toCharArray();

    public static String toHex(byte[] bytes) {
        char[] hexChars = new char[bytes.length * 2];
        for (int j = 0; j < bytes.length; j++) {
            int v = bytes[j] & 0xFF;
            hexChars[j * 2] = HEX_ARRAY[v >>> 4];
            hexChars[j * 2 + 1] = HEX_ARRAY[v & 0x0F];
        }
        return new String(hexChars);
    }

    private static String beautifyObjects(Object[] params) {
        StringBuilder sb = new StringBuilder();
        sb.append('[');
        for (int i=0; i<params.length; i++) {
            if (i>0) {
                sb.append(' ');
            }
            sb.append(beautify(params[i]));
        }
        sb.append(']');
        return sb.toString();
    }

    private static String getHexPrefix(Address addr) {
        return getHexPrefix(addr, 3);
    }

    private static String getHexPrefix(Address addr, int len) {
        return toHex(Arrays.copyOf(addr.toByteArray(), len));
    }

    private static Object beautify(Object o) {
        if (o==null) {
            return "<null>";
        } else if (o instanceof byte[]) {
            return toHex((byte[])o);
        } else if (o instanceof Object[]) {
            return beautifyObjects((Object[])o);
        } else if (o instanceof Address) {
            var a = (Address) o;
            return getHexPrefix(a) + "...";
        } else if (o instanceof Map) {
            var m = (Map<?, ?>) o;
            var es = m.entrySet();
            StringBuilder sb = new StringBuilder();
            sb.append('{');
            boolean first = true;
            for (Map.Entry<?, ?> e : es) {
                if (first) {
                    first =false;
                } else {
                    sb.append(", ");
                }
                sb.append(beautify(e.getKey()));
                sb.append('=');
                sb.append(beautify(e.getValue()));
            }
            sb.append('}');
            return sb.toString();
        }
        return o;
    }

    private static String beautifyObjectGraph(byte[] og) {
        StringBuilder sb = new StringBuilder();
        for (byte b : og) {
            if (isPrint(b)) {
                sb.append((char) b);
            } else {
                int v = b & 0xFF;
                char c1 = HEX_ARRAY[v >>> 4];
                char c2 = HEX_ARRAY[v & 0x0F];
                sb.append("\\x").append(c1).append(c2);
            }
        }
        return sb.toString();
    }

    private void printf(String fmt, Object... inObjs) {
        var outObjs = new Object[inObjs.length];
        for (int i=0; i<inObjs.length; i++) {
            outObjs[i] = beautify(inObjs[i]);
        }
        System.out.printf(fmt, outObjs);
    }

    public StepCost getStepCost() {
        return stepCost;
    }

    public boolean isClassMeteringEnabled() {
        return isClassMeteringEnabled;
    }

    public void enableClassMetering(boolean e) {
        isClassMeteringEnabled = e;
    }

    public void close() {
        for (var p : allProxies) {
            p.close();
        }
    }

    private class MyProxy extends Proxy {
        public MyProxy(Connection client) {
            super(client);
        }

        public void close() {
            try {
                sendMessage(EEProxy.MsgType.CLOSE);
                super.close();
            } catch (IOException e) {
                throw new AssertionError(e);
            }
        }

        public void handleMessages() throws IOException {
            waitFor(EEProxy.MsgType.RESULT);
        }
    }
}
