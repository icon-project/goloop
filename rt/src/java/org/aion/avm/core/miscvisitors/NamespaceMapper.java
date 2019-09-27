package org.aion.avm.core.miscvisitors;

import org.aion.avm.core.rejection.RejectedClassException;
import org.aion.avm.core.util.DebugNameResolver;
import org.aion.avm.core.util.DescriptorParser;
import i.PackageConstants;
import i.RuntimeAssertionError;
import org.objectweb.asm.Handle;
import org.objectweb.asm.Type;


/**
 * Given a pre-transformed class name or descriptor, this helper class converts it into its post-transformed appearance.
 * This primarily means mapping namespaces but also handling some special-cases, such as array wrapping.
 * 
 * Note that all methods operate on slash-style class names.
 * Failures to map a type will throw RejectedClassException.
 * 
 * Note that this was originally carved from UserClassMappingVisitor so some of its history can be found there.
 */
public class NamespaceMapper {
    private static final String FIELD_PREFIX = "avm_";
    private static final String METHOD_PREFIX = "avm_";

    private final PreRenameClassAccessRules preRenameClassAccessRules;
    private final String shadowPackageSlash;

    public NamespaceMapper(PreRenameClassAccessRules preRenameClassAccessRules, String shadowPackageSlash) {
        this.preRenameClassAccessRules = preRenameClassAccessRules;
        this.shadowPackageSlash = shadowPackageSlash;
    }

    public NamespaceMapper(PreRenameClassAccessRules preRenameClassAccessRules) {
        this.preRenameClassAccessRules = preRenameClassAccessRules;
        this.shadowPackageSlash = PackageConstants.kShadowSlashPrefix;
    }

    /**
     * @param name The pre-transform field name.
     * @return The post-transform field name.
     */
    public static String mapFieldName(String name) {
        return FIELD_PREFIX  + name;
    }

    /**
     * @param name The pre-transform method name.
     * @return The post-transform method name.
     */
    public static String mapMethodName(String name) {
        if ("<init>".equals(name) || "<clinit>".equals(name)) {
            return name;
        }

        return METHOD_PREFIX + name;
    }

    /**
     * @param type The pre-transform method type.
     * @return The post-transform method type.
     */
    public Type mapMethodType(Type type, boolean preserveDebuggability) {
        return Type.getMethodType(mapDescriptor(type.getDescriptor(), preserveDebuggability));
    }

    /**
     * @param methodHandle The pre-transform method handle.
     * @param mapMethodDescriptor True if the underlying descriptor should be mapped or false to leave it unchanged.
     * @return The post-transform method handle.
     */
    public Handle mapHandle(Handle methodHandle, boolean mapMethodDescriptor, boolean preserveDebuggability) {
        String methodOwner = methodHandle.getOwner();
        String methodName = methodHandle.getName();
        String methodDescriptor = methodHandle.getDesc();

        String newMethodOwner = mapType(methodOwner, preserveDebuggability);
        String newMethodName = mapMethodName(methodName);
        String newMethodDescriptor = mapMethodDescriptor ? mapDescriptor(methodDescriptor,  preserveDebuggability) : methodDescriptor;
        return new Handle(methodHandle.getTag(), newMethodOwner, newMethodName, newMethodDescriptor, methodHandle.isInterface());
    }

