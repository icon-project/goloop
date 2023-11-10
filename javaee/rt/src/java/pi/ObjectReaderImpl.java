package pi;

import a.ByteArray;
import foundation.icon.ee.io.DataReader;
import foundation.icon.ee.types.Status;
import i.GenericPredefinedException;
import i.IInstrumentation;
import i.IObject;
import i.IObjectDeserializer;
import i.IObjectSerializer;
import i.RuntimeAssertionError;
import org.aion.avm.RuntimeMethodFeeSchedule;
import p.score.Address;
import p.score.ObjectReader;

import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Modifier;
import java.util.function.Supplier;

public class ObjectReaderImpl
        extends s.java.lang.Object
        implements ObjectReader, AutoCloseable {
    private DataReader reader;
    private int level = 0;
    private long lastChargePos = 0;

    public ObjectReaderImpl(DataReader reader) {
        this.reader = reader;
    }

    private void charge() {
        var pos = reader.getTotalReadBytes();
        int l = (int)(pos - lastChargePos);
        IInstrumentation.charge(
                RuntimeMethodFeeSchedule.ObjectReader_readPricePerByte * l
        );
        lastChargePos = pos;
    }

    private void chargeSkip() {
        var pos = reader.getTotalReadBytes();
        int l = (int)(pos - lastChargePos);
        IInstrumentation.charge(
                RuntimeMethodFeeSchedule.ObjectReader_skipPricePerByte * l
        );
        lastChargePos = pos;
    }

    private <T> T wrapRead(Supplier<T> s) {
        try {
            if (reader == null || !reader.hasNext()) {
                throw new IllegalStateException();
            }
            T ret =  s.get();
            charge();
            return ret;
        } catch (Exception e) {
            reader = null;
            throw e;
        }
    }

    private void wrapVoidRead(Runnable r) {
        try {
            if (reader == null || !reader.hasNext()) {
                throw new IllegalStateException();
            }
            r.run();
            charge();
        } catch (Exception e) {
            reader = null;
            throw e;
        }
    }

    private <T> T wrapReadOrDefault(Supplier<T> s, T def) {
        try {
            if (reader == null) {
                throw new IllegalStateException();
            }
            if (!reader.hasNext()) {
                return def;
            }
            T ret =  s.get();
            charge();
            return ret;
        } catch (Exception e) {
            reader = null;
            throw e;
        }
    }

    public boolean avm_readBoolean() {
        return wrapRead(() -> reader.readBoolean());
    }

    public byte avm_readByte() {
        return wrapRead(() -> reader.readByte());
    }

    public short avm_readShort() {
        return wrapRead(() -> reader.readShort());
    }

    public char avm_readChar() {
        return wrapRead(() -> reader.readChar());
    }

    public int avm_readInt() {
        return wrapRead(() -> reader.readInt());
    }

    public float avm_readFloat() {
        return wrapRead(() -> reader.readFloat());
    }

    public long avm_readLong() {
        return wrapRead(() -> reader.readLong());
    }

    public double avm_readDouble() {
        return wrapRead(() -> reader.readDouble());
    }

    public s.java.math.BigInteger avm_readBigInteger() {
        return wrapRead(() -> {
            var u = reader.readBigInteger();
            return s.java.math.BigInteger.newWithCharge(u);
        });
    }

    public s.java.lang.String avm_readString() {
        return wrapRead(() -> {
            var u = reader.readString();
            return s.java.lang.String.newWithCharge(u);
        });
    }

    public ByteArray avm_readByteArray() {
        return wrapRead(() -> {
            var u = reader.readByteArray();
            return ByteArray.newWithCharge(u);
        });
    }

    public Address avm_readAddress() {
        return wrapRead(() -> {
            byte[] u = reader.readByteArray();
            if (u.length != Address.LENGTH) {
                throw new IllegalStateException();
            }
            return Address.newWithCharge(u);
        });
    }

    public <T extends IObject> T avm_read(s.java.lang.Class<T> c) {
        return wrapRead(() -> read(c));
    }

    public <T extends IObject> T read(s.java.lang.Class<T> c) {
        @SuppressWarnings("unchecked")
        T res = (T) _read(c.getRealClass());
        return res;
    }

    public <T extends IObject> T avm_readOrDefault(s.java.lang.Class<T> c, T def) {
        return wrapReadOrDefault(() -> read(c), def);
    }

    public <T extends IObject> T avm_readNullable(s.java.lang.Class<T> c) {
        return wrapRead(() -> readNullable(c));
    }

    private <T extends IObject> T readNullable(s.java.lang.Class<T> c) {
        if (reader.readNullity()) {
            charge();
            return null;
        }
        charge();
        return read(c);
    }

    public <T extends IObject> T avm_readNullableOrDefault(s.java.lang.Class<T> c, T def) {
        return wrapReadOrDefault(() -> readNullable(c), def);
    }

    public void avm_beginList() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.ObjectReader_beginBase);
        wrapVoidRead(() -> {
            ++level;
            reader.readListHeader();
        });
    }

    public void avm_beginMap() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.ObjectReader_beginBase);
        wrapVoidRead(() -> {
            ++level;
            reader.readMapHeader();
        });
    }

    public boolean avm_beginNullableList() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.ObjectReader_beginBase);
        return wrapRead(() -> {
            if (reader.readNullity()) {
                return false;
            }
            charge();
            ++level;
            reader.readListHeader();
            return true;
        });
    }

    public boolean avm_beginNullableMap() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.ObjectReader_beginBase);
        return wrapRead(() -> {
            if (reader.readNullity()) {
                return false;
            }
            charge();
            ++level;
            reader.readMapHeader();
            return true;
        });
    }

    public boolean avm_hasNext() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.ObjectReader_hasNext);
        try {
            if (reader == null) {
                throw new IllegalStateException();
            }
            return reader.hasNext();
        } catch (Exception e) {
            reader = null;
            throw e;
        }
    }

    public void avm_end() {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.ObjectReader_endBase);
        try {
            if (reader == null) {
                throw new IllegalStateException();
            }
            if (level == 0) {
                throw new IllegalStateException();
            }
            while (reader.hasNext()) {
                reader.skip(1);
            }
            chargeSkip();
            reader.readFooter();
            charge();
            --level;
        } catch (Exception e) {
            reader = null;
            throw e;
        }
    }

    public <T extends IObject> IObject _read(Class<T> c) {
        if (c == s.java.lang.Boolean.class) {
            return s.java.lang.Boolean.avm_valueOf(reader.readBoolean());
        } else if (c == s.java.lang.Byte.class) {
            return s.java.lang.Byte.avm_valueOf(reader.readByte());
        } else if (c == s.java.lang.Short.class) {
            return s.java.lang.Short.avm_valueOf(reader.readShort());
        } else if (c == s.java.lang.Character.class) {
            return s.java.lang.Character.avm_valueOf(reader.readChar());
        } else if (c == s.java.lang.Integer.class) {
            return s.java.lang.Integer.avm_valueOf(reader.readInt());
        } else if (c == s.java.lang.Float.class) {
            return s.java.lang.Float.avm_valueOf(reader.readFloat());
        } else if (c == s.java.lang.Long.class) {
            return s.java.lang.Long.avm_valueOf(reader.readLong());
        } else if (c == s.java.lang.Double.class) {
            return s.java.lang.Double.avm_valueOf(reader.readDouble());
        } else if (c == s.java.lang.String.class) {
            var u = reader.readString();
            return s.java.lang.String.newWithCharge(u);
        } else if (c == s.java.math.BigInteger.class) {
            var u = reader.readBigInteger();
            return s.java.math.BigInteger.newWithCharge(u);
        } else if (c == ByteArray.class) {
            var u = reader.readByteArray();
            return ByteArray.newWithCharge(u);
        } else if (c == Address.class) {
            byte[] u = reader.readByteArray();
            if (u.length != Address.LENGTH) {
                throw new IllegalStateException();
            }
            return Address.newWithCharge(u);
        } else {
            IInstrumentation.charge(
                    RuntimeMethodFeeSchedule.ObjectReader_customMethodBase
            );
            try {
                var m = c.getDeclaredMethod("avm_readObject", ObjectReader.class);
                if ((m.getModifiers()& Modifier.STATIC) == 0
                        || (m.getModifiers()&Modifier.PUBLIC) == 0) {
                    throw new IllegalArgumentException();
                }
                var res = m.invoke(null, this);
                return (IObject) res;
            } catch (NoSuchMethodException
                    | IllegalAccessException
                    | InvocationTargetException e) {
                e.printStackTrace();
                throw new IllegalArgumentException();
            }
        }
    }

    public void avm_skip() {
        wrapVoidRead(() -> {
            reader.skip(1);
            chargeSkip();
        });
    }

    private static final int STEP = 1<< 16;

    public void avm_skip(int count) {
        if (count <= 0) {
            return;
        }
        wrapVoidRead(() -> {
            int l = count;
            for (; l>=STEP; l-=STEP) {
                reader.skip(STEP);
                chargeSkip();
            }
            reader.skip(l);
            chargeSkip();
        });
    }

    public void close() {
        reader = null;
        level = 0;
        lastChargePos = 0;
    }

    public ObjectReaderImpl(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        RuntimeAssertionError.unimplemented("cannot deserialize ObjectReaderImpl");
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        throw new GenericPredefinedException(Status.IllegalObjectGraph);
    }
}
