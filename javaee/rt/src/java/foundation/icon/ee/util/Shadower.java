package foundation.icon.ee.util;

import foundation.icon.ee.struct.Property;
import foundation.icon.ee.types.Address;
import i.IObject;
import i.IObjectArray;
import i.RuntimeAssertionError;
import pi.UnmodifiableArrayList;
import pi.UnmodifiableArrayMap;

import java.lang.reflect.InvocationTargetException;
import java.math.BigInteger;
import java.util.Map;

public class Shadower {
    /**
     * Shadows internal objects according to a shadow class.
     *
     * @param obj internal object. Map, array, byte[], Address, String, Boolean,
     *            BigInteger.
     * @param c   shadow class
     * @param <T> shadow class
     * @return shadow object
     * @throws IllegalArgumentException thrown if obj and c does not match
     *                                  correctly or obj is not a valid parameter/return object or c is
     *                                  not a valid parameter/return class.
     * @throws ArithmeticException if integer value is out of range.
     */
    public static <T> T shadow(Object obj, Class<T> c) {
        @SuppressWarnings("unchecked")
        T res = (T) shadowImpl(obj, c);
        return res;
    }

    /**
     * Shadows internal objects according to shadow classes.
     * @param objs internal objects.
     * @param c shadow classes.
     * @return shadow objects
     * @throws IllegalArgumentException thrown if obj and c does not match
     *                                  correctly or obj is not a valid parameter/return object or c is
     *                                  not a valid parameter/return class.
     * @throws ArithmeticException if integer value is out of range.
     */
    public static Object[] shadowObjects(Object[] objs, Class<?>[] c) {
        if (objs.length != c.length) {
            throw new IllegalArgumentException();
        }
        var res = new Object[objs.length];
        for (int i = 0; i < objs.length; ++i) {
            res[i] = shadowImpl(objs[i], c[i]);
        }
        return res;
    }

    /**
     * Shadows internal objects according to shadow class for return value.
     * @param obj internal object.
     * @param c shadow class.
     * @return shadow object.
     * @throws IllegalArgumentException thrown if obj and c does not match
     *                                  correctly or obj is not a valid parameter/return object or c is
     *                                  not a valid parameter/return class.
     * @throws ArithmeticException if integer value is out of range.
     */
    public static <T, U extends IObject> U shadowReturnValue(Object obj, Class<T> c) {
        if (obj == null) {
            return null;
        } else if (c.isPrimitive()) {
            @SuppressWarnings("unchecked")
            U res = (U) shadowPrimitiveReturnValue(obj, c);
            return res;
        }
        @SuppressWarnings("unchecked")
        U res = (U) shadowImpl(obj, c);
        return res;
    }

    private static i.IObjectArray newObjectArray(Class<?> c, int l) throws ClassNotFoundException, NoSuchMethodException, InvocationTargetException, IllegalAccessException {
        var post = c.getName().substring(2);
        int i = 0;
        for (; i < post.length(); ++i) {
            if (post.charAt(i) != '_') {
                break;
            }
        }
        post = post.substring(i); // skip _
        var aName = "a." + "$".repeat(i) + post;
        Class<?> aClass = null;
        aClass = c.getClassLoader().loadClass(aName);
        var m = aClass.getMethod("initArray", int.class);
        return (i.IObjectArray) m.invoke(null, l);
    }

    // called only for ObjectArray
    private static Class<?> getElementClass(Class<?> cls) throws ClassNotFoundException {
        // a.$$I : int[][]
        // w.__Lu.user.Class : interface for u.user.Class[][]
        // a.$$Lu.user.Class : class for u.user.Class[][]
        var elem = cls.getName().substring(3);
        if (elem.startsWith("$")) {
            switch (elem) {
                case "$Z":
                    return a.BooleanArray.class;
                case "$C":
                    return a.CharArray.class;
                case "$B":
                    return a.ByteArray.class;
                case "$S":
                    return a.ShortArray.class;
                case "$I":
                    return a.IntArray.class;
                case "$J":
                    return a.LongArray.class;
            }
            elem = "a.$" + elem.substring(1);
        } else if (elem.startsWith("_")) {
            elem = "w._" + elem.substring(1);
        } else {
            // starts with 'L'
            elem = elem.substring(1);
        }
        return cls.getClassLoader().loadClass(elem);
    }

