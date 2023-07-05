/*
 * Copyright 2019 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package org.aion.avm.core;

import i.IDBStorage;
import i.IInstrumentation;

import java.math.BigInteger;

public class DBStorage implements IDBStorage {
    private final IExternalState ctx;

    public DBStorage(IExternalState ctx) {
        this.ctx = ctx;
    }

    public void setArrayLength(byte[] key, int l) {
        byte[] v;
        if (l == 0) {
            v = null;
        } else {
            v = BigInteger.valueOf(l).toByteArray();
        }
        setBytes(key, v);
    }

    public int getArrayLength(byte[] key) {
        var bs = getBytes(key);
        if (bs == null)
            return 0;
        return new BigInteger(bs).intValue();
    }

    private void charge(long cost) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(cost);
    }

    private void chargeImmediately(long cost) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergyImmediately(cost);
    }

    private boolean tryCharge(long cost) {
        return IInstrumentation.attachedThreadInstrumentation.get().tryChargeEnergy(cost);
    }

    public void setBytes(byte[] k, byte[] v) {
        if (ctx.isReadOnly()) {
            throw new IllegalStateException();
        }
        var stepCost = ctx.getStepCost();

        // Limit proxy connection's pending read buffer by consuming not to
        // block sender (=proxy in sm). Do not wait unless we have too many
        // pending items.
        this.ctx.limitPendingCallbackLength();

        if (v == null) {
            var rb = stepCost.replaceBase();

            if (tryCharge(rb)) {
                ctx.putStorage(k, null, prevSize -> {
                    if (prevSize > 0) {
                        chargeImmediately(stepCost.setStorageDelete(prevSize) - rb);
                    }
                });
            } else {
                var prev = ctx.getStorage(k);
                if (prev != null) {
                    chargeImmediately(stepCost.setStorageDelete(prev.length));
                } else {
                    chargeImmediately(rb);
                }
                ctx.putStorage(k, null, null);
            }
        } else {
            if (tryCharge(stepCost.setStorageSet(v.length))) {
                ctx.putStorage(k, v, prevSize -> {
                    if (prevSize > 0) {
                        chargeImmediately(-stepCost.setBase() + stepCost.replaceBase()
                                + prevSize * stepCost.delete());
                    }
                });
            } else {
                var prev = ctx.getStorage(k);
                if (prev != null) {
                    chargeImmediately(stepCost.setStorageReplace(prev.length, v.length));
                } else {
                    chargeImmediately(stepCost.setStorageSet(v.length));
                }
                ctx.putStorage(k, v, null);
            }
        }
    }

    public byte[] getBytes(byte[] key) {
        var value = ctx.getStorage(key);
        var stepCost = ctx.getStepCost();
        int len = 0;
        if (value != null)
            len = value.length;
        charge(stepCost.getStorage(len));
        return value;
    }

    public void flush() {
    }
}
