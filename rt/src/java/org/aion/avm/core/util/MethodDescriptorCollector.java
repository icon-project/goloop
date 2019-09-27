package org.aion.avm.core.util;

import i.PackageConstants;
import i.RuntimeAssertionError;
import org.aion.avm.core.arraywrapping.ArrayNameMapper;
import org.aion.avm.core.classloading.AvmSharedClassLoader;

import java.lang.reflect.Constructor;
import java.lang.reflect.Executable;
import java.lang.reflect.Method;
import java.lang.reflect.Modifier;
import java.util.*;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public class MethodDescriptorCollector {

    // this classes are omitted since no one can reference them
    private static List<String> omittedClassNames = new ArrayList<>(Arrays.asList(
            "java/lang/invoke/MethodHandles",
            "java/lang/invoke/MethodHandle",
            "java/lang/invoke/MethodType",
            "java/lang/invoke/CallSite",
            "java/lang/invoke/MethodHandles$Lookup",
            "java/lang/invoke/LambdaMetafactory",
            "java/lang/invoke/StringConcatFactory"));


    public static Map<String, List<String>> getClassNameMethodDescriptorMap(List<String> jclClassList, AvmSharedClassLoader classLoader) throws ClassNotFoundException {
        Map<String, List<String>> whitelistClassMethodMap = new HashMap<>();
        jclClassList.removeAll(omittedClassNames);

        //only expecting shadow classes
        jclClassList.replaceAll(s -> PackageConstants.kShadowSlashPrefix + s);

        for (String className : jclClassList) {
            Class<?> c = classLoader.loadClass(Helpers.internalNameToFulllyQualifiedName(className), true);

            // constructor methods
            // note that private constructors are not rejected at this level
            List<String> methodList = Stream.of(c.getDeclaredConstructors())
                    .filter(MethodDescriptorCollector::hasValidParameterType)
                    .map(MethodDescriptorCollector::buildMethodNameDescriptorString)
                    .collect(Collectors.toList());

            // public methods
            methodList.addAll(Stream.of(c.getMethods())
                    .filter(method -> hasAvmMethodPrefix(method) && hasValidParameterType(method))
                    .map(MethodDescriptorCollector::buildMethodNameDescriptorString)
                    .collect(Collectors.toList()));

            whitelistClassMethodMap.put(c.getName(), Collections.unmodifiableList(methodList));
        }
        return whitelistClassMethodMap;
    }

    public static String buildMethodNameDescriptorString(String methodName, String methodDescriptor) {
        // remove the return type from descriptor
        String methodDescriptorWithoutReturnType = methodDescriptor.substring(0, methodDescriptor.lastIndexOf(')') + 1);
        return buildMethodDescriptorMapValue(methodName, methodDescriptorWithoutReturnType);
    }

    public static String buildMethodNameDescriptorString(Executable method) {
        String methodName = getMethodName(method);
        String descriptorString = buildDescriptorStringForParameters(method.getParameterTypes());
        return buildMethodDescriptorMapValue(methodName, descriptorString);
    }

    private static String buildDescriptorStringForParameters(Class<?>[] parameterTypes) {
        StringBuilder builder = new StringBuilder();
        builder.append(DescriptorParser.ARGS_START);
        for (Class<?> param : parameterTypes) {
            writeClass(builder, param);
        }
        builder.append(DescriptorParser.ARGS_END);
        return builder.toString();
    }

    private static String buildMethodDescriptorMapValue(String methodName, String methodDescriptorWithoutReturnType) {
        return methodName + methodDescriptorWithoutReturnType;
    }

    private static String getMethodName(Executable method) {
        if (method instanceof Constructor) {
            RuntimeAssertionError.assertTrue(!Modifier.isStatic(method.getModifiers()));
            return "<init>";
        } else {
            return method.getName();
        }
    }

    private static boolean hasValidParameterType(Executable method) {
        for (Class<?> c : method.getParameterTypes()) {
            if (!isShadowClass(c.getName()) &&
                    !isArrayWrapperClass(c.getName()) &&
                    !isPrimitive(c) &&
                    !isSupportedInternalType(c.getName())) {
                if (method instanceof Method) {
                    throw RuntimeAssertionError.unreachable("Transformed method " + method.getDeclaringClass() + "." + method.getName() + " should not have an unsupported parameter type: " + c.getName());
                }
                return false;
            }
        }
        return true;
    }

    private static boolean isShadowClass(String className) {
        return className.startsWith(PackageConstants.kShadowDotPrefix);
    }

    private static boolean isArrayWrapperClass(String className) {
        return className.startsWith(PackageConstants.kArrayWrapperDotPrefix);
    }

    private static boolean isSupportedInternalType(String className) {
        return className.equals(PackageConstants.kInternalDotPrefix + "IObject");
    }

    private static boolean hasAvmMethodPrefix(Method method) {
        return method.getName().startsWith("avm_");
    }

    private static boolean isPrimitive(Class<?> type) {
        if (type == null) {
            return false;
        }
        return type.isPrimitive();
    }

    private static void writeClass(StringBuilder builder, Class<?> clazz) {
        if (clazz.isArray()) {
            builder.append(DescriptorParser.ARRAY);
            writeClass(builder, clazz.getComponentType());
        } else if (!clazz.isPrimitive()) {
            String className = clazz.getName();
            if (className.startsWith(PackageConstants.kArrayWrapperDotPrefix)) {
                builder.append(ArrayNameMapper.getOriginalNameOf(Helpers.fulllyQualifiedNameToInternalName(className)));
            } else if ((PackageConstants.kInternalDotPrefix + "IObject").equals(className)) {
                // Explicitly map IObject to shadow Object, since this method is only building the descriptor for shadow class method parameter types.
                builder.append(DescriptorParser.OBJECT_START);
                builder.append(PackageConstants.kShadowSlashPrefix + "java/lang/Object");
                builder.append(DescriptorParser.OBJECT_END);
            } else {
                builder.append(DescriptorParser.OBJECT_START);
                builder.append(Helpers.fulllyQualifiedNameToInternalName(className));
                builder.append(DescriptorParser.OBJECT_END);
            }
        } else if (Byte.TYPE == clazz) {
            builder.append(DescriptorParser.BYTE);
        } else if (Character.TYPE == clazz) {
            builder.append(DescriptorParser.CHAR);
        } else if (Double.TYPE == clazz) {
            builder.append(DescriptorParser.DOUBLE);
        } else if (Float.TYPE == clazz) {
            builder.append(DescriptorParser.FLOAT);
        } else if (Integer.TYPE == clazz) {
            builder.append(DescriptorParser.INTEGER);
        } else if (Long.TYPE == clazz) {
            builder.append(DescriptorParser.LONG);
        } else if (Short.TYPE == clazz) {
            builder.append(DescriptorParser.SHORT);
        } else if (Boolean.TYPE == clazz) {
            builder.append(DescriptorParser.BOOLEAN);
        } else if (Void.TYPE == clazz) {
            builder.append(DescriptorParser.VOID);
        } else {
            // This means we haven't implemented something.
            RuntimeAssertionError.unreachable("Missing descriptor type: " + clazz);
        }
    }
}
