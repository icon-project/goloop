package org.aion.avm.core;

import i.*;
import org.aion.avm.embed.hash.HashUtils;

class DBImplBase extends s.java.lang.Object {
    DBStorage storage;
    byte[] id;
    byte[] hash;

    DBImplBase(DBStorage storage, s.java.lang.String id) {
        this.storage = storage;
        this.id = encodeKey(id);
    }

    DBImplBase(DBStorage storage, byte[] id) {
        this.storage = storage;
        this.id = id;
    }

    void ensureStorage() {
        if (storage==null) {
            var fc = IInstrumentation.attachedThreadInstrumentation.get().getFrameContext();
            storage = ((BlockchainRuntimeImpl)fc.getBlockchainRuntime()).getDBStorage();
        }
    }

    static byte[] encodeKey(Object k) {
        var c = new RLPCoder();
        c.encode(k);
        return c.toByteArray();
    }

    static byte[] catEncodedKey(byte[] prefix, Object k) {
        var c = new RLPCoder();
        c.write(prefix);
        c.encode(k);
        return c.toByteArray();
    }

    static byte[] catEncodedKey(byte[] prefix, int k) {
        var c = new RLPCoder();
        c.write(prefix);
        c.encode(k);
        return c.toByteArray();
    }

    void charge(long c) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(c);
    }

    void assumeValidValue(Object value) {
        if (value instanceof DBImplBase) {
            throw new InvalidDBAccessException();
        }
    }

    byte[] getIDHash() {
        if (hash == null) {
            hash = HashUtils.sha3_256(id);
        }
        return hash;
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