    private static Object shadowImpl(Object obj, Class<?> c) {
        try {
            return _shadow(obj, c);
        } catch (ClassCastException e) {
            throw new IllegalArgumentException(e);
        }
    }

    private static IObject shadowPrimitiveReturnValue(Object obj, Class<?> c) {
        if (c == boolean.class) {
            return s.java.lang.Boolean.avm_valueOf((Boolean) obj);
        } else if(c == char.class) {
            requireValidCharRange((BigInteger) obj);
            return s.java.lang.Character.avm_valueOf(
                    (char)((BigInteger)obj).intValue());
        } else if(c == byte.class) {
            return s.java.lang.Byte.avm_valueOf(((BigInteger)obj).byteValueExact());
        } else if(c == short.class) {
            return s.java.lang.Short.avm_valueOf(((BigInteger)obj).shortValueExact());
        } else if(c == int.class) {
            return s.java.lang.Integer.avm_valueOf(((BigInteger)obj).intValueExact());
        } else if(c == long.class) {
            return s.java.lang.Long.avm_valueOf(((BigInteger)obj).longValueExact());
        }
        return null;
    }

    private static void requireValidCharRange(BigInteger v) {
        if (v.compareTo(BigInteger.valueOf(Character.MAX_VALUE))>0 ||
                v.compareTo(BigInteger.valueOf(Character.MIN_VALUE))<0) {
            throw new ArithmeticException("out of char range");
        }
    }

