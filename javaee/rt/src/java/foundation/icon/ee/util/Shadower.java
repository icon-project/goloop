package foundation.icon.ee.util;

import foundation.icon.ee.types.Address;
import i.IObject;
import i.RuntimeAssertionError;
import pi.UnmodifiableArrayList;
import pi.UnmodifiableArrayMap;

import java.math.BigInteger;
import java.util.Map;

public class Shadower {
    public static s.java.lang.Object shadow(Object obj) {
        if (obj==null) {
            return null;
        } else if (obj instanceof Boolean) {
            return s.java.lang.Boolean.avm_valueOf((Boolean) obj);
        } else if (obj instanceof BigInteger) {
            return s.java.math.BigInteger.newWithCharge((BigInteger)obj);
        } else if (obj instanceof String) {
            return s.java.lang.String.newWithCharge((String)obj);
        } else if (obj instanceof byte[]) {
            return a.ByteArray.newWithCharge((byte[])obj);
        } else if (obj instanceof Address) {
            return p.score.Address.newWithCharge(((Address)obj).toByteArray());
        } else if (obj instanceof Object[]) {
            var o = (Object[]) obj;
            var sa = new IObject[o.length];
            int i = 0;
            for (var e : o) {
                sa[i++] = Shadower.shadow(e);
            }
            return new UnmodifiableArrayList<>(sa);
        } else if (obj instanceof Map) {
            var o = (Map<?, ?>) obj;
            var skv = new IObject[o.size() * 2];
            int i = 0;
            for (Map.Entry<?, ?> e : o.entrySet()) {
                skv[i++] = shadow(e.getKey());
                skv[i++] = shadow(e.getValue());
            }
            return new UnmodifiableArrayMap<>(skv);
        } else {
            RuntimeAssertionError.unreachable("invalid shadow type");
            return null;
        }
    }
}
