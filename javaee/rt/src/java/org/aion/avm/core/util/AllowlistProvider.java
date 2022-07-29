package org.aion.avm.core.util;

import i.PackageConstants;
import org.aion.avm.ArrayClassNameMapper;
import org.aion.avm.core.NodeEnvironment;
import org.aion.avm.utilities.Utilities;

import java.lang.reflect.Constructor;
import java.lang.reflect.Executable;
import java.lang.reflect.Method;
import java.lang.reflect.Modifier;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Comparator;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public class AllowlistProvider {
    /**
     * @return Map of supported Class objects to a list of supported methods.
     */
    public static Map<Class<?>, List<MethodDescriptor>> getClassLibraryMap() throws ClassNotFoundException {
        Map<Class<?>, List<MethodDescriptor>> classDeclaredMethodMap = new HashMap<>();
        List<Class<?>> shadowClasses = getCallableShadowClasses();

        for (Class<?> c : shadowClasses) {
            String associatedJclName = mapClassName(c.getName());
            Class<?> jclClass = Class.forName(associatedJclName);

            List<MethodDescriptor> declaredMethodList = Stream.of(c.getMethods(), c.getDeclaredConstructors())
                    .flatMap(Stream::of)
                    .filter(method -> isSupportedExecutable(method) && hasValidParamTypes(method))
                    .map(AllowlistProvider::generateMethodDescriptor)
                    .sorted(Comparator.comparing(m -> m.parameters))
                    .collect(Collectors.toList());
            classDeclaredMethodMap.put(jclClass, declaredMethodList);
        }
        return classDeclaredMethodMap;
    }

    private static List<Class<?>> getCallableShadowClasses() throws ClassNotFoundException {
        List<Class<?>> shadowClasses = new ArrayList<>();

        List<String> jclClassNames = NodeEnvironment.singleton.getJclSlashClassNames();
        jclClassNames.removeAll(MethodDescriptorCollector.getOmittedClassNames());
        jclClassNames.removeAll(Arrays.asList(
                "score/RevertedException",
                "score/UserRevertedException",
                "score/UserRevertException"
        ));
        jclClassNames.replaceAll(s -> PackageConstants.kShadowSlashPrefix + s);

        for (String className : jclClassNames) {
            shadowClasses.add(NodeEnvironment.singleton.loadSharedClass(Utilities.internalNameToFullyQualifiedName(className)));
        }
        return shadowClasses;
    }

    private static String mapClassName(String className) {
        if (isShadowClass(className)) {
            return className.substring(PackageConstants.kShadowDotPrefix.length());
        } else if (isArrayWrapperClass(className)) {
            return ArrayClassNameMapper.getOriginalNameFromWrapper(Utilities.fullyQualifiedNameToInternalName(className));
        } else if (isSupportedInternalType(className)) {
            return "java.lang.Object";
        } else {
            return className;
        }
    }

    private static boolean hasValidParamTypes(Executable method) {
        for (Class<?> c : method.getParameterTypes()) {
            if (!isShadowClass(c.getName()) &&
                    !isArrayWrapperClass(c.getName()) &&
                    !isPrimitive(c) &&
                    !isSupportedInternalType(c.getName())) {
                if (method instanceof Method) {
                    throw new AssertionError(
                            "Transformed method " + method.getDeclaringClass() + "." + method.getName()
                            + " should not have an unsupported parameter type: " + c.getName());
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
        return className.equals(PackageConstants.kInternalDotPrefix + "IObject") ||
                className.equals(PackageConstants.kInternalDotPrefix + "IObjectArray");
    }

    private static boolean isSupportedExecutable(Executable method) {
        if (method instanceof Constructor) {
            // Enum has protected modifier
            return Modifier.isPublic(method.getModifiers()) || Modifier.isProtected(method.getModifiers());
        } else
            return method.getName().startsWith("avm_");
    }

    private static boolean isPrimitive(Class<?> type) {
        if (type == null) {
            return false;
        }
        return type.isPrimitive();
    }

    private static MethodDescriptor generateMethodDescriptor(Executable method) {
        String methodName = mapMethodName(method);
        String descriptorString = buildDescriptorString(method);
        return new MethodDescriptor(methodName, descriptorString, Modifier.isStatic(method.getModifiers()));
    }

    private static String mapMethodName(Executable method) {
        if (method instanceof Constructor) {
            if (Modifier.isStatic(method.getModifiers())) {
                throw new AssertionError("Static constructor should not exist.");
            }
            return "<init>";
        } else {
            return method.getName().substring("avm_".length());
        }
    }

    private static String buildDescriptorString(Executable method) {
        Class<?>[] parameterTypes = method.getParameterTypes();
        StringBuilder builder = new StringBuilder();
        builder.append(DescriptorParser.ARGS_START);
        for (Class<?> param : parameterTypes) {
            writeClass(builder, param, method);
        }
        builder.append(DescriptorParser.ARGS_END);
        if (method instanceof Method) {
            writeClass(builder, ((Method) method).getReturnType(), method);
        } else {
            builder.append("V");
        }
        return builder.toString();
    }

    private static void writeClass(StringBuilder builder, Class<?> clazz, Executable method) {
        if (clazz.isArray()) {
            builder.append(DescriptorParser.ARRAY);
            writeClass(builder, clazz.getComponentType(), method);
        } else if (!clazz.isPrimitive()) {
            String className = clazz.getName();
            if (isArrayWrapperClass(className)) {
                builder.append(ArrayClassNameMapper.getOriginalNameFromWrapper(Utilities.fullyQualifiedNameToInternalName(className)));
            } else {
                String mappedClassName = mapClassName(className);
                if ((PackageConstants.kInternalDotPrefix + "IObjectArray").equals(className)) {
                    builder.append(DescriptorParser.ARRAY);
                    if ("s.java.util.Map".equals(method.getDeclaringClass().getName())
                            && "avm_ofEntries".equals(method.getName())) {
                        mappedClassName = "java.util.Map$Entry";
                    }
                }
                builder.append(DescriptorParser.OBJECT_START);
                builder.append(Utilities.fullyQualifiedNameToInternalName(mappedClassName));
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
            throw new AssertionError("Missing descriptor type: " + clazz);
        }
    }

    public static class MethodDescriptor {
        public final String name;
        public final String parameters;
        public final boolean isStatic;

        public MethodDescriptor(String name, String parameters, boolean isStatic) {
            this.name = name;
            this.parameters = parameters;
            this.isStatic = isStatic;
        }

        @Override
        public String toString() {
            return "MethodDescriptor{" +
                    "name='" + name + '\'' +
                    ", parameters=" + parameters +
                    ", isStatic=" + isStatic +
                    '}';
        }

        @Override
        public boolean equals(Object o) {
            if (this == o) return true;
            if (o == null || getClass() != o.getClass()) return false;
            MethodDescriptor that = (MethodDescriptor) o;
            return isStatic == that.isStatic &&
                    name.equals(that.name) &&
                    parameters.equals(that.parameters);
        }

        @Override
        public int hashCode() {
            int result = Objects.hash(name, isStatic);
            result = 31 * result + parameters.hashCode();
            return result;
        }
    }
}
