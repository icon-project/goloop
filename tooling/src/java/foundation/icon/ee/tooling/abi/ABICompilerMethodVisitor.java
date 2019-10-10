/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package foundation.icon.ee.tooling.abi;

import foundation.icon.ee.types.Method;
import org.aion.avm.tooling.abi.ABIUtils;
import org.objectweb.asm.AnnotationVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.StringJoiner;

public class ABICompilerMethodVisitor extends MethodVisitor {
    private int access;
    private String methodName;
    private String methodDescriptor;
    private List<String> paramNames = new ArrayList<>();
    private boolean[] optional;
    private int flags;
    private int indexed;
    private boolean isOnInstall = false;
    private boolean isFallback = false;
    private boolean isEventLog = false;

    private static final int MAX_INDEXED_COUNT = 3;
    private static final Set<String> reservedEventNames = Set.of(
            "ICXTransfer"
    );
    private static final Map<String, Integer> dataTypeMap = Map.of(
            "B", Method.DataType.INTEGER,
            "C", Method.DataType.INTEGER,
            "S", Method.DataType.INTEGER,
            "I", Method.DataType.INTEGER,
            "J", Method.DataType.INTEGER,
            "Ljava/math/BigInteger;", Method.DataType.INTEGER,
            "Ljava/lang/String;", Method.DataType.STRING,
            "[B", Method.DataType.BYTES,
            "Z", Method.DataType.BOOL,
            "Lavm/Address;", Method.DataType.ADDRESS
    );

    public ABICompilerMethodVisitor(int access, String methodName, String methodDescriptor, MethodVisitor mv) {
        super(Opcodes.ASM6, mv);
        this.access = access;
        this.methodName = methodName;
        this.methodDescriptor = methodDescriptor;

        if (methodName.equals("onInstall") && checkIfPublicAndStatic(access)) {
            if (Type.getReturnType(methodDescriptor) != Type.VOID_TYPE) {
                throw new ABICompilerException("onInstall method must have void return type", methodName);
            }
            isOnInstall = true;
        } else if (methodName.equals("fallback") && checkIfPublicAndStatic(access)) {
            if (Type.getReturnType(methodDescriptor) != Type.VOID_TYPE) {
                throw new ABICompilerException("fallback method must have void return type", methodName);
            }
            if (Type.getArgumentTypes(methodDescriptor).length != 0) {
                throw new ABICompilerException("fallback method cannot take arguments", methodName);
            }
            isFallback = true;
        }
    }

    @Override
    public void visitParameter(String name, int access) {
        if (access == 0) {
            paramNames.add(name);
        }
    }

    @Override
    public AnnotationVisitor visitAnnotation(String descriptor, boolean visible) {
        if (Type.getType(descriptor).getClassName().equals(External.class.getName())) {
            if (!checkIfPublicAndStatic(this.access)) {
                throw new ABICompilerException("@External methods must be public and static", methodName);
            }
            checkArgumentsAndReturnType();
            flags |= Method.Flags.EXTERNAL;
            // to process readonly element
            return new AnnotationVisitor(Opcodes.ASM6) {
                @Override
                public void visit(String name, Object value) {
                    if ("readonly".equals(name) && Boolean.TRUE.equals(value)) {
                        flags |= Method.Flags.READONLY;
                    }
                }
            };
        } else if (Type.getType(descriptor).getClassName().equals(Payable.class.getName())) {
            if (!checkIfPublicAndStatic(this.access)) {
                throw new ABICompilerException("@Payable methods must be public and static", methodName);
            }
            flags |= Method.Flags.PAYABLE;
            return null;
        } else if (Type.getType(descriptor).getClassName().equals(EventLog.class.getName())) {
            boolean isStatic = (this.access & Opcodes.ACC_STATIC) != 0;
            if (!isStatic) {
                throw new ABICompilerException("@EventLog methods must be static", methodName);
            }
            if (Type.getReturnType(methodDescriptor) != Type.VOID_TYPE) {
                throw new ABICompilerException("@EventLog methods must have void return type", methodName);
            }
            if (reservedEventNames.contains(methodName)) {
                throw new ABICompilerException("Reserved event log name", methodName);
            }
            isEventLog = true;
            return new AnnotationVisitor(Opcodes.ASM6) {
                @Override
                public void visit(String name, Object value) {
                    if ("indexed".equals(name) && (value instanceof Integer)) {
                        indexed = (int) value;
                    }
                }
            };
        }
        return super.visitAnnotation(descriptor, visible);
    }

    @Override
    public void visitAnnotableParameterCount(int parameterCount, boolean visible) {
        optional = new boolean[parameterCount];
    }

    @Override
    public AnnotationVisitor visitParameterAnnotation(int parameter, String descriptor, boolean visible) {
        if (Type.getType(descriptor).getClassName().equals(Optional.class.getName())) {
            optional[parameter] = true;
        }
        return null;
    }

