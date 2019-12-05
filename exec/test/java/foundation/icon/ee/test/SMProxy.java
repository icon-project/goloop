package foundation.icon.ee.test;

import avm.Address;
import foundation.icon.ee.ipc.Connection;
import foundation.icon.ee.ipc.EEProxy;
import foundation.icon.ee.ipc.Proxy;
import foundation.icon.ee.ipc.TypedObj;
import foundation.icon.ee.tooling.deploy.OptimizedJarBuilder;
import foundation.icon.ee.utils.Crypto;
import org.aion.avm.core.util.ByteArrayWrapper;
import org.aion.avm.utilities.JarBuilder;
import org.msgpack.value.ArrayValue;

import java.io.IOException;
import java.math.BigInteger;
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;

import static foundation.icon.ee.ipc.EEProxy.Info;

public class SMProxy extends Proxy {
    private static final BigInteger scoreBase = new BigInteger(1, Arrays.copyOf(new byte[]{1, 2}, 21));
    private static final BigInteger extBase = new BigInteger(1, Arrays.copyOf(new byte[]{1}, 20));

    private static class Account {
        public Address address;
        public BigInteger balance = BigInteger.ZERO;
        public int nextHash = 0;
        public byte[] objectGraph = new byte[0];
        public byte[] objectGraphHash = Crypto.sha3_256(new byte[0]);
        public Map<ByteArrayWrapper, byte[]> storage = new HashMap<>();

        Account(byte[] addr) {
            address = new Address(addr);
        }
    }

    private FileSystem fs = new FileSystem();
    private BigInteger nextScoreAddr = scoreBase;
    private BigInteger nextExtAddr = extBase;
    private BigInteger value = BigInteger.valueOf(0);
    private BigInteger stepLimit = BigInteger.valueOf(1000000000);
    private Map<ByteArrayWrapper, Account> accounts = new HashMap<>();
    private Account current;
    private Address origin;
    private Map<String, Object> info = new HashMap<>();

    public SMProxy(Connection conn) {
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
        current = getAccount(origin);
    }

    public static byte[] makeJar(Class c) {
        return makeJar(c.getName(), new Class[]{c});
    }

    public static byte[] makeJar(String name, Class[] all) {
        byte[] preopt = JarBuilder.buildJarForExplicitMainAndClasses(name, all);
        return new OptimizedJarBuilder(true, preopt)
                .withUnreachableMethodRemover()
                .withRenamer().getOptimizedBytes();
    }

    public Address newScoreAddress() {
        var addr = new Address(nextScoreAddr.toByteArray());
        nextScoreAddr = nextScoreAddr.add(BigInteger.ONE);
        return addr;
    }

    public Address newExternalAddress() {
        var addr = Arrays.copyOf(new byte[]{0, 1}, 21);
        var next = nextExtAddr.toByteArray();
        System.arraycopy(next, 0, addr, 1, 20);
        nextExtAddr = nextExtAddr.add(BigInteger.ONE);
        return new Address(addr);
    }

    private Account getAccount(Address addr) {
        var ba = addr.toByteArray();
        var baw = new ByteArrayWrapper(ba);
        var account = accounts.get(baw);
        if (account==null) {
            account = new Account(ba);
            accounts.put(baw, account);
        }
        return account;
    }

    public Contract deploy(Class main, Object ... params) {
        byte[] jar = makeJar(main);
        return doDeploy(jar, params);
    }

    public Address getOrigin() {
        return origin;
    }

    public Contract deploy(Class main, Class[] all, Object ... params) {
        byte[] jar = makeJar(main.getName(), all);
        return doDeploy(jar, params);
    }

