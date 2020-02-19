package foundation.icon.ee.util;

import foundation.icon.ee.types.Address;

public class Unshadower {
    public static Object unshadow(s.java.lang.Object so) {
        if (so==null) {
            return null;
        } else if (so instanceof s.java.lang.Boolean) {
            var o = (s.java.lang.Boolean) so;
            return o.getUnderlying();
        } else if (so instanceof s.java.lang.Character) {
            var o = (s.java.lang.Character) so;
            return o.getUnderlying();
        } else if (so instanceof s.java.lang.Byte) {
            var o = (s.java.lang.Byte) so;
            return o.getUnderlying();
        } else if (so instanceof s.java.lang.Short) {
            var o = (s.java.lang.Short) so;
            return o.getUnderlying();
        } else if (so instanceof s.java.lang.Integer) {
            var o = (s.java.lang.Integer) so;
            return o.getUnderlying();
        } else if (so instanceof s.java.lang.Long) {
            var o = (s.java.lang.Long) so;
            return o.getUnderlying();
        } else if (so instanceof s.java.lang.String) {
            var o = (s.java.lang.String) so;
            return o.getUnderlying();
        } else if (so instanceof s.java.math.BigInteger) {
            var o = (s.java.math.BigInteger) so;
            return o.getUnderlying();
        } else if (so instanceof a.ByteArray) {
            var o = (a.ByteArray) so;
            return o.getUnderlying();
        } else if (so instanceof p.score.Address) {
            var o = (p.score.Address) so;
            return new Address(o.toByteArray());
        } else if (so instanceof s.java.util.List) {
            var o = (s.java.util.List) so;
            var sa = o.getData();
            var oa = new Object[sa.length];
            for (int i = 0; i < sa.length; i++) {
                oa[i] = Unshadower.unshadow((s.java.lang.Object) sa[i]);
            }
            return oa;
        } else if (so instanceof s.java.util.Map) {
            var o = (s.java.util.Map) so;
            var skv = o.getData();
            var map = new java.util.HashMap();
            for (int i = 0; i < skv.length; i += 2) {
                map.put(
                        Unshadower.unshadow((s.java.lang.Object) skv[i]),
                        Unshadower.unshadow((s.java.lang.Object) skv[i + 1])
                );
            }
            return map;
        } else {
            throw new IllegalArgumentException();
        }
    }
}
