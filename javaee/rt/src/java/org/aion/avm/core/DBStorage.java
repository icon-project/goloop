package org.aion.avm.core;

import i.IDBStorage;
import i.IInstrumentation;
import org.aion.avm.StorageFees;

import java.math.BigInteger;

public class DBStorage implements IDBStorage {
    private IExternalState ctx;

    public DBStorage(IExternalState ctx) {
        this.ctx = ctx;
    }

    public void setArrayLength(byte[] key, int l) {
        if (ctx.isQuery()) {
            throw new IllegalStateException();
        }
        byte[] v;
        if (l==0) {
            v = null;
        } else {
            v = BigInteger.valueOf(l).toByteArray();
        }
        ctx.putStorage(key, v);
    }

    public int getArrayLength(byte[] key) {
        var bs = ctx.getStorage(key);
        if (bs==null)
            return 0;
        return new BigInteger(bs).intValue();
    }

    private void charge(int cost) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(cost);
    }

    public void setBytes(byte[] key, byte[] value) {
        if (ctx.isQuery()) {
            throw new IllegalStateException();
        }
        if (value != null)
            charge(value.length * StorageFees.WRITE_PRICE_PER_BYTE);
        ctx.putStorage(key, value);
    }

    public byte[] getBytes(byte[] key) {
        var value = ctx.getStorage(key);
        if (value != null)
            charge(value.length * StorageFees.READ_PRICE_PER_BYTE);
        return value;
    }

    public void flush() {
    }
}
