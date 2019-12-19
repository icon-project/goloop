package foundation.icon.ee.utils;

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
        } else if (so instanceof p.avm.Address) {
            var o = (p.avm.Address) so;
            return new Address(o.toByteArray());
        } else {
            return null;
        }
    }
}
