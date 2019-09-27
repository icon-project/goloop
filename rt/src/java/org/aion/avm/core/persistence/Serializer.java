package org.aion.avm.core.persistence;

import java.lang.reflect.Field;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.nio.BufferOverflowException;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.util.LinkedList;
import java.util.List;
import java.util.Queue;

import i.OutOfEnergyException;
import i.RuntimeAssertionError;


public class Serializer {
    // (Should make this Map a list since the graph is probably dense?)
    public static void serializeEntireGraph(ByteBuffer outputBuffer, List<Object> out_instanceIndex, List<Integer> out_calleeToCallerIndexMap, IGlobalResolver resolver, SortedFieldCache cache, IPersistenceNameMapper classNameMapper, int nextHashCode, Class<?>[] sortedRoots, Class<?> constantClass) {
        // We define the storage as big-endian.
        RuntimeAssertionError.assertTrue(ByteOrder.BIG_ENDIAN == outputBuffer.order());
        // We cannot be both serializing to build an index (that is done when serializing caller state before entering a callee frame)
        // and serializing to build a mapping (that is done when serializing the callee frame before using it to interpret remap the caller's graph).
        // In the common case, we are doing neither (these arguments are only used for reentrant calls).
        RuntimeAssertionError.assertTrue((null == out_instanceIndex) || (null == out_calleeToCallerIndexMap));
        
        // We can write the next hash, first, since we already know it.
        outputBuffer.putInt(nextHashCode);
        
        // We are going to perform a breadth-first traversal so we need a queue.
        Queue<Object> toProcessQueue = new LinkedList<>();
        // Create the object serializer (it maintains the state of the serialization and can also be passed in to objects to request that they serialize).
        ByteBufferObjectSerializer objectSerializer = new ByteBufferObjectSerializer(outputBuffer, toProcessQueue, cache, resolver, classNameMapper);
        
        // Next, we serialize all the class statics from the user's classes.
        serializeClassStatics(objectSerializer, cache, sortedRoots, constantClass);
        
        // Finally, we serialize the rest of the graph.
        serializeGraphFromWorkQueue(out_instanceIndex, out_calleeToCallerIndexMap, objectSerializer, cache, toProcessQueue);
    }



    private static void serializeClassStatics(ByteBufferObjectSerializer objectSerializer, SortedFieldCache cache, Class<?>[] sortedRoots, Class<?> constantClass) {
        try {
            // First, we serialize the constants.
            serializeConstantClass(objectSerializer, cache, constantClass);
            // Then, we serialize the user-defined static fields.
            for (Class<?> clazz : sortedRoots) {
                serializeOneUserClass(objectSerializer, cache, clazz);
            }
        } catch (BufferOverflowException e) {
            // This is if we run off the end of the buffer, which is an example of out of energy.
            throw new OutOfEnergyException();
        }
    }

    private static void serializeConstantClass(ByteBufferObjectSerializer objectSerializer, SortedFieldCache cache, Class<?> clazz) {
        // Note that we don't serialize the class name - the roots are in the same sorted order for reading and writing.
        Field[] constants = cache.getConstantFields(clazz);
        serializeFieldsForClass(objectSerializer, constants);
    }

    private static void serializeOneUserClass(ByteBufferObjectSerializer objectSerializer, SortedFieldCache cache, Class<?> clazz) {
        // Note that we don't serialize the class name - the roots are in the same sorted order for reading and writing.
        Field[] fields = cache.getUserStaticFields(clazz);
        serializeFieldsForClass(objectSerializer, fields);
    }

    private static void serializeGraphFromWorkQueue(List<Object> out_instanceIndex, List<Integer> out_calleeToCallerIndexMap, ByteBufferObjectSerializer objectSerializer, SortedFieldCache cache, Queue<Object> toProcessQueue) {
        Method serializeSelfMethod = cache.getSerializeSelfMethod();
        Field readIndexField = cache.getReadIndexField();
        
        try {
            while (!toProcessQueue.isEmpty()) {
                Object instance = toProcessQueue.remove();
                // We first need to serialize the class name.
                String internalClassName = instance.getClass().getName();
                objectSerializer.writeClassName(internalClassName);
                serializeSelfMethod.invoke(instance, null, objectSerializer);
                if (null != out_instanceIndex) {
                    out_instanceIndex.add(instance);
                } else if (null != out_calleeToCallerIndexMap) {
                    int readIndex = readIndexField.getInt(instance);
                    out_calleeToCallerIndexMap.add(readIndex);
                }
            }
        } catch (InvocationTargetException e) {
            Throwable cause = e.getCause();
            if (cause instanceof OutOfEnergyException) {
                // This can happen within our deserialization path for various reasons.
                throw (OutOfEnergyException) cause;
            } else if (cause instanceof BufferOverflowException) {
                // This is if we run off the end of the buffer, which is an example of out of energy.
                throw new OutOfEnergyException();
            } else {
                // This shouldn't happen but is distinct from reflection errors.
                throw RuntimeAssertionError.unexpected(e);
            }
        } catch (IllegalAccessException | IllegalArgumentException e) {
            // Reflection errors can't happen since we set this up so we could access it.
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    private static void serializeFieldsForClass(ByteBufferObjectSerializer objectSerializer, Field[] fields) {
        try {
            for (Field field : fields) {
                // We need to crack the type, here.
                Class<?> type = field.getType();
                if (boolean.class == type) {
                    boolean val = field.getBoolean(null);
                    objectSerializer.writeBoolean(val);
                } else if (byte.class == type) {
                    byte val = field.getByte(null);
                    objectSerializer.writeByte(val);
                } else if (short.class == type) {
                    short val = field.getShort(null);
                    objectSerializer.writeShort(val);
                } else if (char.class == type) {
                    char val = field.getChar(null);
                    objectSerializer.writeChar(val);
                } else if (int.class == type) {
                    int val = field.getInt(null);
                    objectSerializer.writeInt(val);
                } else if (float.class == type) {
                    float actual = field.getFloat(null);
                    objectSerializer.writeFloat(actual);
                } else if (long.class == type) {
                    long val = field.getLong(null);
                    objectSerializer.writeLong(val);
                } else if (double.class == type) {
                    double actual = field.getDouble(null);
                    objectSerializer.writeDouble(actual);
                } else {
                    Object target = field.get(null);
                    objectSerializer.writeObject(target);
                }
            }
        } catch (IllegalAccessException e) {
            // Reflection errors can't happen since we set this up so we could access it.
            throw RuntimeAssertionError.unexpected(e);
        }
    }
}
