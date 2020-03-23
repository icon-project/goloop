package org.aion.avm.core;

import i.IDBStorage;
import i.IInstrumentation;
import i.InternedClasses;
import org.aion.avm.StorageFees;
import org.aion.avm.core.persistence.LoadedDApp;
import p.score.ValueBuffer;

public class DBStorage implements IDBStorage {
    private IExternalState ctx;
    private LoadedDApp dapp;
    private InternedClasses icm;

    public DBStorage(IExternalState ctx, LoadedDApp dapp, InternedClasses icm) {
        this.ctx = ctx;
        this.dapp = dapp;
        this.icm = icm;
    }

    public void setArrayLength(byte[] key, int l) {
        if (ctx.isQuery()) {
            throw new IllegalStateException();
        }
        byte[] v;
        if (l==0) {
            v = null;
        } else {
            var vb = new ValueBuffer();
            vb.set(l);
            v = vb.asByteArray();
        }
        ctx.putStorage(key, v);
    }

    public int getArrayLength(byte[] key) {
        var bs = ctx.getStorage(key);
        if (bs==null)
            return 0;
        var vb = new ValueBuffer();
        vb.set(bs);
        return vb.asInt();
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
