package org.aion.avm.core;

import i.IDBStorage;
import i.IInstrumentation;

import java.math.BigInteger;

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
            v = BigInteger.valueOf(l).toByteArray();
        }
        setBytes(key, v);
    }

    public int getArrayLength(byte[] key) {
        var bs = getBytes(key);
        if (bs==null)
            return 0;
        return new BigInteger(bs).intValue();
    }

    private void charge(int cost) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(cost);
    }

    private void chargeImmediately(int cost) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergyImmediately(cost);
    }

    public void setBytes(byte[] k, byte[] v) {
        if (ctx.isQuery()) {
            throw new IllegalStateException();
        }
        var stepCost = ctx.getStepCost();
        if (v==null) {
            charge(stepCost.replaceBase());
            ctx.putStorage(k, v, prevSize -> {
                if (prevSize>=0) {
                    chargeImmediately(stepCost.defaultDelete());
                }
            });
        } else {
            var e = Math.max(stepCost.replaceBase(),
                    v.length) * stepCost.replace();
            charge(e + stepCost.defaultSet());
            ctx.putStorage(k, v, prevSize -> {
                if (prevSize<0) {
                    chargeImmediately(-stepCost.defaultSet());
                }
            });
        }
    }

    public byte[] getBytes(byte[] key) {
        var value = ctx.getStorage(key);
        var stepCost = ctx.getStepCost();
        int len = 0;
        if (value != null)
            len = value.length;
        charge(stepCost.defaultGet() + len * stepCost.get());
        return value;
    }

    public void flush() {
    }
}
