/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package p.score;

import java.lang.invoke.MethodHandle;
import java.lang.invoke.MethodHandleInfo;
import java.lang.invoke.MethodHandles;
import java.lang.invoke.MethodType;
import java.lang.reflect.Constructor;
import java.lang.reflect.Executable;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;

import i.*;
import s.java.lang.Float;
import s.java.lang.Integer;
import s.java.lang.Long;
import s.java.lang.Short;


public final class InternalFunction extends s.java.lang.Object implements s.java.util.function.Function {
    private static final String METHOD_PREFIX = "avm_";
    private static final String INIT_NAME = "<init>";

    public static InternalFunction createFunction(MethodHandles.Lookup lookup, MethodHandle target) {
        // Note that we need to convert this from a MethodHandle to a traditional reflection Method since we need to serialize it
        // and can't access the right MethodHandles.Lookup instance, later on.
        // We do that here, just to statically prove it is working.
        MethodHandleInfo info = lookup.revealDirect(target);
        Class<?> receiver = info.getDeclaringClass();
        String methodName = info.getName();
        MethodType type = info.getMethodType();
        Class<?> parameterType;

        if (info.getReferenceKind() == MethodHandleInfo.REF_invokeStatic || info.getReferenceKind() == MethodHandleInfo.REF_newInvokeSpecial) {
            parameterType = type.parameterType(0);
        } else if (info.getReferenceKind() == MethodHandleInfo.REF_invokeVirtual || info.getReferenceKind() == MethodHandleInfo.REF_invokeInterface) {
            parameterType = null;
        } else {
            throw RuntimeAssertionError.unimplemented("Unexpected MethodType " + info.getReferenceKind());
        }

        RuntimeAssertionError.assertTrue(methodName.startsWith(METHOD_PREFIX) || methodName.equals(INIT_NAME));

        return new InternalFunction(receiver, methodName, parameterType);
    }


    // AKI-131: These are only used for serialization support so they are REAL objects, not shadow ones.
    private Class<?> receiver;
    private String methodName;
    private Class<?> parameterType;

    private Executable target;

    private InternalFunction(Class<?> receiver, String methodName, Class<?> parameterType) {
        // We call the hidden super-class so this doesn't update our hash code.
        super(null, null, 0);
        this.receiver = receiver;
        this.methodName = methodName;
        this.parameterType = parameterType;
        this.target = createAccessibleMethod(receiver, methodName, parameterType);
    }

    // Deserializer support.
    public InternalFunction(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(InternalFunction.class, deserializer);

        // We write the classes as direct class objects reference but the method name, inline.
        // Note that we can only store the class if it is a shadow class, so unwrap it.
        Object originalReceiver = deserializer.readObject();
        String externalMethodName = CodecIdioms.deserializeString(deserializer);
        Object originalParameter = deserializer.readObject();
        // (remember that the pre-pass always returns null).
        if (null != originalReceiver) {
            Class<?> receiver = ((s.java.lang.Class<?>) originalReceiver).getRealClass();
            // Note that the method name needs a prefix added.
            String methodName;
            if(externalMethodName.equals(INIT_NAME)){
                methodName = externalMethodName;
            } else {
                methodName = METHOD_PREFIX + externalMethodName;
            }
            Class<?> parameterType = null;
            if(originalParameter != null) {
                parameterType = ((s.java.lang.Class<?>) originalParameter).getRealClass();
            }

            this.receiver = receiver;
            this.methodName = methodName;
            this.parameterType = parameterType;
            this.target = createAccessibleMethod(receiver, methodName, parameterType);
        }
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(InternalFunction.class, serializer);

        // We save the classes as object references and the method name, inline.
        // Note that we can only store the class if it is a shadow class, so unwrap it.
        s.java.lang.Class<?> receiverClass = new s.java.lang.Class<>(this.receiver);
        // Note that we need to strip the prefix from the method.
        String methodName;
        if(this.methodName.equals(INIT_NAME)){
            methodName = this.methodName;
        } else {
            methodName = this.methodName.substring(METHOD_PREFIX.length());
        }
        s.java.lang.Class<?> parameterClass = null;
        if(this.parameterType != null){
            parameterClass = getShadowCanonicalType(this.parameterType);
        }

        serializer.writeObject(receiverClass);
        CodecIdioms.serializeString(serializer, methodName);
        serializer.writeObject(parameterClass);
    }

    @Override
    public i.IObject avm_apply(i.IObject input) throws e.s.java.lang.Throwable {
        try {
            Object result;
            if (target instanceof Constructor) {
                result = ((Constructor<?>) target).newInstance(mapInputToParameterType(input, this.parameterType));
            } else {
                if (parameterType == null) {
                    // invokeVirtual and invokeInterface case
                    result = ((Method) target).invoke(input);
                } else {
                    // invokeStatic and newInvokeSpecial case
                    result= ((Method) target).invoke(null, mapInputToParameterType(input, this.parameterType));
                }
            }
            return mapBoxedType(result);
        } catch (IllegalAccessException | IllegalArgumentException | InstantiationException e) {
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
            throw RuntimeAssertionError.unexpected(cause);
        }
    }

