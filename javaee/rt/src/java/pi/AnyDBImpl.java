package pi;

import foundation.icon.ee.util.Crypto;
import foundation.icon.ee.util.ValueCodec;
import i.*;
import org.aion.avm.RuntimeMethodFeeSchedule;
import p.score.AnyDB;
import s.java.lang.Class;
import s.java.lang.String;

public class AnyDBImpl extends s.java.lang.Object implements AnyDB {
    private static final byte TYPE_ARRAY_DB = 0;
    private static final byte TYPE_DICT_DB = 1;
    private static final byte TYPE_VAR_DB = 2;

    private Class<?> leafValue;
    // id[0] is set by type during operation
    // id[0] is set 0 before serialization
    private byte[] id;
    private byte[] hash;

    public AnyDBImpl(String id, Class<?> vc) {
        this(catEncodedKey(new byte[]{(byte) 0}, id), vc);
    }

    private AnyDBImpl(byte[] id, Class<?> vc) {
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
        getDBStorage().setBytes(getStorageKey(TYPE_VAR_DB), encode(value));
    }

    public IObject avm_get() {
        return decode(getDBStorage().getBytes(getStorageKey(TYPE_VAR_DB)));
    }

    public IObject avm_getOrDefault(IObject defaultValue) {
        var out = decode(getDBStorage().getBytes(getStorageKey(TYPE_VAR_DB)));
        return (out != null) ? out : defaultValue;
    }

    // CollectionDB
    public void avm_set(IObject key, IObject value) {
        getDBStorage().setBytes(getItemStorageKey(key), encode(value));
    }

    public IObject avm_at(IObject key) {
        IInstrumentation.attachedThreadInstrumentation.get()
                .chargeEnergy(RuntimeMethodFeeSchedule.DictDB_avm_at);
        return new AnyDBImpl(getSubDBID(key), leafValue);
    }

    public IObject avm_get(IObject key) {
        return decode(getDBStorage().getBytes(getItemStorageKey(key)));
    }

    public IObject avm_getOrDefault(IObject key, IObject defaultValue) {
        var out = decode(getDBStorage().getBytes(getItemStorageKey(key)));
        return (out != null) ? out : defaultValue;
    }

    public void avm_add(IObject value) {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey(TYPE_ARRAY_DB));
        s.setBytes(getItemStorageKey(sz), encode(value));
        s.setArrayLength(getStorageKey(TYPE_ARRAY_DB), sz + 1);
    }

    public void avm_set(int index, IObject value) {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey(TYPE_ARRAY_DB));
        if (index >= sz || index < 0) {
            throw new IllegalArgumentException();
        }
        s.setBytes(getItemStorageKey(index), encode(value));
    }

    public void avm_removeLast() {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey(TYPE_ARRAY_DB));
        if (sz <= 0) {
            throw new IllegalStateException();
        }
        s.setBytes(getItemStorageKey(sz - 1), null);
        s.setArrayLength(getStorageKey(TYPE_ARRAY_DB), sz - 1);
    }

    public IObject avm_pop() {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey(TYPE_ARRAY_DB));
        if (sz <= 0) {
            throw new IllegalStateException();
        }
        var o = decode(s.getBytes(getItemStorageKey(sz - 1)));
        s.setBytes(getItemStorageKey(sz - 1), null);
        s.setArrayLength(getStorageKey(TYPE_ARRAY_DB), sz - 1);
        return o;
    }

    public IObject avm_get(int index) {
        IDBStorage s = getDBStorage();
        int sz = s.getArrayLength(getStorageKey(TYPE_ARRAY_DB));
        if (index >= sz || index < 0) {
            throw new IllegalArgumentException();
        }
        return decode(s.getBytes(getItemStorageKey(index)));
    }

    public int avm_size() {
        return getDBStorage().getArrayLength(getStorageKey(TYPE_ARRAY_DB));
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

    private byte[] getStorageKey(byte type) {
        IInstrumentation.charge(
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_sha3_256_base +
                RuntimeMethodFeeSchedule.BlockchainRuntime_avm_sha3_256_per_bytes * id.length);
        if (hash == null) {
            id[0] = type;
            hash = Crypto.sha3_256(id);
        }
        return hash;
    }

    private byte[] getItemStorageKey(IObject key) {
        id[0] = TYPE_DICT_DB;
        return hashWithCharge(catEncodedKey(id, key));
    }

    private byte[] getItemStorageKey(int key) {
        id[0] = TYPE_ARRAY_DB;
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
        // to make consistent object graph
        this.id[0] = 0;
        CodecIdioms.serializeByteArray(serializer, this.id);
        serializer.writeObject(this.leafValue);
    }
}
