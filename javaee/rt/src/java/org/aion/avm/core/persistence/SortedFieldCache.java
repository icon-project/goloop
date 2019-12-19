package org.aion.avm.core.persistence;

import java.lang.reflect.Constructor;
import java.lang.reflect.Field;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.lang.reflect.Modifier;
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;

import i.RuntimeAssertionError;


/**
 * Caches field/method and general reflection data associated with a specific loaded contract.
 * In the future, we might store these in the SoftCache, along-side the code.
 */
public class SortedFieldCache {
    private static final String CONSTANT_FIELD_PREFIX = "const_";

    private final Map<String, Class<?>> internalNameClasses;
    private final Map<Class<?>, Field[]> constantFields;
    private final Map<Class<?>, Field[]> staticFields;
    private final Map<Class<?>, Field[]> instanceFields;
    private final ClassLoader dappClassLoader;
    private final Method serializeSelf;
    private final Method deserializeSelf;
    private final Field readIndex;

    public SortedFieldCache(ClassLoader dappClassLoader, Method serializeSelf, Method deserializeSelf, Field readIndex) {
        this.internalNameClasses = new HashMap<>();
        this.constantFields = new HashMap<>();
        this.staticFields = new HashMap<>();
        this.instanceFields = new HashMap<>();
        this.dappClassLoader = dappClassLoader;
        this.serializeSelf = serializeSelf;
        this.deserializeSelf = deserializeSelf;
        this.readIndex = readIndex;
    }

    public Field[] getConstantFields(Class<?> clazz) {
        Field[] result = this.constantFields.get(clazz);
        if (null == result) {
            Field[] allFields = clazz.getDeclaredFields();
            result = Arrays.stream(allFields)
                    .filter((field) -> Modifier.STATIC == (Modifier.STATIC & field.getModifiers()))
                    .filter((field) -> field.getName().startsWith(CONSTANT_FIELD_PREFIX))
                    .sorted((f1, f2) -> f1.getName().compareTo(f2.getName()))
                    .map((field) -> {field.setAccessible(true); return field;})
                    .toArray(Field[]::new);
            this.constantFields.put(clazz, result);
        }
        return result;
    }

    public Field[] getUserStaticFields(Class<?> clazz) {
        Field[] result = this.staticFields.get(clazz);
        if (null == result) {
            Field[] allFields = clazz.getDeclaredFields();
            result = Arrays.stream(allFields)
                    .filter((field) -> Modifier.STATIC == (Modifier.STATIC & field.getModifiers()))
                    .filter((field) -> !field.getName().startsWith(CONSTANT_FIELD_PREFIX))
                    .sorted((f1, f2) -> f1.getName().compareTo(f2.getName()))
                    .map((field) -> {field.setAccessible(true); return field;})
                    .toArray(Field[]::new);
            this.staticFields.put(clazz, result);
        }
        return result;
    }

    public Field[] getInstanceFields(Class<?> clazz) {
        Field[] result = this.instanceFields.get(clazz);
        if (null == result) {
            Field[] allFields = clazz.getDeclaredFields();
            result = Arrays.stream(allFields)
                    .filter((field) -> Modifier.STATIC != (Modifier.STATIC & field.getModifiers()))
                    .sorted((f1, f2) -> f1.getName().compareTo(f2.getName()))
                    .map((field) -> {field.setAccessible(true); return field;})
                    .toArray(Field[]::new);
            this.instanceFields.put(clazz, result);
        }
        return result;
    }

    public Method getSerializeSelfMethod() {
        return this.serializeSelf;
    }

    public Method getDeserializeSelfMethod() {
        return this.deserializeSelf;
    }

    public Field getReadIndexField() {
        return this.readIndex;
    }

    public Object getNewInstance(String internalClassName, int readIndex) {
        Class<?> clazz = this.internalNameClasses.get(internalClassName);
        if (null == clazz) {
            try {
                clazz = this.dappClassLoader.loadClass(internalClassName);
            } catch (ClassNotFoundException e) {
                // We can't fail to find this since we wrote it to the datastore.
                throw RuntimeAssertionError.unexpected(e);
            }
            this.internalNameClasses.put(internalClassName, clazz);
        }
        // We define the Void class, since we just need to define a constructor that the user can't hook
        // into (and their references to this would be mapped to shadow). 
        try {
            Constructor<?> constructor = clazz.getConstructor(Void.class, int.class);
            constructor.setAccessible(true);
            return constructor.newInstance((Void)null, readIndex);
        } catch (NoSuchMethodException | SecurityException | InstantiationException | IllegalAccessException | IllegalArgumentException | InvocationTargetException e) {
            // We can't fail to find this since the type is datastore-safe.
            throw RuntimeAssertionError.unexpected(e);
        }
    }
}
