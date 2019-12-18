package i;

import foundation.icon.ee.utils.Crypto;
import foundation.icon.ee.utils.ValueCodec;

public class DBImplBase extends s.java.lang.Object {
    public static final int TYPE_ARRAY_DB = 0;
    public static final int TYPE_DICT_DB = 1;
    public static final int TYPE_VAR_DB = 2;

    private byte[] id;
    private byte[] hash;
    protected s.java.lang.Class<?> leafValue;

    public DBImplBase(int type, s.java.lang.String id, s.java.lang.Class<?> vc) {
        this(catEncodedKey(new byte[]{(byte) type}, id), vc);
    }

    public DBImplBase(byte[] id, s.java.lang.Class<?> vc) {
        this.id = id;
        this.leafValue = vc;
    }

    public DBImplBase(Void ignore, int readIndex) {
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

    public IDBStorage chargeAndGetDBStorage(int cost) {
        IInstrumentation ins = IInstrumentation.attachedThreadInstrumentation.get();
        ins.chargeEnergy(cost);
        return ins.getFrameContext().getDBStorage();
    }

    public byte[] getStorageKey() {
        if (hash == null) {
            hash = Crypto.sha3_256(id);
        }
        return hash;
    }

    public byte[] getStorageKey(IObject key) {
        return Crypto.sha3_256(catEncodedKey(id, key));
    }

    public byte[] getStorageKey(int key) {
        return Crypto.sha3_256(catEncodedKey(id, key));
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
        super.deserializeSelf(DBImplBase.class, deserializer);
        this.id = CodecIdioms.deserializeByteArray(deserializer);
        this.leafValue = (s.java.lang.Class<?>) deserializer.readObject();
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(DBImplBase.class, serializer);
        CodecIdioms.serializeByteArray(serializer, this.id);
        serializer.writeObject(this.leafValue);
    }
}
