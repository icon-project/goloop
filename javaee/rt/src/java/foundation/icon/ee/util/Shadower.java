package foundation.icon.ee.util;

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
            return s.java.lang.Boolean.valueOf((Boolean) obj);
        } else if (obj instanceof BigInteger) {
            return new s.java.math.BigInteger((BigInteger)obj);
        } else if (obj instanceof String) {
            return new s.java.lang.String((String)obj);
        } else if (obj instanceof byte[]) {
            return new a.ByteArray((byte[])obj);
        } else if (obj instanceof avm.Address) {
            return new p.avm.Address(((avm.Address)obj).toByteArray());
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