    /**
     * @param descriptor The pre-transform descriptor.
     * @return The post-transform descriptor.
     * @note This does not map array types in the descriptor.
     */
    public String mapDescriptor(String descriptor, boolean preserveDebuggability) {
        StringBuilder builder = DescriptorParser.parse(descriptor, new DescriptorParser.Callbacks<>() {
            @Override
            public StringBuilder readObject(int arrayDimensions, String type, StringBuilder userData) {
                writeArrayDimensions(userData, arrayDimensions);
                String newType = mapType(type, preserveDebuggability);
                userData.append(DescriptorParser.OBJECT_START);
                userData.append(newType);
                userData.append(DescriptorParser.OBJECT_END);
                return userData;
            }
            @Override
            public StringBuilder readBoolean(int arrayDimensions, StringBuilder userData) {
                writeArrayDimensions(userData, arrayDimensions);
                userData.append(DescriptorParser.BOOLEAN);
                return userData;
            }
            @Override
            public StringBuilder readShort(int arrayDimensions, StringBuilder userData) {
                writeArrayDimensions(userData, arrayDimensions);
                userData.append(DescriptorParser.SHORT);
                return userData;
            }
            @Override
            public StringBuilder readLong(int arrayDimensions, StringBuilder userData) {
                writeArrayDimensions(userData, arrayDimensions);
                userData.append(DescriptorParser.LONG);
                return userData;
            }
            @Override
            public StringBuilder readInteger(int arrayDimensions, StringBuilder userData) {
                writeArrayDimensions(userData, arrayDimensions);
                userData.append(DescriptorParser.INTEGER);
                return userData;
            }
            @Override
            public StringBuilder readFloat(int arrayDimensions, StringBuilder userData) {
                writeArrayDimensions(userData, arrayDimensions);
                userData.append(DescriptorParser.FLOAT);
                return userData;
            }
            @Override
            public StringBuilder readDouble(int arrayDimensions, StringBuilder userData) {
                writeArrayDimensions(userData, arrayDimensions);
                userData.append(DescriptorParser.DOUBLE);
                return userData;
            }
            @Override
            public StringBuilder readChar(int arrayDimensions, StringBuilder userData) {
                writeArrayDimensions(userData, arrayDimensions);
                userData.append(DescriptorParser.CHAR);
                return userData;
            }
            @Override
            public StringBuilder readByte(int arrayDimensions, StringBuilder userData) {
                writeArrayDimensions(userData, arrayDimensions);
                userData.append(DescriptorParser.BYTE);
                return userData;
            }
            @Override
            public StringBuilder argumentStart(StringBuilder userData) {
                userData.append(DescriptorParser.ARGS_START);
                return userData;
            }
            @Override
            public StringBuilder argumentEnd(StringBuilder userData) {
                userData.append(DescriptorParser.ARGS_END);
                return userData;
            }
            @Override
            public StringBuilder readVoid(StringBuilder userData) {
                userData.append(DescriptorParser.VOID);
                return userData;
            }
            private void writeArrayDimensions(StringBuilder builder, int dimensions) {
                for (int i = 0; i < dimensions; ++i) {
                    builder.append(DescriptorParser.ARRAY);
                }
            }
        }, new StringBuilder());
        
        return builder.toString();
    }

    /**
     * @param types The pre-transform types.
     * @return The post-transform types.
     */
    public String[] mapTypeArray(String[] types, boolean preserveDebuggability) {
        String[] newNames = null;
        if (null != types) {
            newNames = new String[types.length];
            for (int i = 0; i < types.length; ++i) {
                newNames[i] = mapType(types[i], preserveDebuggability);
            }
        }
        return newNames;
    }

    /**
     * @param type The pre-transform type name.
     * @param preserveDebuggability True if we cannot rename types.
     * @return The post-transform type name.
     */
    public String mapType(String type, boolean preserveDebuggability) {
        RuntimeAssertionError.assertTrue(-1 == type.indexOf("."));
        
        String newType = null;
        if (type.startsWith("[")){
            newType = mapDescriptor(type, preserveDebuggability);
        }else {
            if (this.preRenameClassAccessRules.isUserDefinedClassOrInterface(type)) {
                newType = DebugNameResolver.getUserPackageSlashPrefix(type, preserveDebuggability);
            } else if (this.preRenameClassAccessRules.isJclClass(type)) {
                newType =  shadowPackageSlash + type;
            } else if (this.preRenameClassAccessRules.isApiClass(type)) {
                newType =  PackageConstants.kShadowApiSlashPrefix + type;
            } else {
                // NOTE:  We probably want to make this into a private exception so that this helper can be an isolated utility.
                // We are currently throwing RejectedClassException, directly, since that was the original use in UserClassMappingVisitor.
                RejectedClassException.nonWhiteListedClass(type);
            }
        }
        return newType;
    }
}
