package pi;

import a.ByteArray;
import foundation.icon.ee.io.DataWriter;
import i.IObject;
import i.IObjectArray;
import i.IObjectDeserializer;
import i.IObjectSerializer;
import p.score.Address;
import p.score.ObjectWriter;

import java.lang.invoke.MethodHandle;
import java.lang.invoke.MethodHandles;
import java.lang.invoke.MethodType;
import java.util.Objects;

public class ObjectWriterImpl
        extends s.java.lang.Object
        implements ObjectWriter, AutoCloseable {
    private static final MethodHandles.Lookup lookup = MethodHandles.lookup();

    private DataWriter writer;
    private int level = 0;

    public ObjectWriterImpl(DataWriter writer) {
        this.writer = writer;
    }

    private void wrapWrite(Runnable r) {
        try {
            if (writer == null) {
                throw new IllegalStateException();
            }
            r.run();
        } catch (Exception e) {
            writer = null;
            throw e;
        }
    }

    public void avm_write(boolean v) {
        wrapWrite(() -> writer.write(v));
    }

    public void avm_write(byte v) {
        wrapWrite(() -> writer.write(v));
    }

    public void avm_write(short v) {
        wrapWrite(() -> writer.write(v));
    }

    public void avm_write(char v) {
        wrapWrite(() -> writer.write(v));
    }

    public void avm_write(int v) {
        wrapWrite(() -> writer.write(v));
    }

    public void avm_write(float v) {
        wrapWrite(() -> writer.write(v));
    }

    public void avm_write(long v) {
        wrapWrite(() -> writer.write(v));
    }

    public void avm_write(double v) {
        wrapWrite(() -> writer.write(v));
    }

    public void avm_write(s.java.math.BigInteger v) {
        wrapWrite(() -> {
            Objects.requireNonNull(v);
            writer.write(v.getUnderlying());
        });
    }

    public void avm_write(s.java.lang.String v) {
        wrapWrite(() -> {
            Objects.requireNonNull(v);
            writer.write(v.getUnderlying());
        });
    }

    public void avm_write(ByteArray v) {
        wrapWrite(() -> {
            Objects.requireNonNull(v);
            writer.write(v.getUnderlying());
        });
    }

    public void avm_write(Address v) {
        wrapWrite(() -> {
            Objects.requireNonNull(v);
            writer.write(v.toByteArray());
        });
    }

    public void avm_write(IObject v) {
        wrapWrite(() -> {
            Objects.requireNonNull(v);
            write(v);
        });
    }

    private void write(IObject v) {
        var c = v.getClass();
        if (c == s.java.lang.Boolean.class) {
            writer.write(((s.java.lang.Boolean) v).getUnderlying());
        } else if (c == s.java.lang.Byte.class) {
            writer.write(((s.java.lang.Byte) v).getUnderlying());
        } else if (c == s.java.lang.Short.class) {
            writer.write(((s.java.lang.Short) v).getUnderlying());
        } else if (c == s.java.lang.Character.class) {
            writer.write(((s.java.lang.Character) v).getUnderlying());
        } else if (c == s.java.lang.Integer.class) {
            writer.write(((s.java.lang.Integer) v).getUnderlying());
        } else if (c == s.java.lang.Float.class) {
            writer.write(((s.java.lang.Float) v).getUnderlying());
        } else if (c == s.java.lang.Long.class) {
            writer.write(((s.java.lang.Long) v).getUnderlying());
        } else if (c == s.java.lang.Double.class) {
            writer.write(((s.java.lang.Double) v).getUnderlying());
        } else if (c == s.java.math.BigInteger.class) {
            writer.write(((s.java.math.BigInteger) v).getUnderlying());
        } else if (c == s.java.lang.String.class) {
            writer.write(((s.java.lang.String) v).getUnderlying());
        } else if (c == ByteArray.class) {
            writer.write(((ByteArray) v).getUnderlying());
        } else if (c == Address.class) {
            writer.write(((Address) v).toByteArray());
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
            } catch (Throwable t) {
                t.printStackTrace();
                throw new IllegalArgumentException(t);
            }
        }
    }


    private void writeNullable(IObject v) {
        if (v == null) {
            writer.writeNullity(true);
        } else {
            writer.writeNullity(false);
            write(v);
        }
    }

    public void avm_writeNullable(IObject v) {
        wrapWrite(() -> writeNullable(v));
    }

    public void avm_write(IObjectArray v) {
        wrapWrite(() -> {
            for (int i = 0; i < v.length(); i++) {
                write((IObject) v.get(i));
            }
        });
    }

    public void avm_writeNullable(IObjectArray v) {
        wrapWrite(() -> {
            for (int i = 0; i < v.length(); i++) {
                writeNullable((IObject) v.get(i));
            }
        });
    }

    public void avm_writeNull() {
        wrapWrite(() -> writer.writeNullity(true));
    }

    public void avm_beginList(int l) {
        wrapWrite(() -> {
            ++level;
            writer.writeListHeader(l);
        });
    }

    public void avm_writeListOf(IObjectArray v) {
        wrapWrite(() -> {
            writer.writeListHeader(v.length());
            for (int i = 0; i < v.length(); i++) {
                Objects.requireNonNull(v.get(i));
                write((IObject) v.get(i));
            }
            writer.writeFooter();
        });
    }

    public void avm_writeListOfNullable(IObjectArray v) {
        wrapWrite(() -> {
            writer.writeListHeader(v.length());
            for (int i = 0; i < v.length(); i++) {
                writeNullable((IObject) v.get(i));
            }
            writer.writeFooter();
        });
    }

    public void avm_beginNullableList(int l) {
        wrapWrite(() -> {
            ++level;
            writer.writeNullity(false);
            writer.writeListHeader(l);
        });
    }

    public void avm_beginMap(int l) {
        wrapWrite(() -> {
            ++level;
            writer.writeMapHeader(l);
        });
    }

    public void avm_beginNullableMap(int l) {
        wrapWrite(() -> {
            ++level;
            writer.writeNullity(false);
            writer.writeMapHeader(l);
        });
    }

    public void avm_end() {
        wrapWrite(() -> {
            if (level == 0) {
                throw new IllegalStateException();
            }
            writer.writeFooter();
            --level;
        });
    }

    public void flush() {
        wrapWrite(() -> writer.flush());
    }

    public byte[] toByteArray() {
        wrapWrite(() -> writer.flush());
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
