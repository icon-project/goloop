package org.aion.avm.core.persistence;

import java.lang.reflect.Field;
import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.util.List;

import i.IObjectDeserializer;
import i.RuntimeAssertionError;


public class ByteBufferObjectDeserializer implements IObjectDeserializer {
    private final ByteBuffer buffer;
    private final SortedFieldCache cache;
    private final IGlobalResolver resolver;
    private final IPersistenceNameMapper classNameMapper;
    // Note that this will be null if this is our pre-pass where we are merely walking through the buffer to find the instance types.
    private final List<Object> instanceList;

    public ByteBufferObjectDeserializer(ByteBuffer buffer, List<Object> instanceList, SortedFieldCache cache, IGlobalResolver resolver, IPersistenceNameMapper classNameMapper) {
        this.buffer = buffer;
        this.cache = cache;
        this.resolver = resolver;
        this.classNameMapper = classNameMapper;
        this.instanceList = instanceList;
    }

    @Override
    public boolean readBoolean() {
        return ((byte)0x1 == this.buffer.get());
    }

    @Override
    public byte readByte() {
        return this.buffer.get();
    }

    @Override
    public short readShort() {
        return this.buffer.getShort();
    }

    @Override
    public char readChar() {
        return this.buffer.getChar();
    }

    @Override
    public int readInt() {
        return this.buffer.getInt();
    }

    @Override
    public float readFloat() {
        return Float.intBitsToFloat(this.buffer.getInt());
    }

    @Override
    public long readLong() {
        return this.buffer.getLong();
    }

    @Override
    public double readDouble() {
        return Double.longBitsToDouble(this.buffer.getLong());
    }

    @Override
    public void readByteArray(byte[] result) {
        this.buffer.get(result);
    }

    @Override
    public Object readObject() {
        // NOTE:  If the instance list is null, this is a pre-pass, meaning we shouldn't try to create/resolve the objects, since we will do this again.
        byte refType = this.buffer.get();
        Object result = null;
        switch (refType) {
            case ReferenceConstants.REF_NULL: {
                result = null;
                break;
            }
            case ReferenceConstants.REF_CLASS: {
                String internalClassName = internalReadClassName();
                result = (null != this.instanceList)
                        ? this.resolver.getClassObjectForInternalName(internalClassName)
                        : null;
                break;
            }
            case ReferenceConstants.REF_CONSTANT: {
                int constantIdentifier = this.buffer.getInt();
                result = (null != this.instanceList)
                        ? this.resolver.getConstantForIdentifier(constantIdentifier)
                        : null;
                break;
            }
            case ReferenceConstants.REF_NORMAL: {
                int instanceIndex = this.buffer.getInt();
                result = (null != this.instanceList)
                        ? this.instanceList.get(instanceIndex)
                        : null;
                break;
            }
            default:
                throw RuntimeAssertionError.unreachable("Unknown byte");
        }
        return result;
    }

    @Override
    public String readClassName() {
        return internalReadClassName();
    }

    @Override
    public void automaticallyDeserializeFromRoot(Class<?> rootClass, Object instance) {
        // This is called after any rootClass instance variables have been deserialized.
        // So, we just need to deserialize all the fields defined by other classes, excluding the root class.
        
        // This means, we need to recursively walk to the root.
        internalDeserializeFieldsFromRoot(rootClass, instance.getClass(), instance);
    }


    private void internalDeserializeFieldsFromRoot(Class<?> rootClass, Class<?> thisClass, Object instance) {
        if (rootClass != thisClass) {
            // We can deserialize this one, but first see if we need to call a superclass.
            internalDeserializeFieldsFromRoot(rootClass, thisClass.getSuperclass(), instance);
            
            // Now, serialize the fields in this level.
            try {
                Field[] fields = this.cache.getInstanceFields(thisClass);
                for (Field field : fields) {
                    // We need to crack the type, here.
                    Class<?> type = field.getType();
                    if (boolean.class == type) {
                        boolean val = this.readBoolean();
                        field.setBoolean(instance, val);
                    } else if (byte.class == type) {
                        byte val = this.readByte();
                        field.setByte(instance, val);
                    } else if (short.class == type) {
                        short val = this.readShort();
                        field.setShort(instance, val);
                    } else if (char.class == type) {
                        char val = this.readChar();
                        field.setChar(instance, val);
                    } else if (int.class == type) {
                        int val = this.readInt();
                        field.setInt(instance, val);
                    } else if (float.class == type) {
                        float val = this.readFloat();
                        field.setFloat(instance, val);
                    } else if (long.class == type) {
                        long val = this.readLong();
                        field.setLong(instance, val);
                    } else if (double.class == type) {
                        double val = this.readDouble();
                        field.setDouble(instance, val);
                    } else {
                        // Object types require further logic.
                        Object val = this.readObject();
                        field.set(instance, val);
                    }
                }
            } catch (IllegalAccessException e) {
                // Reflection errors can't happen since we set this up so we could access it.
                throw RuntimeAssertionError.unexpected(e);
            }
        }
    }

    private String internalReadClassName() {
        // We limit class names to 255 UTF-8 bytes so read the length byte.
        int length = (0xff & this.buffer.get());
        RuntimeAssertionError.assertTrue(length > 0);
        byte[] utf8 = new byte[length];
        this.buffer.get(utf8);
        String storageClassName = new String(utf8, StandardCharsets.UTF_8);
        return this.classNameMapper.getInternalClassName(storageClassName);
    }
}
