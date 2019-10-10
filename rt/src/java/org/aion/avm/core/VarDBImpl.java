package org.aion.avm.core;

import i.ValueCodec;
import org.aion.avm.RuntimeMethodFeeSchedule;
import org.aion.avm.StorageFees;
import p.avm.PrimitiveBuffer;
import p.avm.VarDB;
import s.java.lang.String;

public class VarDBImpl<V> extends DBImplBase implements VarDB<V> {
    public VarDBImpl(DBStorage storage, String id) {
        super(storage, id);
    }

    // type of value shall not be DictDB, ArrayDB or VarDB.
    public void avm_putValue(V value) {
        int w = 0;
        try {
            ensureStorage();
            assumeValidValue(value);
            var v = ValueCodec.encodeValue(value);
            w = v.length;
            storage.setValue(getIDHash(), v);
        } finally {
            charge(RuntimeMethodFeeSchedule.VarDB_avm_putValue + StorageFees.WRITE_PRICE_PER_BYTE * w);
        }
    }

    public V avm_getValue() {
        int r = 0;
        try {
            ensureStorage();
            var v = storage.getValue(getIDHash());
            r = v.length;
            return (V) ValueCodec.decodeValue(v);

        } finally {
            charge(RuntimeMethodFeeSchedule.VarDB_avm_getValue + StorageFees.READ_PRICE_PER_BYTE * r);
        }
    }

    public PrimitiveBuffer avm_getValue(PrimitiveBuffer out) {
        int r = 0;
        try {
            ensureStorage();
            var v = storage.getValue(getIDHash());
            r = v.length;
            out.set(v);
            return out;
        } finally {
            charge(RuntimeMethodFeeSchedule.VarDB_avm_getValue
                    + StorageFees.READ_PRICE_PER_BYTE * r);
        }
    }
}
