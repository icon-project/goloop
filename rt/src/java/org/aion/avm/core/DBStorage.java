package org.aion.avm.core;

import i.DBImplBase;
import i.IDBStorage;
import i.IInstrumentation;
import i.IObject;
import i.InternedClasses;
import i.InvalidDBAccessException;
import org.aion.avm.StorageFees;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.types.AionAddress;
import p.avm.ValueBuffer;

public class DBStorage implements IDBStorage {
    private IExternalState ctx;
    private LoadedDApp dapp;
    private InternedClasses icm;

    public DBStorage(IExternalState ctx, LoadedDApp dapp, InternedClasses icm) {
        this.ctx = ctx;
        this.dapp = dapp;
        this.icm = icm;
    }

    private Object deserializeObject(byte[] rawGraphData) {
        return dapp.deserializeObject(icm, rawGraphData);
    }

    private byte[] serializeObject(Object v) {
        return dapp.serializeObject(v);
    }

    public void setArrayLength(byte[] key, int l) {
        byte[] v;
        if (l==0) {
            v = null;
        } else {
            var vb = new ValueBuffer();
            vb.set(l);
            v = vb.asByteArray();
        }
        ctx.putStorage(getAddress(), key, v);
    }

    public int getArrayLength(byte[] key) {
        var bs = ctx.getStorage(getAddress(), key);
        if (bs==null)
            return 0;
        var vb = new ValueBuffer();
        vb.set(bs);
        return vb.asInt();
    }

    private AionAddress getAddress() {
        var addr = IInstrumentation.attachedThreadInstrumentation.get()
                .getFrameContext().getBlockchainRuntime().avm_getAddress();
        return new AionAddress(addr.toByteArray());
    }

    private void charge(int cost) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(cost);
    }

    private void assumeValidValue(Object value) {
        if (value instanceof DBImplBase) {
            throw new InvalidDBAccessException();
        }
    }

    public void setTyped(byte[] key, IObject value) {
        assumeValidValue(value);
        byte[] v = null;
        if (value!=null) {
            v = serializeObject(value);
            charge(v.length * StorageFees.WRITE_PRICE_PER_BYTE);
        }
        ctx.putStorage(getAddress(), key, v);
    }

    public IObject getTyped(byte[] key) {
        var v = ctx.getStorage(getAddress(), key);
        if (v==null)
            return null;
        charge(v.length * StorageFees.READ_PRICE_PER_BYTE);
        return (IObject) deserializeObject(v);
    }

    public void setBytes(byte[] key, byte[] value) {
        if (value != null)
            charge(value.length * StorageFees.WRITE_PRICE_PER_BYTE);
        ctx.putStorage(getAddress(), key, value);
    }

    public byte[] getBytes(byte[] key) {
        var value = ctx.getStorage(getAddress(), key);
        if (value != null)
            charge(value.length * StorageFees.READ_PRICE_PER_BYTE);
        return value;
    }

    public void flush() {
    }
}
