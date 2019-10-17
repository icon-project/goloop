package org.aion.avm.core;

import i.*;
import org.aion.avm.StorageFees;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.types.AionAddress;
import p.avm.ValueBuffer;
import s.java.lang.Integer;

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
            v = serializeObject(Integer.valueOf(l));
        }
        ctx.putStorage(getAddress(), key, v);
    }

    public int getArrayLength(byte[] key) {
        var bs = ctx.getStorage(getAddress(), key);
        if (bs==null)
            return 0;
        Integer i = (Integer) deserializeObject(bs);
        return i.getUnderlying();
    }

    private AionAddress getAddress() {
        var addr = IInstrumentation.attachedThreadInstrumentation.get()
                .getFrameContext().getBlockchainRuntime().avm_getAddress();
        return new AionAddress(addr.toByteArray());
    }

    private void charge(long cost) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(cost);
    }

    void assumeValidValue(Object value) {
        if (value instanceof DBImplBase) {
            throw new InvalidDBAccessException();
        }
    }

    public void setValue(byte[] key, IObject value) {
        assumeValidValue(value);
        byte[] v = null;
        if (value!=null) {
            v = serializeObject(value);
            charge(v.length * StorageFees.WRITE_PRICE_PER_BYTE);
        }
        ctx.putStorage(getAddress(), key, v);
    }

    public IObject getValue(byte[] key) {
        var v = ctx.getStorage(getAddress(), key);
        if (v==null)
            return null;
        charge(v.length * StorageFees.READ_PRICE_PER_BYTE);
        return (IObject) deserializeObject(v);
    }

    public ValueBuffer getValue(byte[] key, ValueBuffer out) {
        var v = ctx.getStorage(getAddress(), key);
        if (out==null)
            out = new ValueBuffer();
        out.set(v);
        if (v!=null)
            charge(v.length);
        return out;
    }

    public void flush() {
    }
}
