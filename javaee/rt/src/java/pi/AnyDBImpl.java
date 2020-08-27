package pi;

import foundation.icon.ee.util.Crypto;
import foundation.icon.ee.util.ValueCodec;
import i.*;
import org.aion.avm.RuntimeMethodFeeSchedule;
import p.score.AnyDB;
import s.java.lang.Class;
import s.java.lang.String;

public class AnyDBImpl extends s.java.lang.Object implements AnyDB {
    public static final int TYPE_ARRAY_DB = 0;
    public static final int TYPE_DICT_DB = 1;
    public static final int TYPE_VAR_DB = 2;
    protected Class<?> leafValue;
    private byte[] id;
    private byte[] hash;

    public AnyDBImpl(int type, String id, Class<?> vc) {
        this(catEncodedKey(new byte[]{(byte) type}, id), vc);
    }

    public AnyDBImpl(byte[] id, Class<?> vc) {
        this.id = id;
        this.leafValue = vc;
    }

    public AnyDBImpl(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    private static byte[] encodeKey(Object k) {
        var c = new RLPCoder();
        c.encode(k);
        return c.toByteArray();
    }

    private static byte[] catEncodedKey(byte[] prefix, Object k) {
        var c = new RLPCoder();
        c.write(prefix);
        c.encode(k);
        return c.toByteArray();
    }

    private static byte[] catEncodedKey(byte[] prefix, int k) {
        var c = new RLPCoder();
        c.write(prefix);
        c.encode(k);
        return c.toByteArray();
    }

    // VarDB
    public void avm_set(IObject value) {
        getDBStorage().setBytes(getStorageKey(), encode(value));
    }

    public IObject avm_get() {
        return decode(getDBStorage().getBytes(getStorageKey()));
    }

    public IObject avm_getOrDefault(IObject defaultValue) {
        var out = decode(getDBStorage().getBytes(getStorageKey()));
        return (out != null) ? out : defaultValue;
    }

    // CollectionDB
    public void avm_set(IObject key, IObject value) {
        getDBStorage().setBytes(getStorageKey(key), encode(value));
    }

    public IObject avm_at(IObject key) {
        IInstrumentation.attachedThreadInstrumentation.get()
                .chargeEnergy(RuntimeMethodFeeSchedule.DictDB_avm_at);
        return new AnyDBImpl(getSubDBID(key), leafValue);
    }

    public IObject avm_get(IObject key) {
        return decode(getDBStorage().getBytes(getStorageKey(key)));
    }

    public IObject avm_getOrDefault(IObject key, IObject defaultValue) {
        var out = decode(getDBStorage().getBytes(getStorageKey(key)));
        return (out != null) ? out : defaultValue;
    }

    public void avm_add(IObject value) {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey());
        s.setBytes(getStorageKey(sz), encode(value));
        s.setArrayLength(getStorageKey(), sz + 1);
    }

    public void avm_set(int index, IObject value) {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey());
        if (index >= sz || index < 0) {
            throw new IllegalArgumentException();
        }
        s.setBytes(getStorageKey(index), encode(value));
    }

    public void avm_removeLast() {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey());
        if (sz <= 0) {
            throw new IllegalStateException();
        }
        s.setBytes(getStorageKey(sz - 1), null);
        s.setArrayLength(getStorageKey(), sz - 1);
    }

    public IObject avm_pop() {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey());
        if (sz <= 0) {
            throw new IllegalStateException();
        }
        var o = decode(s.getBytes(getStorageKey(sz - 1)));
        s.setBytes(getStorageKey(sz - 1), null);
        s.setArrayLength(getStorageKey(), sz - 1);
        return o;
    }

    public IObject avm_get(int index) {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey());
        if (index >= sz || index < 0) {
            throw new IllegalArgumentException();
        }
        return decode(s.getBytes(getStorageKey(index)));
    }

    public int avm_size() {
        return getDBStorage().getArrayLength(getStorageKey());
    }

    public IDBStorage getDBStorage() {
        return IInstrumentation.getCurrentFrameContext().getDBStorage();
    }

    public IDBStorage chargeAndGetDBStorage(int cost) {
        IInstrumentation ins = IInstrumentation.attachedThreadInstrumentation.get();
        ins.chargeEnergy(cost);
        return ins.getFrameContext().getDBStorage();
    }

    private byte[] hashWithCharge(byte[] data) {
        IInstrumentation.charge(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_sha3_256_base +
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_sha3_256_per_bytes * (data != null ? data.length : 0));
        return Crypto.sha3_256(data);
    }

    public byte[] getStorageKey() {
        IInstrumentation.charge(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_sha3_256_base +
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_sha3_256_per_bytes * id.length);
        if (hash == null) {
            hash = Crypto.sha3_256(id);
        }
        return hash;
    }

    public byte[] getStorageKey(IObject key) {
        return hashWithCharge(catEncodedKey(id, key));
    }

    public byte[] getStorageKey(int key) {
        return hashWithCharge(catEncodedKey(id, key));
    }

    public byte[] getSubDBID(IObject key) {
        return catEncodedKey(id, key);
    }

    public byte[] encode(IObject obj) {
        return ValueCodec.encode(obj);
    }

    public IObject decode(byte[] raw) {
        return ValueCodec.decode(raw, leafValue);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(AnyDBImpl.class, deserializer);
        this.id = CodecIdioms.deserializeByteArray(deserializer);
        this.leafValue = (Class<?>) deserializer.readObject();
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(AnyDBImpl.class, serializer);
        CodecIdioms.serializeByteArray(serializer, this.id);
        serializer.writeObject(this.leafValue);
    }
}