    @Override
    public void visitEnd() {
        if (isOnInstall() && this.flags != 0) {
            throw new ABICompilerException("onInstall method cannot be annotated", methodName);
        }
        if (isPayable() && isReadonly()) {
            throw new ABICompilerException("Method annotated @Payable cannot be readonly", methodName);
        }
        if (isEventLog() && this.flags != 0) {
            throw new ABICompilerException("Method annotated @EventLog cannot have other annotations", methodName);
        }
        if ((isOnInstall() || isExternal() || isEventLog()) &&
                paramNames.size() != Type.getArgumentTypes(methodDescriptor).length) {
            throw new ABICompilerException(
                    "Method parameters size mismatch (must compile with \'-parameters\')", methodName);
        }
        super.visitEnd();
    }

    private boolean checkIfPublicAndStatic(int access) {
        boolean isPublic = (access & Opcodes.ACC_PUBLIC) != 0;
        boolean isStatic = (access & Opcodes.ACC_STATIC) != 0;
        return isPublic && isStatic;
    }

    // TODO: refactor later
    private void checkArgumentsAndReturnType() {
        for (Type type : Type.getArgumentTypes(this.methodDescriptor)) {
            if (!ABIUtils.isAllowedType(type)) {
                throw new ABICompilerException(
                    type.getClassName() + " is not an allowed parameter type", methodName);
            }
        }
        Type returnType = Type.getReturnType(methodDescriptor);
        if (!ABIUtils.isAllowedType(returnType) && returnType != Type.VOID_TYPE) {
            throw new ABICompilerException(
                returnType.getClassName() + " is not an allowed return type", methodName);
        }
    }

    public String getPublicStaticMethodSignature() {
        StringJoiner arguments = new StringJoiner(", ");
        for (Type type : Type.getArgumentTypes(this.methodDescriptor)) {
            arguments.add(ABIUtils.shortenClassName(type.getClassName()));
        }
        String returnType = Type.getReturnType(this.methodDescriptor).getClassName();
        return ("public ")
                + ("static ")
                + ABIUtils.shortenClassName(returnType) + " "
                + this.methodName + "("
                + arguments.toString()
                + ")";
    }

    public Method getCallableMethodInfo() {
        if (isExternal() || isOnInstall()) {
            Type type = Type.getReturnType(this.methodDescriptor);
            int output = Method.DataType.NONE;
            if (type != Type.VOID_TYPE) {
                output = getDataType(type);
            }
            int optionalCount = 0;
            if (optional != null) {
                for (int i = optional.length - 1; i >= 0; i--) {
                    if (optional[i]) {
                        if (i < optional.length - 1 && !optional[i + 1]) {
                            throw new ABICompilerException("Non-optional parameter follows @Optional parameter", methodName);
                        }
                        optionalCount++;
                    }
                }
            }
            return Method.newFunction(methodName, flags, optionalCount, getMethodParameters(), output);
        }
        if (isFallback() && isPayable()) {
            return Method.newFallback();
        }
        if (isEventLog()) {
            Method.Parameter[] params = getMethodParameters();
            if (indexed < 0 || indexed > params.length || indexed > MAX_INDEXED_COUNT) {
                throw new ABICompilerException("Invalid indexed count=" + indexed, methodName);
            }
            return Method.newEvent(methodName, indexed, params);
        }
        return null;
    }

    private Method.Parameter[] getMethodParameters() {
        Type[] types = Type.getArgumentTypes(this.methodDescriptor);
        Method.Parameter[] params = null;
        if (types.length > 0) {
            params = new Method.Parameter[types.length];
            for (int i = 0; i < types.length; i++) {
                params[i] = new Method.Parameter(
                        paramNames.get(i), getDataType(types[i]),
                        optional != null && optional[i]);
            }
        }
        return params;
    }

    private int getDataType(Type type) {
        int dataType = dataTypeMap.getOrDefault(type.getDescriptor(), Method.DataType.NONE);
        if (dataType == Method.DataType.NONE) {
            throw new ABICompilerException("Unsupported parameter type: " + type.getDescriptor(), methodName);
        }
        return dataType;
    }

    public boolean isExternal() {
        return (this.flags & Method.Flags.EXTERNAL) != 0;
    }

    public boolean isReadonly() {
        return (this.flags & Method.Flags.READONLY) != 0;
    }

    public boolean isPayable() {
        return (this.flags & Method.Flags.PAYABLE) != 0;
    }

    public boolean isOnInstall() {
        return isOnInstall;
    }

    public boolean isFallback() {
        return isFallback;
    }

    public boolean isEventLog() {
        return isEventLog;
    }

    public String getMethodName() {
        return methodName;
    }

    public String getDescriptor() {
        return methodDescriptor;
    }
}
