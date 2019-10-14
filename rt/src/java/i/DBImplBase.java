package i;

import org.aion.avm.embed.hash.HashUtils;

import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;

public class DBImplBase extends s.java.lang.Object {
    byte[] id;
    byte[] hash;

    public DBImplBase(s.java.lang.String id) {
        this.id = encodeKey(id);
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

    private static byte[] hash(byte[] msg) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA3-256");
            return digest.digest(msg);
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException(e);
        }
    }

    public IDBStorage getDBStorage(long cost) {
        IInstrumentation ins = IInstrumentation.attachedThreadInstrumentation.get();
        ins.chargeEnergy(cost);
        return ins.getFrameContext().getBlockchainRuntime().getDBStorage();
    }

    public byte[] getStorageKey() {
        if (hash == null) {
            hash = hash(id);
        }
        return hash;
    }

    public byte[] getStorageKey(IObject key) {
        return hash(catEncodedKey(id, key));
    }

    public byte[] getStorageKey(int key) {
        return hash(catEncodedKey(id, key));
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
