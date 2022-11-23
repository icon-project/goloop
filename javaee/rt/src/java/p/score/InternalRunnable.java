/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package p.score;

import java.lang.invoke.MethodHandle;
import java.lang.invoke.MethodHandleInfo;
import java.lang.invoke.MethodHandles;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;

import foundation.icon.ee.types.UnknownFailureException;
import i.CodecIdioms;
import i.IInstrumentation;
import i.IObjectDeserializer;
import i.IObjectSerializer;
import i.RuntimeAssertionError;


public final class InternalRunnable extends s.java.lang.Object implements s.java.lang.Runnable {
    private static final String METHOD_PREFIX = "avm_";

    public static InternalRunnable createRunnable(MethodHandles.Lookup lookup, MethodHandle target) {
        // Note that we need to convert this from a MethodHandle to a traditional reflection Method since we need to serialize it
        // and can't access the right MethodHandles.Lookup instance, later on.
        // We do that here, just to statically prove it is working.
        MethodHandleInfo info = lookup.revealDirect(target);
        Class<?> receiver = info.getDeclaringClass();
        String methodName = info.getName();
        RuntimeAssertionError.assertTrue(methodName.startsWith(METHOD_PREFIX));

        return new InternalRunnable(receiver, methodName);
    }


    // AKI-131: These are only used for serialization support so they are REAL objects, not shadow ones.
    private Class<?> receiver;
    private String methodName;

    private Method target;

    private InternalRunnable(Class<?> receiver, String methodName) {
        // We call the hidden super-class so this doesn't update our hash code.
        super(null, null, 0);
        this.receiver = receiver;
        this.methodName = methodName;
        this.target = createAccessibleMethod(receiver, methodName);
    }

    // Deserializer support.
    public InternalRunnable(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(InternalRunnable.class, deserializer);

        // We write the class as a direct class object reference but the method name, inline.
        // Note that we can only store the class if it is a shadow class, so unwrap it.
        Object original = deserializer.readObject();
        String externalMethodName = CodecIdioms.deserializeString(deserializer);
        // (remember that the pre-pass always returns null).
        if (null != original) {
            Class<?> clazz = ((s.java.lang.Class<?>)original).getRealClass();
            // Note that the method name needs a prefix added.
            String methodName = METHOD_PREFIX + externalMethodName;

            this.receiver = clazz;
            this.methodName = methodName;
            this.target = createAccessibleMethod(clazz, methodName);
        }
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(InternalRunnable.class, serializer);

        // We save the receiver class as an object reference and the method name, inline.
        // Note that we can only store the class if it is a shadow class, so unwrap it.
        s.java.lang.Class<?> clazz = new s.java.lang.Class<>(this.receiver);
        // Note that we need to strip the prefix from the method.
        String methodName = this.methodName.substring(METHOD_PREFIX.length());

        serializer.writeObject(clazz);
        CodecIdioms.serializeString(serializer, methodName);
    }

    @Override
    public void avm_run() throws e.s.java.lang.Throwable {
        try {
            target.invoke(null);
        } catch (IllegalAccessException | IllegalArgumentException e) {
            // This would be a problem in our setup - an internal error.
            throw RuntimeAssertionError.unexpected(e);
        } catch (InvocationTargetException e) {
            // We need to unwrap this and re-throw it.
            Throwable cause = e.getCause();
            if (cause instanceof RuntimeException) {
                throw (RuntimeException) cause;
            } else if (cause instanceof e.s.java.lang.Throwable) {
                var se = IInstrumentation.attachedThreadInstrumentation
                        .get().unwrapThrowable(cause);
                if (se instanceof s.java.lang.RuntimeException) {
                    throw (e.s.java.lang.Throwable) cause;
                }
            }
            // Any failure below us shouldn't be anything other than RuntimeException.
            throw new UnknownFailureException(cause);
        }
    }


    private static Method createAccessibleMethod(Class<?> receiver, String methodName) {
        Method method = null;
        try {
            method = receiver.getDeclaredMethod(methodName);
        } catch (NoSuchMethodException  e) {
            // We always have direct access to the user code.
            throw RuntimeAssertionError.unexpected(e);
        }
        method.setAccessible(true);
        return method;
    }
}
