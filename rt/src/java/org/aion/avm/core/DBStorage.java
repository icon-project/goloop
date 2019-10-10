package org.aion.avm.core;

import i.ValueCodec;

public class DBStorage {
    private IExternalState ctx;

    public DBStorage(IExternalState ctx) {
        this.ctx = ctx;
    }

    public void setArrayLength(byte[] key, int l) {
        var v = ValueCodec.encodeValue(Integer.valueOf(l));
        setValue(key, v);
    }

    public int getArrayLength(byte[] key) {
        var bs = getValue(key);
        Integer i = (Integer) ValueCodec.decodeValue(bs);
        return i.intValue();
    }

    public void setValue(byte[] key, byte[] value) {
        ctx.putStorage(null, key, value);
    }

    public byte[] getValue(byte[] key) {
        return ctx.getStorage(null, key);
    }

    public void flush() {
    }
}
