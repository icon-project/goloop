package foundation.icon.ee.util;

import a.ByteArray;
import foundation.icon.ee.io.RLPNDataReader;
import foundation.icon.ee.io.RLPNDataWriter;
import i.IObject;
import p.score.Address;
import pi.ObjectReaderImpl;
import pi.ObjectWriterImpl;
import s.java.lang.Boolean;
import s.java.lang.Byte;
import s.java.lang.Character;
import s.java.lang.Class;
import s.java.lang.Integer;
import s.java.lang.Long;
import s.java.lang.Short;
import s.java.lang.String;

import java.math.BigInteger;
import java.nio.charset.StandardCharsets;

public class ValueCodec {
    public static byte[] encode(IObject o) {
        if (o == null) {
            return null;
        } else if (o instanceof Byte) {
            return BigInteger.valueOf(((Byte) o).getUnderlying()).toByteArray();
        } else if (o instanceof Short) {
            return BigInteger.valueOf(((Short) o).getUnderlying()).toByteArray();
        } else if (o instanceof Integer) {
            return BigInteger.valueOf(((Integer) o).getUnderlying()).toByteArray();
        } else if (o instanceof Long) {
            return BigInteger.valueOf(((Long) o).getUnderlying()).toByteArray();
        } else if (o instanceof s.java.math.BigInteger) {
            return ((s.java.math.BigInteger) o).getUnderlying().toByteArray();
        } else if (o instanceof Character) {
            return BigInteger.valueOf(((Character) o).getUnderlying()).toByteArray();
        } else if (o instanceof Boolean) {
            return BigInteger.valueOf(((Boolean) o).getUnderlying() ? 1 : 0).toByteArray();
        } else if (o instanceof Address) {
            return ((Address) o).toByteArray();
        } else if (o instanceof String) {
            return ((String) o).getUnderlying().getBytes(StandardCharsets.UTF_8);
        } else if (o instanceof ByteArray) {
            return ((ByteArray) o).getUnderlying().clone();
        } else {
            try (var owi = new ObjectWriterImpl(new RLPNDataWriter())) {
                owi.avm_write(o);
                return owi.toByteArray();
            }
        }
    }

    public static IObject decode(byte[] raw, Class<?> cls) {
        if (raw == null)
            return null;
        var c = cls.getRealClass();
        if (c == Byte.class) {
            return Byte.avm_valueOf(new BigInteger(raw).byteValue());
        } else if (c == Short.class) {
            return Short.avm_valueOf(new BigInteger(raw).shortValue());
        } else if (c == Integer.class) {
            return Integer.avm_valueOf(new BigInteger(raw).intValue());
        } else if (c == Long.class) {
            return Long.avm_valueOf(new BigInteger(raw).longValue());
        } else if (c == s.java.math.BigInteger.class) {
            return s.java.math.BigInteger.newWithCharge(new BigInteger(raw));
        } else if (c == Character.class) {
            return Character.avm_valueOf((char) new BigInteger(raw).intValue());
        } else if (c == Boolean.class) {
            return Boolean.avm_valueOf(new BigInteger(raw).intValue() != 0);
        } else if (c == Address.class) {
            return Address.newWithCharge(raw);
        } else if (c == String.class) {
            return String.newWithCharge(new java.lang.String(raw, StandardCharsets.UTF_8));
        } else if (c == ByteArray.class) {
            return ByteArray.newWithCharge(raw.clone());
        } else {
            try (var ori = new ObjectReaderImpl(new RLPNDataReader(raw))) {
                return ori.avm_read((Class<? extends IObject>)cls);
            }
        }
    }
}
