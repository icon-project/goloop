package pi;

import a.ByteArray;
import foundation.icon.ee.io.DataReader;
import i.IObject;
import i.IObjectDeserializer;
import i.IObjectSerializer;
import i.RuntimeAssertionError;
import p.score.Address;
import p.score.ObjectReader;

import java.lang.invoke.MethodHandle;
import java.lang.invoke.MethodHandles;
import java.lang.invoke.MethodType;
import java.util.NoSuchElementException;

public class ObjectReaderImpl
        extends s.java.lang.Object
        implements ObjectReader, AutoCloseable {
    private static MethodHandles.Lookup lookup = MethodHandles.lookup();

    private DataReader reader;

    public ObjectReaderImpl(DataReader reader) {
        this.reader = reader;
    }

    public boolean avm_readBoolean() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        return reader.readBoolean();
    }

    public byte avm_readByte() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        return reader.readByte();
    }

    public short avm_readShort() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        return reader.readShort();
    }

    public char avm_readChar() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        return reader.readChar();
    }

    public int avm_readInt() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        return reader.readInt();
    }

    public float avm_readFloat() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        return reader.readFloat();
    }

    public long avm_readLong() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        return reader.readLong();
    }

    public double avm_readDouble() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        return reader.readDouble();
    }

    public s.java.math.BigInteger avm_readBigInteger() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        var u = reader.readBigInteger();
        return s.java.math.BigInteger.newWithCharge(u);
    }

    public s.java.lang.String avm_readString() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        var u = reader.readString();
        return s.java.lang.String.newWithCharge(u);
    }

    public ByteArray avm_readByteArray() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        var u = reader.readByteArray();
        return ByteArray.newWithCharge(u);
    }

    public Address avm_readAddress() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        byte[] u = reader.readByteArray();
        if (u.length != Address.LENGTH) {
            throw new IllegalStateException();
        }
        return Address.newWithCharge(u);
    }

    public <T extends IObject> T avm_read(s.java.lang.Class<T> c) {
        return read(c.getRealClass());
    }

    public <T extends IObject> T read(Class<T> c) {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        return (T) _read(c);
    }

    public <T extends IObject> T avm_readOrDefault(s.java.lang.Class<T> c, T def) {
        try {
            return avm_read(c);
        } catch (NoSuchElementException e) {
            return def;
        }
    }

    public <T extends IObject> T avm_readNullable(s.java.lang.Class<T> c) {
        return readNullable(c.getRealClass());
    }

    public <T extends IObject> T readNullable(Class<T> c) {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        if (reader.readNullity()) {
            return null;
        }
        return (T) _read(c);
    }

    public <T extends IObject> T avm_readNullableOrDefault(s.java.lang.Class<T> c, T def) {
        try {
            var res = avm_readNullable(c);
            return res != null ? res : def;
        } catch (NoSuchElementException e) {
            return def;
        }
    }

    public void avm_beginList() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        reader.readListHeader();
    }

    public void avm_beginMap() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        reader.readMapHeader();
    }

    public void avm_beginNullableList() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        if (reader.readNullity()) {
            throw new IllegalStateException();
        }
        reader.readListHeader();
    }

    public void avm_beginNullableMap() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        if (reader.readNullity()) {
            throw new IllegalStateException();
        }
        reader.readMapHeader();
    }

    public boolean avm_tryBeginNullableList() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        if (reader.readNullity()) {
            return false;
        }
        reader.readListHeader();
        return true;
    }

    public boolean avm_tryBeginNullableMap() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        if (reader.readNullity()) {
            return false;
        }
        reader.readMapHeader();
        return true;
    }

    public boolean avm_hasNext() {
        return reader.hasNext();
    }

    public void avm_end() {
        while (reader.hasNext()) {
            reader.skip(1);
        }
        reader.readFooter();
    }

    public boolean avm_tryReadNull() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        return reader.tryReadNull();
    }

    public <T extends IObject> IObject _read(Class<T> c) {
        if (c == s.java.lang.Boolean.class) {
            return s.java.lang.Boolean.avm_valueOf(avm_readBoolean());
        } else if (c == s.java.lang.Byte.class) {
            return s.java.lang.Byte.avm_valueOf(avm_readByte());
        } else if (c == s.java.lang.Short.class) {
            return s.java.lang.Short.avm_valueOf(avm_readShort());
        } else if (c == s.java.lang.Character.class) {
            return s.java.lang.Character.avm_valueOf(avm_readChar());
        } else if (c == s.java.lang.Integer.class) {
            return s.java.lang.Integer.avm_valueOf(avm_readInt());
        } else if (c == s.java.lang.Float.class) {
            return s.java.lang.Float.avm_valueOf(avm_readFloat());
        } else if (c == s.java.lang.Long.class) {
            return s.java.lang.Long.avm_valueOf(avm_readLong());
        } else if (c == s.java.lang.Double.class) {
            return s.java.lang.Double.avm_valueOf(avm_readDouble());
        } else if (c == s.java.lang.String.class) {
            return avm_readString();
        } else if (c == s.java.math.BigInteger.class) {
            return avm_readBigInteger();
        } else if (c == ByteArray.class) {
            return avm_readByteArray();
        } else if (c == Address.class) {
            return avm_readAddress();
        } else {
            MethodType mt = MethodType.methodType(c, ObjectReader.class);
            MethodHandle mh = null;
            try {
                mh = lookup.findStatic(c, "avm_readObject", mt);
            } catch (NoSuchMethodException | IllegalAccessException e) {
                e.printStackTrace();
                throw new IllegalArgumentException();
            }
            try {
                return (IObject) mh.invoke(this);
            } catch (RuntimeException e) {
                e.printStackTrace();
                throw e;
            } catch (Throwable t) {
                RuntimeAssertionError.unexpected(t);
                return null;
            }
        }
    }

    public void avm_skip() {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        reader.skip(1);
    }

    public void avm_skip(int count) {
        if (!reader.hasNext()) {
            throw new NoSuchElementException();
        }
        reader.skip(count);
    }

    public void close() {
        reader = null;
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(ObjectReaderImpl.class, deserializer);
        reader = null;
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(ObjectReaderImpl.class, serializer);
        assert reader == null;
    }
}
