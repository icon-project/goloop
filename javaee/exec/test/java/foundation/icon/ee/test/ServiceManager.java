package foundation.icon.ee.test;

import score.Address;
import foundation.icon.ee.ipc.Connection;
import foundation.icon.ee.ipc.EEProxy;
import foundation.icon.ee.ipc.Proxy;
import foundation.icon.ee.ipc.TypedObj;
import foundation.icon.ee.score.FileReader;
import foundation.icon.ee.tooling.deploy.OptimizedJarBuilder;
import foundation.icon.ee.types.Status;
import foundation.icon.ee.util.Crypto;
import org.aion.avm.core.util.ByteArrayWrapper;
import org.aion.avm.utilities.JarBuilder;
import org.msgpack.value.ArrayValue;

import java.io.IOException;
import java.math.BigInteger;
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;

import static foundation.icon.ee.ipc.EEProxy.Info;

public class ServiceManager extends Proxy {
    private State state = new State();
    private int nextScoreAddr = 1;
    private int nextExtAddr = 1;
    private BigInteger value = BigInteger.valueOf(0);
    private BigInteger stepLimit = BigInteger.valueOf(1_000_000_000);
    private State.Account current;
    private Address origin;
    private Map<String, Object> info = new HashMap<>();

    public ServiceManager(Connection conn) {
        super(conn);
        origin = newExternalAddress();
        info.put(Info.BLOCK_TIMESTAMP, BigInteger.valueOf(1000000));
        info.put(Info.BLOCK_HEIGHT, BigInteger.valueOf(10));
        info.put(Info.TX_HASH, Arrays.copyOf(new byte[]{1, 2}, 32));
        info.put(Info.TX_INDEX, BigInteger.valueOf(1));
        info.put(Info.TX_FROM, origin);
        info.put(Info.TX_TIMESTAMP, BigInteger.valueOf(1000000));
        info.put(Info.TX_NONCE, BigInteger.valueOf(2));
        info.put(Info.CONTRACT_OWNER, origin);
        current = state.getAccount(origin);
    }

    public static byte[] makeJar(Class<?> c) {
        return makeJar(c.getName(), new Class<?>[]{c});
    }

    public static byte[] makeJar(String name, Class<?>[] all) {
        byte[] preopt = JarBuilder.buildJarForExplicitMainAndClasses(name, all);
        return new OptimizedJarBuilder(true, preopt, true)
                .withUnreachableMethodRemover()
                .withRenamer().getOptimizedBytes();
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
        byte[] jar = makeJar(main);
        return doDeploy(jar, params);
    }

    public Address getOrigin() {
        return origin;
    }

    public Contract deploy(Class<?>[] all, Object ... params) {
        byte[] jar = makeJar(all[0].getName(), all);
        return doDeploy(jar, params);
    }

