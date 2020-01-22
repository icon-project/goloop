package pi;

import a.ByteArray;
import foundation.icon.ee.io.DataWriter;
import i.IObject;
import i.IObjectArray;
import i.IObjectDeserializer;
import i.IObjectSerializer;
import i.RuntimeAssertionError;
import p.avm.Address;
import p.avm.ObjectWriter;

import java.lang.invoke.MethodHandle;
import java.lang.invoke.MethodHandles;
import java.lang.invoke.MethodType;

public class ObjectWriterImpl
        extends s.java.lang.Object
        implements ObjectWriter, AutoCloseable {
    private static MethodHandles.Lookup lookup = MethodHandles.lookup();

    private DataWriter writer;

    public ObjectWriterImpl(DataWriter writer) {
        this.writer = writer;
    }

    public void avm_write(boolean v) {
        writer.write(v);
    }

    public void avm_write(byte v) {
        writer.write(v);
    }

    public void avm_write(short v) {
        writer.write(v);
    }

    public void avm_write(char v) {
        writer.write(v);
    }

    public void avm_write(int v) {
        writer.write(v);
    }

    public void avm_write(float v) {
        writer.write(v);
    }

    public void avm_write(long v) {
        writer.write(v);
    }

    public void avm_write(double v) {
        writer.write(v);
    }

    public void avm_write(s.java.math.BigInteger v) {
        writer.write(v.getUnderlying());
    }

    public void avm_write(s.java.lang.String v) {
        writer.write(v.getUnderlying());
    }

    public void avm_write(ByteArray v) {
        writer.write(v.getUnderlying());
    }

    public void avm_write(Address v) {
        writer.write(v.toByteArray());
    }

    public void avm_write(IObject v) {
        var c = v.getClass();
        if (c == s.java.lang.Boolean.class) {
            avm_write(((s.java.lang.Boolean) v).getUnderlying());
        } else if (c == s.java.lang.Byte.class) {
            avm_write(((s.java.lang.Byte) v).getUnderlying());
        } else if (c == s.java.lang.Short.class) {
            avm_write(((s.java.lang.Short) v).getUnderlying());
        } else if (c == s.java.lang.Character.class) {
            avm_write(((s.java.lang.Character) v).getUnderlying());
        } else if (c == s.java.lang.Integer.class) {
            avm_write(((s.java.lang.Integer) v).getUnderlying());
        } else if (c == s.java.lang.Float.class) {
            avm_write(((s.java.lang.Float) v).getUnderlying());
        } else if (c == s.java.lang.Long.class) {
            avm_write(((s.java.lang.Long) v).getUnderlying());
        } else if (c == s.java.lang.Double.class) {
            avm_write(((s.java.lang.Double) v).getUnderlying());
        } else if (c == s.java.math.BigInteger.class) {
            avm_write((s.java.math.BigInteger) v);
        } else if (c == s.java.lang.String.class) {
            avm_write((s.java.lang.String) v);
        } else if (c == ByteArray.class) {
            avm_write((ByteArray) v);
        } else if (c == Address.class) {
            avm_write((Address) v);
        } else {
            MethodType mt = MethodType.methodType(void.class, ObjectWriter.class, c);
            MethodHandle mh;
            try {
                mh = lookup.findStatic(c, "avm_writeObject", mt);
            } catch (NoSuchMethodException | IllegalAccessException e) {
                e.printStackTrace();
                throw new IllegalArgumentException();
            }
            try {
                mh.invoke(this, v);
            } catch (RuntimeException e) {
                e.printStackTrace();
                throw e;
            } catch (Throwable t) {
                RuntimeAssertionError.unexpected(t);
            }
        }
    }

    public void avm_writeNullable(IObject v) {
        if (v != null) {
            avm_write(v);
        } else {
            avm_writeNull();
        }
    }

    public void avm_write(IObjectArray v) {
        for (int i = 0; i < v.length(); i++) {
            avm_write((IObject) v.get(i));
        }
    }

    public void avm_writeNullable(IObjectArray v) {
        for (int i = 0; i < v.length(); i++) {
            avm_writeNullable((IObject) v.get(i));
        }
    }

    public void avm_writeNull() {
        writer.writeNull();
    }

    public void avm_beginList(int l) {
        writer.writeListHeader(l);
    }

    public void avm_writeListOf(IObjectArray v) {
        avm_beginList(v.length());
        avm_write(v);
        avm_end();
    }

    public void avm_beginNullableList(int l) {
        writer.writeNullity(false);
        writer.writeListHeader(l);
    }

    public void avm_beginMap(int l) {
        writer.writeMapHeader(l);
    }

    public void avm_beginNullableMap(int l) {
        writer.writeNullity(false);
        writer.writeMapHeader(l);
    }

    public void avm_end() {
        writer.writeFooter();
    }

    public void flush() {
        writer.flush();
    }

    public byte[] toByteArray() {
        return writer.toByteArray();
    }

    public void close() {
        writer = null;
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(ObjectWriterImpl.class, deserializer);
        writer = null;
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(ObjectWriterImpl.class, serializer);
        assert writer == null;
    }
}
