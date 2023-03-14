package foundation.icon.ee.util;

import foundation.icon.ee.struct.Property;
import foundation.icon.ee.types.Address;
import i.PackageConstants;
import i.RuntimeAssertionError;
import org.objectweb.asm.Type;
import pi.UnmodifiableArrayList;
import pi.UnmodifiableArrayMap;

import java.lang.reflect.InvocationTargetException;
import java.math.BigInteger;
import java.util.ArrayList;

public class Unshadower {
    public static Object unshadow(Object so) {
        if (so==null) {
            return null;
        } else if (so instanceof Boolean) {
            return so;
        } else if (so instanceof Character) {
            return BigInteger.valueOf((char)so);
        } else if (so instanceof Byte) {
            return BigInteger.valueOf((byte)so);
        } else if (so instanceof Short) {
            return BigInteger.valueOf((short)so);
        } else if (so instanceof Integer) {
            return BigInteger.valueOf((int)so);
        } else if (so instanceof Long) {
            return BigInteger.valueOf((long)so);
        } else if (so instanceof s.java.lang.Boolean) {
            return ((s.java.lang.Boolean)so).getUnderlying();
        } else if (so instanceof s.java.lang.Character) {
            return BigInteger.valueOf(
                    (int)((s.java.lang.Character)so).getUnderlying());
        } else if (so instanceof s.java.lang.Byte) {
            return BigInteger.valueOf(((s.java.lang.Byte)so).getUnderlying());
        } else if (so instanceof s.java.lang.Short) {
            return BigInteger.valueOf(((s.java.lang.Short)so).getUnderlying());
        } else if (so instanceof s.java.lang.Integer) {
            return BigInteger.valueOf(((s.java.lang.Integer)so).getUnderlying());
        } else if (so instanceof s.java.lang.Long) {
            return BigInteger.valueOf(((s.java.lang.Long)so).getUnderlying());
        } else if (so instanceof s.java.lang.String) {
            var o = (s.java.lang.String) so;
            return o.getUnderlying();
        } else if (so instanceof s.java.math.BigInteger) {
            var o = (s.java.math.BigInteger) so;
            return o.getUnderlying();
        } else if (so instanceof p.score.Address) {
            var o = (p.score.Address) so;
            return new Address(o.toByteArray());
        } else if (so instanceof a.BooleanArray) {
            var o = (a.BooleanArray) so;
            var res = new Object[o.length()];
            for (int i=0; i<o.length(); i++) {
                res[i] = o.get(i);
            }
            return res;
        } else if (so instanceof a.CharArray) {
            var o = (a.CharArray) so;
            var res = new Object[o.length()];
            for (int i=0; i<o.length(); i++) {
                res[i] = BigInteger.valueOf((int)o.get(i));
            }
            return res;
        } else if (so instanceof a.ByteArray) {
            var o = (a.ByteArray) so;
            return o.getUnderlying();
        } else if (so instanceof a.ShortArray) {
            var o = (a.ShortArray) so;
            var res = new Object[o.length()];
            for (int i=0; i<o.length(); i++) {
                res[i] = BigInteger.valueOf(o.get(i));
            }
            return res;
        } else if (so instanceof a.IntArray) {
            var o = (a.IntArray) so;
            var res = new Object[o.length()];
            for (int i=0; i<o.length(); i++) {
                res[i] = BigInteger.valueOf(o.get(i));
            }
            return res;
        } else if (so instanceof a.LongArray) {
            var o = (a.LongArray) so;
            var res = new Object[o.length()];
            for (int i = 0; i < o.length(); i++) {
                res[i] = BigInteger.valueOf(o.get(i));
            }
            return res;
        } else if (so instanceof a.ObjectArray) {
            var o = (a.ObjectArray) so;
            var res = new Object[o.length()];
            for (int i = 0; i< o.length(); i++) {
                res[i] = unshadow(o.get(i));
            }
            return res;
        } else if (so instanceof UnmodifiableArrayList) {
            var o = (UnmodifiableArrayList<?>) so;
            var sa = o.getData();
            var oa = new Object[sa.length];
            for (int i = 0; i < sa.length; i++) {
                oa[i] = Unshadower.unshadow(sa[i]);
            }
            return oa;
        } else if (so instanceof s.java.util.List) {
            var o = (s.java.util.List<?>) so;
            var l = new ArrayList<>();
            var it = o.avm_iterator();
            while (it.avm_hasNext()) {
                l.add(Unshadower.unshadow(it.avm_next()));
            }
            return l.toArray();
        } else if (so instanceof UnmodifiableArrayMap) {
            var o = (UnmodifiableArrayMap<?, ?>) so;
            var skv = o.getData();
            var map = new java.util.LinkedHashMap<>();
            for (int i = 0; i < skv.length; i += 2) {
                var k = skv[i];
                var v = skv[i+1];
                if (!(k instanceof s.java.lang.String)) {
                    throw new IllegalArgumentException("map key is not a string");
                }
                map.put(Unshadower.unshadow(k), Unshadower.unshadow(v));
            }
            return map;
        } else if (so instanceof s.java.util.Map) {
            var o = (s.java.util.Map<?, ?>) so;
            var map = new java.util.LinkedHashMap<>();
            var it = o.avm_entrySet().avm_iterator();
            while (it.avm_hasNext()) {
                var e = it.avm_next();
                var k = e.avm_getKey();
                if (!(k instanceof s.java.lang.String)) {
                    throw new IllegalArgumentException("map key is not a string");
                }
                map.put(
                        Unshadower.unshadow(k),
                        Unshadower.unshadow(e.avm_getValue())
                );
            }
            return map;
        } else {
            var rProps = Property.getReadableProperties(so);
            if (rProps.isEmpty()) {
                throw new IllegalArgumentException();
            }
            var map = new java.util.TreeMap<>();
            for (var rp : rProps) {
                try {
                    map.put(rp.getName(), Unshadower.unshadow(rp.get(so)));
                } catch (InvocationTargetException | IllegalAccessException e) {
                    // IllegalAccessException can be thrown if this object is
                    // not public.
                    throw new IllegalArgumentException(e);
                }
            }
            return map;
        }
    }

