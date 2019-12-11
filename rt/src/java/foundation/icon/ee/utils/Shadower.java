package foundation.icon.ee.utils;

import java.math.BigInteger;

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
        } else {
            return null;
        }
    }
}
