package i;

import foundation.icon.ee.utils.Crypto;

public class DBImplBase extends s.java.lang.Object {
    public static final int TYPE_ARRAY_DB = 0;
    public static final int TYPE_DICT_DB = 1;
    public static final int TYPE_VAR_DB = 2;

    byte[] id;
    byte[] hash;

    public DBImplBase(int type, s.java.lang.String id) {
        this.id = catEncodedKey(new byte[]{(byte)type}, id);
    }

    public DBImplBase(byte[] id) {
        this.id = id;
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

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(DBImplBase.class, deserializer);
        this.id = CodecIdioms.deserializeByteArray(deserializer);
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(DBImplBase.class, serializer);
        CodecIdioms.serializeByteArray(serializer, this.id);
    }
}
