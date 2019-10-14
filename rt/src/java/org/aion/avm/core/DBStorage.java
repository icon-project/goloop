package org.aion.avm.core;

import i.*;
import org.aion.avm.StorageFees;
import org.aion.types.AionAddress;
import p.avm.PrimitiveBuffer;
import s.java.lang.Integer;

public class DBStorage implements IDBStorage {
    private IExternalState ctx;

    public DBStorage(IExternalState ctx) {
        this.ctx = ctx;
    }

    public void setArrayLength(byte[] key, int l) {
        byte[] v;
        if (l==0) {
            v = null;
        } else {
            v = ValueCodec.encodeValue(Integer.valueOf(l));
        }
        ctx.putStorage(getAddress(), key, v);
    }

    public int getArrayLength(byte[] key) {
        var bs = ctx.getStorage(getAddress(), key);
        if (bs==null)
            return 0;
        Integer i = (Integer) ValueCodec.decodeValue(bs);
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
            v = ValueCodec.encodeValue(value);
            charge(v.length * StorageFees.WRITE_PRICE_PER_BYTE);
        }
        ctx.putStorage(getAddress(), key, v);
    }

    public IObject getValue(byte[] key) {
        var v = ctx.getStorage(getAddress(), key);
        if (v==null)
            return null;
        charge(v.length * StorageFees.READ_PRICE_PER_BYTE);
        return (IObject) ValueCodec.decodeValue(v);
    }

    public PrimitiveBuffer getValue(byte[] key, PrimitiveBuffer out) {
        var v = ctx.getStorage(getAddress(), key);
        if (out==null)
            out = new PrimitiveBuffer();
        out.set(v);
        if (v!=null)
            charge(v.length);
        return out;
    }

    public void flush() {
    }
}
