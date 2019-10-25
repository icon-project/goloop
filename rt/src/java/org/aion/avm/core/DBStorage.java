package org.aion.avm.core;

import i.*;
import org.aion.avm.StorageFees;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.types.AionAddress;
import p.avm.Value;
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

    public void setValue(byte[] key, Value value) {
        byte[] v = null;
        if (value!=null) {
            if (value instanceof ValueBuffer) {
                v = ((ValueBuffer)value).asByteArray();
            } else {
                v = value.avm_asByteArray().getUnderlying();
            }
            charge(v.length * StorageFees.WRITE_PRICE_PER_BYTE);
        }
        ctx.putStorage(getAddress(), key, v);
    }

    public Value getValue(byte[] key, ValueBuffer out) {
        var v = ctx.getStorage(getAddress(), key);
        if (v==null)
            return null;
        charge(v.length);
        if (out==null)
            out = new ValueBuffer();
        out.set(v);
        return out;
    }

    public void flush() {
    }
}