    private static final char CLASS_START = 'L';
    private static final char CLASS_END = ';';

    private static String unshadowArrayDescriptor(String sName) {
        if (sName.startsWith("a/$")) {
            var dim = Strings.countPrefixRun(sName.substring(2), '$');
            var elem = sName.substring(2 + dim);
            if (elem.startsWith("L")) {
                elem += ";";
            }
            return "[".repeat(dim) + unshadowDescriptor(elem);
        } else if (sName.startsWith("w/_")) {
            var dim = Strings.countPrefixRun(sName.substring(2), '_');
            var elem = sName.substring(2 + dim);
            if (elem.startsWith("L")) {
                elem += ";";
            }
            return "[".repeat(dim) + unshadowDescriptor(elem);
        }
        switch (sName) {
            case "a/BooleanArray": return "[Z";
            case "a/CharArray": return "[C";
            case "a/ByteArray": return "[B";
            case "a/ShortArray": return "[S";
            case "a/IntArray": return "[I";
            case "a/LongArray": return "[J";
        }
        throw RuntimeAssertionError.unreachable("bad array type");
    }

    private static String unshadowClassName(String sName) {
        if (sName.startsWith(PackageConstants.kShadowSlashPrefix)) {
            return sName.substring(PackageConstants.kShadowSlashPrefix.length());
        } else if (sName.startsWith(PackageConstants.kShadowApiSlashPrefix)) {
            return sName.substring(PackageConstants.kShadowApiSlashPrefix.length());
        } else if (sName.startsWith(PackageConstants.kExceptionWrapperSlashPrefix)) {
            return sName.substring(PackageConstants.kExceptionWrapperSlashPrefix.length());
        } else if (sName.startsWith(PackageConstants.kUserSlashPrefix)) {
            return sName.substring(PackageConstants.kUserSlashPrefix.length());
        }
        return sName;
    }

    public static String unshadowDescriptor(String desc) {
        StringBuilder res = new StringBuilder();
        StringBuilder className = null;
        int i = 0;
        while (i < desc.length()) {
            char c = desc.charAt(i++);
            if (className != null) {
                if (c == CLASS_END) {
                    var name = className.toString();
                    if (name.startsWith(PackageConstants.kArrayWrapperSlashPrefix)
                            || name.startsWith(PackageConstants.kArrayWrapperUnifyingSlashPrefix) ) {
                        res.append(unshadowArrayDescriptor(name));
                    } else {
                        res.append(CLASS_START);
                        res.append(unshadowClassName(className.toString()));
                        res.append(CLASS_END);
                    }
                    className = null;
                } else {
                    className.append(c);
                }
            } else if (c == CLASS_START) {
                className = new StringBuilder();
            } else {
                res.append(c);
            }
        }
        return res.toString();
    }

    public static Type unshadowType(Type type) {
        return Type.getType(unshadowDescriptor(type.getDescriptor()));
    }
}