    private static Executable createAccessibleMethod(Class<?> receiver, String methodName, Class<?> parameterType) {
        Executable method = null;
        try {
            if (methodName.equals(INIT_NAME)) {
                method = receiver.getConstructor(parameterType);
            } else if (parameterType == null) {
                // invokeVirtual and invokeInterface case
                method = receiver.getDeclaredMethod(methodName);
            } else {
                // invokeStatic case
                method = receiver.getDeclaredMethod(methodName, parameterType);
            }
        } catch (NoSuchMethodException e) {
            // We always have direct access to the user code.
            throw RuntimeAssertionError.unexpected(e);
        }
        method.setAccessible(true);
        return method;
    }

    private static s.java.lang.Object mapBoxedType(Object obj) {
        // the external version of valueOf is used since the user would be creating the shadow abject anyway
        s.java.lang.Object ret = null;
        if (null == obj) {
            ret = null;
        } else if (obj instanceof s.java.lang.Object) {
            ret = (s.java.lang.Object) obj;
        } else {
            Class<?> argClass = obj.getClass();
            if (argClass.equals(java.lang.Short.class)) {
                ret = Short.avm_valueOf(((java.lang.Short) obj));
            } else if (argClass.equals(java.lang.Integer.class)) {
                ret = Integer.avm_valueOf(((java.lang.Integer) obj));
            } else if (argClass.equals(java.lang.Long.class)) {
                ret = Long.avm_valueOf(((java.lang.Long) obj));
            } else if (argClass.equals(java.lang.Float.class)) {
                ret = Float.avm_valueOf(((java.lang.Float) obj));
            } else if (argClass.equals(java.lang.Double.class)) {
                ret = s.java.lang.Double.avm_valueOf(((java.lang.Double) obj));
            } else if (argClass.equals(java.lang.Boolean.class)) {
                ret = s.java.lang.Boolean.avm_valueOf(((java.lang.Boolean) obj));
            } else if (argClass.equals(java.lang.Byte.class)) {
                ret = s.java.lang.Byte.avm_valueOf(((java.lang.Byte) obj));
            } else if (argClass.equals(java.lang.Character.class)) {
                ret = s.java.lang.Character.avm_valueOf(((java.lang.Character) obj));
            }else {
                RuntimeAssertionError.unreachable("InternalFunction received an unexpected type " + argClass.getName());
            }
        }
        return ret;
    }

    private static java.lang.Object mapInputToParameterType(IObject input, Class<?> parameterType) {
        java.lang.Object ret = null;
        if (!parameterType.isPrimitive()) {
            ret = input;
        } else {
            if (parameterType.equals(short.class)) {
                ret = ((s.java.lang.Short) input).getUnderlying();
            } else if (parameterType.equals(int.class)) {
                ret = ((s.java.lang.Integer) input).getUnderlying();
            } else if (parameterType.equals(long.class)) {
                ret = ((s.java.lang.Long) input).getUnderlying();
            } else if (parameterType.equals(float.class)) {
                ret = ((s.java.lang.Float) input).getUnderlying();
            } else if (parameterType.equals(double.class)) {
                ret = ((s.java.lang.Double) input).getUnderlying();
            } else if (parameterType.equals(boolean.class)) {
                ret = ((s.java.lang.Boolean) input).getUnderlying();
            } else if (parameterType.equals(byte.class)) {
                ret = ((s.java.lang.Byte) input).getUnderlying();
            } else if (parameterType.equals(char.class)) {
                ret = ((s.java.lang.Character) input).getUnderlying();
            }else {
                RuntimeAssertionError.unreachable("InternalFunction received an unexpected type " + parameterType.getName());
            }
        }
        return ret;
    }

    private static s.java.lang.Class<?> getShadowCanonicalType(Class<?> parameterType) {
        s.java.lang.Class<?> ret = null;
        if(!parameterType.isPrimitive()){
            ret = new s.java.lang.Class<>(parameterType);
        } else {
            if (parameterType.equals(short.class)) {
                ret = s.java.lang.Short.avm_TYPE;
            } else if (parameterType.equals(int.class)) {
                ret = s.java.lang.Integer.avm_TYPE;
            } else if (parameterType.equals(long.class)) {
                ret = s.java.lang.Long.avm_TYPE;
            } else if (parameterType.equals(float.class)) {
                ret = s.java.lang.Float.avm_TYPE;
            } else if (parameterType.equals(double.class)) {
                ret = s.java.lang.Double.avm_TYPE;
            } else if (parameterType.equals(boolean.class)) {
                ret = s.java.lang.Boolean.avm_TYPE;
            } else if (parameterType.equals(byte.class)) {
                ret = s.java.lang.Byte.avm_TYPE;
            } else if (parameterType.equals(char.class)) {
                ret = s.java.lang.Character.avm_TYPE;
            } else {
                RuntimeAssertionError.unreachable("InternalFunction received an unexpected type " + parameterType.getName());
            }
        }
        return ret;
    }
}
