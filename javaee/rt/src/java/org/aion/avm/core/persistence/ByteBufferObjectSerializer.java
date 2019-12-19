package org.aion.avm.core.persistence;

import java.lang.reflect.Field;
import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.util.IdentityHashMap;
import java.util.Queue;

import i.IObjectSerializer;
import i.RuntimeAssertionError;


public class ByteBufferObjectSerializer implements IObjectSerializer {
    private final ByteBuffer buffer;
    private final SortedFieldCache cache;
    private final IGlobalResolver resolver;
    private final IPersistenceNameMapper classNameMapper;
    private final InstanceIndexMapper instanceMapper;

    public ByteBufferObjectSerializer(ByteBuffer buffer, Queue<Object> out_ToProcessQueue, SortedFieldCache cache, IGlobalResolver resolver, IPersistenceNameMapper classNameMapper) {
        this.buffer = buffer;
        this.cache = cache;
        this.resolver = resolver;
        this.classNameMapper = classNameMapper;
        this.instanceMapper = new InstanceIndexMapper(out_ToProcessQueue);
    }

    @Override
    public void writeBoolean(boolean value) {
        this.buffer.put((byte) (value ? 0x1 : 0x0));
    }

    @Override
    public void writeByte(byte value) {
        this.buffer.put(value);
    }

    @Override
    public void writeShort(short value) {
        this.buffer.putShort(value);
    }

    @Override
    public void writeChar(char value) {
        this.buffer.putChar(value);
    }

    @Override
    public void writeInt(int value) {
        this.buffer.putInt(value);
    }

    @Override
    public void writeFloat(float value) {
        this.buffer.putInt(Float.floatToIntBits(value));
    }

    @Override
    public void writeLong(long value) {
        this.buffer.putLong(value);
    }

    @Override
    public void writeDouble(double value) {
        this.buffer.putLong(Double.doubleToLongBits(value));
    }

    @Override
    public void writeByteArray(byte[] value) {
        this.buffer.put(value);
    }

    @Override
    public void writeObject(Object value) {
        // Note that these need to be interpreted in a specific order, due to precedence of how difference kinds of immortal objects are resolved:
        // 1) Null - Since we can't proceed further with null
        // 2) Constant - Since some classes are constants and we need to see them as constants, first
        // 3) Class - Classes need to be interned, so they can't just be instantiated like normal instances
        // 4) Instance - Regular instances are the final case since they only have meaning within a graph
        if (null == value) {
            this.buffer.put(ReferenceConstants.REF_NULL);
        } else {
            int constantIdentifier = this.resolver.getAsConstant(value);
            if (0 != constantIdentifier) {
                this.buffer.put(ReferenceConstants.REF_CONSTANT);
                this.buffer.putInt(constantIdentifier);
            } else {
                String internalClassName = this.resolver.getAsInternalClassName(value);
                if (null != internalClassName) {
                    this.buffer.put(ReferenceConstants.REF_CLASS);
                    internalWriteClassName(internalClassName);
                } else {
                    int instanceIndex = instanceMapper.getIndexForInstance(value);
                    this.buffer.put(ReferenceConstants.REF_NORMAL);
                    this.buffer.putInt(instanceIndex);
                }
            }
        }
    }

    @Override
    public void writeClassName(String internalClassName) {
        internalWriteClassName(internalClassName);
    }

    @Override
    public void automaticallySerializeToRoot(Class<?> rootClass, Object instance) {
        // This is called after any root information has been serialized, including class name and root instance variables.
        // So, we just need to serialize all the fields defined by other classes, excluding the root class.
        
        // This means, we need to recursively walk to the root.
        internalSerializeFieldsToRoot(rootClass, instance.getClass(), instance);
    }


    private void internalSerializeFieldsToRoot(Class<?> rootClass, Class<?> thisClass, Object instance) {
        if (rootClass != thisClass) {
            // We can serialize this one, but first see if we need to call a superclass.
            internalSerializeFieldsToRoot(rootClass, thisClass.getSuperclass(), instance);
            
            // Now, serialize the fields in this level.
            try {
                Field[] fields = this.cache.getInstanceFields(thisClass);
                for (Field field : fields) {
                    // We need to crack the type, here.
                    Class<?> type = field.getType();
                    if (boolean.class == type) {
                        boolean val = field.getBoolean(instance);
                        this.writeBoolean(val);
                    } else if (byte.class == type) {
                        byte val = field.getByte(instance);
                        this.writeByte(val);
                    } else if (short.class == type) {
                        short val = field.getShort(instance);
                        this.writeShort(val);
                    } else if (char.class == type) {
                        char val = field.getChar(instance);
                        this.writeChar(val);
                    } else if (int.class == type) {
                        int val = field.getInt(instance);
                        this.writeInt(val);
                    } else if (float.class == type) {
                        float actual = field.getFloat(instance);
                        this.writeFloat(actual);
                    } else if (long.class == type) {
                        long val = field.getLong(instance);
                        this.writeLong(val);
                    } else if (double.class == type) {
                        double actual = field.getDouble(instance);
                        this.writeDouble(actual);
                    } else {
                        // Object types require further logic.
                        Object target = field.get(instance);
                        this.writeObject(target);
                    }
                }
            } catch (IllegalAccessException e) {
                // Reflection errors can't happen since we set this up so we could access it.
                throw RuntimeAssertionError.unexpected(e);
            }
        }
    }

    private void internalWriteClassName(String internalClassName) {
        String storageName = this.classNameMapper.getStorageClassName(internalClassName);
        byte[] utf8 = storageName.getBytes(StandardCharsets.UTF_8);
        // We limit class names to 255 UTF-8 bytes so read the length byte.
        RuntimeAssertionError.assertTrue(utf8.length > 0);
        RuntimeAssertionError.assertTrue(utf8.length <= 255);
        this.buffer.put((byte) utf8.length);
        this.buffer.put(utf8);
    }


    private static class InstanceIndexMapper {
        private int nextInstanceIndex;
        private final IdentityHashMap<Object, Integer> instanceToIndex;
        private final Queue<Object> sharedToProcessQueue;
        
        public InstanceIndexMapper(Queue<Object> sharedToProcessQueue) {
            this.nextInstanceIndex = 0;
            this.instanceToIndex = new IdentityHashMap<>();
            this.sharedToProcessQueue = sharedToProcessQueue;
        }
        
        public int getIndexForInstance(Object instance) {
            Integer indexObject = this.instanceToIndex.get(instance);
            if (null == indexObject) {
                indexObject = this.nextInstanceIndex;
                this.nextInstanceIndex += 1;
                this.instanceToIndex.put(instance, indexObject);
                // This is the first time we saw this, in breadth-first traversal order, so add it to the queue.
                this.sharedToProcessQueue.add(instance);
            }
            return indexObject;
        }
    }
}