    private static Object _shadow(Object obj, Class<?> c) {
        if (obj == null) {
            return null;
        } else if (c == boolean.class) {
            return obj;
        } else if (c == char.class) {
            requireValidCharRange((BigInteger) obj);
            return (char) ((BigInteger) obj).intValue();
        } else if (c == byte.class) {
            return ((BigInteger) obj).byteValueExact();
        } else if (c == short.class) {
            return ((BigInteger) obj).shortValueExact();
        } else if (c == int.class) {
            return ((BigInteger) obj).intValueExact();
        } else if (c == long.class) {
            return ((BigInteger) obj).longValueExact();
        } else if (c == s.java.lang.Boolean.class) {
            return s.java.lang.Boolean.avm_valueOf((Boolean) obj);
        } else if (c == s.java.lang.Character.class) {
            requireValidCharRange((BigInteger) obj);
            return s.java.lang.Character.avm_valueOf(
                    (char)((BigInteger)obj).intValue());
        } else if (c == s.java.lang.Byte.class) {
            return s.java.lang.Byte.avm_valueOf(((BigInteger)obj).byteValueExact());
        } else if (c == s.java.lang.Short.class) {
            return s.java.lang.Short.avm_valueOf(((BigInteger)obj).shortValueExact());
        } else if (c == s.java.lang.Integer.class) {
            return s.java.lang.Integer.avm_valueOf(((BigInteger)obj).intValueExact());
        } else if (c == s.java.lang.Long.class) {
            return s.java.lang.Long.avm_valueOf(((BigInteger)obj).longValueExact());
        } else if (c == s.java.math.BigInteger.class) {
            return s.java.math.BigInteger.newWithCharge((BigInteger)obj);
        } else if (c == s.java.lang.String.class) {
            return s.java.lang.String.newWithCharge((String)obj);
        } else if (c == p.score.Address.class) {
            return p.score.Address.newWithCharge(((Address)obj).toByteArray());
        } else if (c == a.BooleanArray.class) {
            var o = (Object[]) obj;
            var res = a.BooleanArray.initArray(o.length);
            for (int i=0; i<o.length; ++i) {
                res.set(i, (Boolean)o[i]);
            }
            return res;
        } else if (c == a.CharArray.class) {
            var o = (Object[]) obj;
            var res = a.CharArray.initArray(o.length);
            for (int i=0; i<o.length; ++i) {
                res.set(i, (char)((BigInteger)o[i]).intValue());
            }
            return res;
        } else if (c == a.ByteArray.class) {
            return a.ByteArray.newWithCharge((byte[])obj);
        } else if (c == a.ShortArray.class) {
            var o = (Object[]) obj;
            var res = a.ShortArray.initArray(o.length);
            for (int i=0; i<o.length; ++i) {
                res.set(i, ((BigInteger)o[i]).shortValue());
            }
            return res;
        } else if (c == a.IntArray.class) {
            var o = (Object[]) obj;
            var res = a.IntArray.initArray(o.length);
            for (int i=0; i<o.length; ++i) {
                res.set(i, ((BigInteger)o[i]).intValue());
            }
            return res;
        } else if (c == a.LongArray.class) {
            var o = (Object[]) obj;
            var res = a.LongArray.initArray(o.length);
            for (int i=0; i<o.length; ++i) {
                res.set(i, ((BigInteger)o[i]).longValue());
            }
            return res;
        } else if (a.ObjectArray.class.isAssignableFrom(c)) {
            try {
                var o = (Object[]) obj;
                var m = c.getMethod("initArray", int.class);
                a.ObjectArray res = (a.ObjectArray) m.invoke(null, o.length);
                var elemClass = getElementClass(c);
                for (int i = 0; i < o.length; ++i) {
                    res.set(i, _shadow(o[i], elemClass));
                }
                return res;
            } catch (NoSuchMethodException
                    | IllegalAccessException
                    | InvocationTargetException
                    | ClassNotFoundException e) {
                throw new IllegalArgumentException(e);
            }
        } else if (i.IObjectArray.class.isAssignableFrom(c)) {
            var o = (Object[]) obj;
            IObjectArray res = null;
            try {
                res = newObjectArray(c, o.length);
            } catch (ClassNotFoundException | NoSuchMethodException | InvocationTargetException | IllegalAccessException e) {
                throw new IllegalArgumentException(e);
            }
            Class<?> elemClass = null;
            try {
                elemClass = getElementClass(c);
            } catch (ClassNotFoundException e) {
                throw new IllegalArgumentException(e);
            }
            for (int i = 0; i < o.length; ++i) {
                res.set(i, _shadow(o[i], elemClass));
            }
            return res;
        } else if (c == s.java.util.List.class) {
            var o = (Object[]) obj;
            var sa = new IObject[o.length];
            int i = 0;
            for (var e : o) {
                sa[i++] = shadow(e);
            }
            return new UnmodifiableArrayList<>(sa);
        } else if (c == s.java.util.Map.class) {
            @SuppressWarnings("unchecked")
            var o = (Map<String, Object>) obj;
            var skv = new IObject[o.size() * 2];
            int i = 0;
            for (var e : o.entrySet()) {
                skv[i++] = shadow(e.getKey());
                skv[i++] = shadow(e.getValue());
            }
            return new UnmodifiableArrayMap<>(skv);
        } else {
            // this must be a writable struct
            try {
                @SuppressWarnings("unchecked")
                var o = (Map<String, Object>) obj;
                var ctor = c.getConstructor();
                var res = ctor.newInstance();
                for (var e : o.entrySet()) {
                    var wp = Property.getWritableProperty(c, e.getKey());
                    if (wp == null) {
                        throw new IllegalArgumentException();
                    }
                    wp.set(res, _shadow(e.getValue(), wp.getType()));
                }
                return res;
            } catch (NoSuchMethodException
                    | IllegalAccessException
                    | InstantiationException
                    | InvocationTargetException e) {
                throw new IllegalArgumentException();
            }
        }
    }

    /**
     * Shadows internal objects.
     * @param obj internal object
     * @return shadow object
     */
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
            @SuppressWarnings("unchecked")
            var o = (Map<String, Object>) obj;
            var skv = new IObject[o.size() * 2];
            int i = 0;
            for (Map.Entry<String, ?> e : o.entrySet()) {
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