    private Contract doDeploy(byte[] jar, Object ... params) {
        Address scoreAddr = newScoreAddress();
        String path = getHexPrefix(scoreAddr) + "/optimized";
        try {
            var prev = current;
            var prevState = new State(state);
            state.writeFile(path, jar);
            current = state.getAccount(scoreAddr);
            info.put(Info.CONTRACT_OWNER, origin);
            var res = invoke(path, false, origin, scoreAddr, value, stepLimit, "<init>", params);
            if (res.getStatus()!=0) {
                state = prevState;
                current = state.getAccount(prev.address);
                throw new IllegalArgumentException(
                        String.format("deploy failed status=%d ret=%s",
                                res.getStatus(),
                                res.getResult()
                        )
                );
            }
            current = prev;
            return new Contract(this, scoreAddr);
        } catch (Exception e) {
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

    public FileReader getFileReader() {
        return state;
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

    private Object[] unpackByteArrayArray(ArrayValue arr) {
        var res = new Object[arr.size()];
        for (int i=0; i<res.length; i++) {
            res[i] = arr.get(i).asRawValue().asByteArray();
        }
        return res;
    }

    private Message waitFor(int type) throws IOException {
        while (true) {
            Message msg = getNextMessage();
            if (msg.type==type) {
                return msg;
            }
            switch(msg.type) {
                case EEProxy.MsgType.GETVALUE: {
                    var key = msg.value.asRawValue().asByteArray();
                    var value = current.storage.get(new ByteArrayWrapper(key));
                    printf("RECV getValue %s => %s%n", key, value);
                    sendMessage(EEProxy.MsgType.GETVALUE, value!=null, value);
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
                            sendMessage(EEProxy.MsgType.SETVALUE, false, 0);
                        } else {
                            sendMessage(EEProxy.MsgType.SETVALUE, true, old.length);
                        }
                    }
                    break;
                }
                case EEProxy.MsgType.CALL: {
                    var data = msg.value.asArrayValue();
                    var to = new Address(data.get(0).asRawValue().asByteArray());
                    var value = new BigInteger(data.get(1).asRawValue().asByteArray());
                    var stepLimit = new BigInteger(data.get(2).asRawValue().asByteArray());
                    String method = data.get(3).asStringValue().asString();
                    Object[] params = (Object[]) TypedObj.decodeAny(data.get(4));
                    BigInteger stepsContractCall = BigInteger.valueOf(5000);
                    stepLimit = stepLimit.subtract(stepsContractCall);
                    printf("RECV call to=%s value=%d stepLimit=%d method=%s params=%s%n",
                            to, value, stepLimit, method, params);
                    var res = invoke(to, value, stepLimit, method, params);
                    sendMessage(EEProxy.MsgType.RESULT, res.getStatus(),
                            res.getStepUsed().add(stepsContractCall),
                            TypedObj.encodeAny(res.getResult()));
                    break;
                }
                case EEProxy.MsgType.EVENT: {
                    var data = msg.value.asArrayValue();
                    var indexed = unpackByteArrayArray(data.get(0).asArrayValue());
                    var nonIndexed = unpackByteArrayArray(data.get(0).asArrayValue());
                    printf("RECV event indxed=%s data=%s%n", indexed, nonIndexed);
                    break;
                }
                case EEProxy.MsgType.GETBALANCE: {
                    var addr = new Address(msg.value.asRawValue().asByteArray());
                    var balance = state.getAccount(addr).balance;
                    sendMessage(EEProxy.MsgType.GETBALANCE, (Object) balance.toByteArray());
                    printf("RECV getBalance %s => %d%n", addr, balance);
                    break;
                }
                case EEProxy.MsgType.LOG: {
                    var data = msg.value.asArrayValue();
                    var level = data.get(0).asIntegerValue().asInt();
                    var logMsg = data.get(1).asStringValue().asString();
                    // filter only Context.println
                    if (logMsg.startsWith("org.aion.avm.core.BlockchainRuntimeImpl PRT|")) {
                        printf("RECV log level=%d %s%n", level, logMsg);
                    }
                    break;
                }
                case EEProxy.MsgType.SETCODE:{
                    var code = msg.value.asRawValue().asByteArray();
                    state.writeFile(getHexPrefix(current.address) + "/transformed", code);
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
                        sendMessage(EEProxy.MsgType.GETOBJGRAPH, nextHash, ogh, og);
                    } else {
                        printf("RECV getObjGraph flag=%d => next=%d hash=%s%n", flag, nextHash, ogh);
                        sendMessage(EEProxy.MsgType.GETOBJGRAPH, nextHash, ogh);
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
            }
        }
    }

    public Result invoke(Address to, BigInteger value, BigInteger stepLimit,
                         String method, Object[] params) throws IOException {
        return invoke(false, to, value, stepLimit, method, params);
    }

    public Result invoke(boolean query, Address to, BigInteger value, BigInteger stepLimit,
                         String method, Object[] params) throws IOException {
        var prev = current;
        var prevState = new State(state);
        var from = current.address;
        current = state.getAccount(to);
        info.put(Info.CONTRACT_OWNER, from);
        var code = getHexPrefix(to) + "/transformed";
        if (state.readFile(code) == null) {
            return new Result(
                    Status.ContractNotFound,
                    BigInteger.ZERO,
                    "Contract not found");
        }
        var res = invoke(code, query, from, to, value, stepLimit, method, params);
        if (res.getStatus()!=0) {
            state = prevState;
            current = state.getAccount(prev.address);
        } else {
            current = prev;
        }
        return res;
    }

    public Result invoke(String code, boolean isQuery, Address from,
                     Address to, BigInteger value, BigInteger stepLimit,
                     String method, Object[] params) throws IOException {
        printf("SEND invoke code=%s isQuery=%b from=%s to=%s value=%d stepLimit=%d method=%s params=%s%n",
                code, isQuery, from, to, value, stepLimit, method,
                params);
        sendMessage(EEProxy.MsgType.INVOKE, code, isQuery, from, to, value, stepLimit,
                method, TypedObj.encodeAny(params), TypedObj.encodeAny(info));
        var msg = waitFor(EEProxy.MsgType.RESULT);
        if (msg.type!= EEProxy.MsgType.RESULT) {
            throw new AssertionError(String.format("unexpected message type %d", msg.type));
        }
        var data = msg.value.asArrayValue();
        var status = data.get(0).asIntegerValue().asInt();
        var stepUsed = new BigInteger(data.get(1).asRawValue().asByteArray());
        var result = TypedObj.decodeAny(data.get(2));
        printf("RECV result status=%d stepUsed=%d ret=%s%n", status, stepUsed, result);
        return new Result(status, stepUsed, result);
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
}