    private Contract doDeploy(byte[] jar, Object ... params) {
        Address scoreAddr = newScoreAddress();
        String path = scoreAddr.toString() + "/optimized";
        fs.writeFile(path, jar);
        try {
            var prev = current;
            current = getAccount(scoreAddr);
            info.put(Info.CONTRACT_OWNER, origin);
            var res = invoke(path, false, origin, scoreAddr, value, stepLimit, "onInstall", params);
            if (res.getStatus()!=0) {
                throw new IllegalArgumentException("deploy failed");
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

    public FileSystem getFileSystem() {
        return fs;
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
            Message msg = getNextMessageNoLog();
            if (msg.type==type) {
                return msg;
            }
            switch(msg.type) {
                case EEProxy.MsgType.GETVALUE: {
                    var key = msg.value.asRawValue().asByteArray();
                    var value = current.storage.get(new ByteArrayWrapper(key));
                    System.out.format("RECV getValue %s => %s%n", beautify(key), beautify(value));
                    sendMessage(EEProxy.MsgType.GETVALUE, value!=null, value);
                    break;
                }
                case EEProxy.MsgType.SETVALUE: {
                    var data = msg.value.asArrayValue();
                    var key = data.get(0).asRawValue().asByteArray();
                    var isDelete = data.get(1).asBooleanValue().getBoolean();
                    if (isDelete) {
                        current.storage.remove(new ByteArrayWrapper(key));
                        System.out.format("RECV setValue %s isDelete=%b%n", beautify(key), isDelete);
                    } else {
                        var value = data.get(2).asRawValue().asByteArray();
                        current.storage.put(new ByteArrayWrapper(key), value);
                        System.out.format("RECV setValue %s isDelete=%b %s%n", beautify(key), isDelete, beautify(value));
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
                    System.out.format("RECV call to=%s value=%d stepLimit=%d method=%s params=%s%n",
                            to, value, stepLimit, method, beautify(params));
                    var res = invoke(to, value, stepLimit, method, params);
                    sendMessage(EEProxy.MsgType.RESULT, res.getStatus(), res.getStepUsed(), TypedObj.encodeAny(res.getResult()));
                    break;
                }
                case EEProxy.MsgType.EVENT: {
                    var data = msg.value.asArrayValue();
                    var indexed = unpackByteArrayArray(data.get(0).asArrayValue());
                    var nonIndexed = unpackByteArrayArray(data.get(0).asArrayValue());
                    System.out.format("RECV event indxed=%s data=%s%n", beautify(indexed), beautify(nonIndexed));
                    break;
                }
                case EEProxy.MsgType.GETBALANCE: {
                    var addr = new Address(msg.value.asRawValue().asByteArray());
                    var balance = getAccount(addr).balance;
                    sendMessage(EEProxy.MsgType.GETBALANCE, (Object) balance.toByteArray());
                    System.out.format("RECV getBalance %s => %d%n", addr, balance);
                    break;
                }
                case EEProxy.MsgType.LOG: {
                    var data = msg.value.asArrayValue();
                    var level = data.get(0).asIntegerValue().asInt();
                    var logMsg = data.get(1).asStringValue().asString();
                    // filter only Blockchain.println
                    if (logMsg.startsWith("org.aion.avm.core.BlockchainRuntimeImpl PRT|")) {
                        System.out.format("RECV log level=%d %s%n", level, logMsg);
                    }
                    break;
                }
                case EEProxy.MsgType.SETCODE:{
                    var code = msg.value.asRawValue().asByteArray();
                    fs.writeFile(current.address.toString() + "/transformed", code);
                    System.out.format("RECV setCode hash=%s len=%d%n", beautify(Crypto.sha3_256(code)), code.length);
                    break;
                }
                case EEProxy.MsgType.GETOBJGRAPH: {
                    var flag = msg.value.asIntegerValue().asInt();
                    var nextHash = current.nextHash;
                    var ogh = current.objectGraphHash;
                    if ((flag&1)!=0) {
                        var og = current.objectGraph;
                        System.out.format("RECV getObjGraph flag=%d => next=%d hash=%s graphLen=%d graph=%s%n", flag, nextHash, beautify(ogh), og.length, beautifyObjectGraph(og));
                        sendMessage(EEProxy.MsgType.GETOBJGRAPH, nextHash, ogh, og);
                    } else {
                        System.out.format("RECV getObjGraph flag=%d => next=%d hash=%s%n", flag, nextHash, beautify(ogh));
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
                        System.out.format("RECV setObjGraph flag=%d next=%d hash=%s graphLen=%d graph=%s%n", flag, nextHash, beautify(current.objectGraphHash), og.length, beautifyObjectGraph(og));
                    } else {
                        System.out.format("RECV setObjGraph flag=%d next=%d%n", flag, nextHash);
                    }
                    break;
                }
            }
        }
    }

    public Result invoke(Address to, BigInteger value, BigInteger stepLimit,
                         String method, Object[] params) throws IOException {
        var prev = current;
        var from = current.address;
        current = getAccount(to);
        info.put(Info.CONTRACT_OWNER, from);
        var res = invoke(to.toString()+"/transformed", false, from, to, value, stepLimit, method, params);
        current = prev;
        return res;
    }

    public Result invoke(String code, boolean isQuery, Address from,
                     Address to, BigInteger value, BigInteger stepLimit,
                     String method, Object[] params) throws IOException {
        System.out.format("SEND invoke code=%s isQuery=%b from=%s to=%s value=%d stepLimit=%d method=%s params=%s%n",
                code, isQuery, from, to, value, stepLimit, method,
                beautify(params));
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
        System.out.format("RECV result status=%d stepUsed=%d ret=%s%n", status, stepUsed, beautify(result));
        return new Result(status, stepUsed, result);
    }

    private static boolean isPrint(int ch) {
        if ( ch >= 0x20 && ch <= 0x7e )
            return true;
        return false;
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

    private static String beautify(Object o) {
        if (o==null) {
            return "<null>";
        } else if (o instanceof byte[]) {
            return toHex((byte[])o);
        } else if (o instanceof Object[]) {
            return beautifyObjects((Object[])o);
        }
        return o.toString();
    }

    private static String beautifyObjectGraph(byte[] og) {
        StringBuilder sb = new StringBuilder();
        for (int i=0; i<og.length; i++) {
            if (isPrint(og[i])) {
                sb.append((char)og[i]);
            } else {
                int v = og[i] & 0xFF;
                char c1 = HEX_ARRAY[v >>> 4];
                char c2 = HEX_ARRAY[v & 0x0F];
                sb.append("\\x"+c1+c2);
            }
        }
        return sb.toString();
    }
}
