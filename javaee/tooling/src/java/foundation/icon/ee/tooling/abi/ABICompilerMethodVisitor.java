/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package foundation.icon.ee.tooling.abi;

import foundation.icon.ee.types.Method;
import org.objectweb.asm.AnnotationVisitor;
import org.objectweb.asm.Label;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;
import org.objectweb.asm.tree.MethodNode;
import org.objectweb.asm.util.ASMifier;
import org.objectweb.asm.util.TraceMethodVisitor;
import score.Address;

import java.io.PrintWriter;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Map;
import java.util.Set;

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
    private MethodVisitor pmv = null;
    private boolean stripLineNumber;

    private static final int MAX_INDEXED_COUNT = 3;
    private static final Set<String> reservedEventNames = Set.of(
            "ICXTransfer"
    );

    @SuppressWarnings("unchecked")
    private static final Map.Entry<String, Integer>[] dataTypeEntries = new Map.Entry[]{
            // allowed types for both param and return
            Map.entry("B", Method.DataType.INTEGER),
            Map.entry("C", Method.DataType.INTEGER),
            Map.entry("S", Method.DataType.INTEGER),
            Map.entry("I", Method.DataType.INTEGER),
            Map.entry("J", Method.DataType.INTEGER),
            Map.entry("Ljava/math/BigInteger;", Method.DataType.INTEGER),
            Map.entry("Ljava/lang/String;", Method.DataType.STRING),
            Map.entry("[B", Method.DataType.BYTES),
            Map.entry("Z", Method.DataType.BOOL),
            Map.entry("Lscore/Address;", Method.DataType.ADDRESS),
            // allowed types only for return
            Map.entry("V", Method.DataType.NONE),
            Map.entry("Ljava/util/List;", Method.DataType.LIST),
            Map.entry("Ljava/util/Map;", Method.DataType.DICT),
    };

    private static final Map<String, Integer> paramTypeMap = Map.ofEntries(
            Arrays.copyOfRange(dataTypeEntries, 0, 10)
    );

    private static final Map<String, Integer> returnTypeMap = Map.ofEntries(
            dataTypeEntries
    );

    private static final List<String> returnTypes = List.of(
            // allowed types for both param and return
            "Z",
            "C",
            "B",
            "S",
            "I",
            "J",
            Type.getDescriptor(BigInteger.class),
            Type.getDescriptor(String.class),
            Type.getDescriptor(Address.class),
            "[B",
            // allowed types only for return
            Type.getDescriptor(List.class),
            Type.getDescriptor(Map.class),
            "V"
    );

    private static final List<String> paramTypes = List.of(
            Arrays.copyOfRange(returnTypes.toArray(new String[0]), 0, 10)
    );

    private static boolean isAllowedParamType(Type type) {
        return paramTypes.contains(type.getDescriptor());
    }

    private static boolean isAllowedReturnType(Type type) {
        return returnTypes.contains(type.getDescriptor());
    }

    public ABICompilerMethodVisitor(int access, String methodName, String methodDescriptor, MethodVisitor mv, boolean stripLineNumber) {
        super(Opcodes.ASM7, mv);
        this.access = access;
        this.methodName = methodName;
        this.methodDescriptor = methodDescriptor;

        if (methodName.equals("<init>") && checkIfPublicAndNonStatic(access)) {
            if (Type.getReturnType(methodDescriptor) != Type.VOID_TYPE) {
                throw new ABICompilerException("<init> method must have void return type", methodName);
            }
            isOnInstall = true;
        } else if (methodName.equals("fallback") && checkIfPublicAndNonStatic(access)) {
            if (Type.getReturnType(methodDescriptor) != Type.VOID_TYPE) {
                throw new ABICompilerException("fallback method must have void return type", methodName);
            }
            if (Type.getArgumentTypes(methodDescriptor).length != 0) {
                throw new ABICompilerException("fallback method cannot take arguments", methodName);
            }
            isFallback = true;
        }
        this.stripLineNumber = stripLineNumber;
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
            if (!checkIfPublicAndNonStatic(this.access)) {
                throw new ABICompilerException("@External methods must be public and non-static", methodName);
            }
            checkArgumentsAndReturnType();
            flags |= Method.Flags.EXTERNAL;
            // to process readonly element
            return new AnnotationVisitor(Opcodes.ASM7) {
                @Override
                public void visit(String name, Object value) {
                    if ("readonly".equals(name) && Boolean.TRUE.equals(value)) {
                        flags |= Method.Flags.READONLY;
                    }
                }
            };
        } else if (Type.getType(descriptor).getClassName().equals(Payable.class.getName())) {
            if (!checkIfPublicAndNonStatic(this.access)) {
                throw new ABICompilerException("@Payable methods must be public and non-static", methodName);
            }
            flags |= Method.Flags.PAYABLE;
            return null;
        } else if (Type.getType(descriptor).getClassName().equals(EventLog.class.getName())) {
            boolean isStatic = (this.access & Opcodes.ACC_STATIC) != 0;
            if (isStatic) {
                throw new ABICompilerException("@EventLog methods must be non-static", methodName);
            }
            if (Type.getReturnType(methodDescriptor) != Type.VOID_TYPE) {
                throw new ABICompilerException("@EventLog methods must have void return type", methodName);
            }
            if (reservedEventNames.contains(methodName)) {
                throw new ABICompilerException("Reserved event log name", methodName);
            }
            var args = Type.getArgumentTypes(methodDescriptor);
            for (Type t : args) {
                if (!paramTypeMap.containsKey(t.getDescriptor())) {
                    throw new ABICompilerException("Bad argument type for @EventLog method", methodName);
                }
            }
            isEventLog = true;
            return new AnnotationVisitor(Opcodes.ASM7) {
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
    public void visitCode() {
        super.visitCode();
        if (isEventLog()) {
            pmv = mv;
            mv = null;
        }
    }

    private void emitSetValueArrayElementString(int index, String value) {
        super.visitInsn(Opcodes.DUP);
        if (index <= 5) {
            super.visitInsn(Opcodes.ICONST_0 + index);
        } else {
            super.visitIntInsn(Opcodes.BIPUSH, index);
        }
        super.visitLdcInsn(value);
        super.visitInsn(Opcodes.AASTORE);
    }

    private void emitSetValueArrayElementByArg(int index, Type argType, int argPos) {
        super.visitInsn(Opcodes.DUP);
        if (index <= 5) {
            super.visitInsn(Opcodes.ICONST_0 + index);
        } else {
            super.visitIntInsn(Opcodes.BIPUSH, index);
        }
        switch (argType.getSort()) {
        case Type.BYTE:
        case Type.SHORT:
        case Type.INT:
        case Type.CHAR:
        case Type.BOOLEAN:
            super.visitVarInsn(Opcodes.ILOAD, argPos);
            super.visitInsn(Opcodes.I2L);
            super.visitMethodInsn(Opcodes.INVOKESTATIC,
                    "java/math/BigInteger", "valueOf",
                    "(J)Ljava/math/BigInteger;", false);
            break;
        case Type.LONG:
            super.visitVarInsn(Opcodes.LLOAD, argPos);
            super.visitMethodInsn(Opcodes.INVOKESTATIC,
                    "java/math/BigInteger", "valueOf",
                    "(J)Ljava/math/BigInteger;", false);
            break;
        case Type.ARRAY:
        case Type.OBJECT:
            super.visitVarInsn(Opcodes.ALOAD, argPos);
            break;
        default:
            assert false : "bad param type "+argType+" for @EventLog";
        }
        super.visitInsn(Opcodes.AASTORE);
    }

    private static String getEventParamType(Type type) {
        switch (type.getSort()) {
        case Type.BYTE:
        case Type.SHORT:
        case Type.INT:
        case Type.CHAR:
        case Type.LONG:
            return "int";
        case Type.BOOLEAN:
            return "bool";
        case Type.ARRAY:
            if (type.getDescriptor().equals("[B")) {
                return "bytes";
            }
        case Type.OBJECT:
            if (type.getDescriptor().equals("Ljava/lang/String;")) {
                return "str";
            } else if (type.getDescriptor().equals("Ljava/math/BigInteger;")) {
                return "int";
            } else if (type.getDescriptor().equals("Lscore/Address;")) {
                return "Address";
            }
        default:
            assert false : "bad param type "+type+" for @EventLog";
        }
        return null;
    }

    private String getEventSignature(Type[] args) {
        StringBuilder sb = new StringBuilder();
        sb.append(methodName);
        sb.append("(");
        for (int i=0; i<args.length; i++) {
            if (i>0) {
                sb.append(",");
            }
            sb.append(getEventParamType(args[i]));
        }
        sb.append(")");
        return sb.toString();
    }

    private void emitEventLogBody(Type[] args, int argsSize) {
        int argPos = 1;
        // Object[] indexedArr = new Object[${indexed+1}];
        super.visitIntInsn(Opcodes.BIPUSH, indexed+1);
        super.visitTypeInsn(Opcodes.ANEWARRAY, "java/lang/Object");
        // indexedArr[0] = ${event signature};
        emitSetValueArrayElementString(0, getEventSignature(args));
        for (int i=0; i<indexed; i++) {
            // indexedArr[${i+1}] = ${args[i]};
            emitSetValueArrayElementByArg(i+1, args[i], argPos);
            argPos += args[i].getSize();
        }
        super.visitVarInsn(Opcodes.ASTORE, argsSize+1);

        // Object[] dataArr = new Object[${args.len-indexed}];
        super.visitIntInsn(Opcodes.BIPUSH, args.length-indexed);
        super.visitTypeInsn(Opcodes.ANEWARRAY, "java/lang/Object");
        for (int i=0; i<args.length-indexed; i++) {
            // dataArr[$i] = ${args[indexed+i]};
            emitSetValueArrayElementByArg(i, args[indexed+i], argPos);
            argPos += args[indexed+i].getSize();
        }
        super.visitVarInsn(Opcodes.ASTORE, argsSize+2);

        // Context.log(indexedArr, dataArr);
        super.visitVarInsn(Opcodes.ALOAD, argsSize+1);
        super.visitVarInsn(Opcodes.ALOAD, argsSize+2);
        super.visitMethodInsn(Opcodes.INVOKESTATIC, "score/Context", "logEvent", "([Ljava/lang/Object;[Ljava/lang/Object;)V", false);
        super.visitInsn(Opcodes.RETURN);
        super.visitMaxs(0, 0);
    }

    @Override
    public void visitEnd() {
        if (isOnInstall() && this.flags != 0) {
            throw new ABICompilerException("<init> method cannot be annotated", methodName);
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
                    "Method parameters size mismatch (must compile with '-parameters')", methodName);
        }
        if (pmv != null) {
            mv = pmv;
            pmv = null;
        }
        if (isEventLog()) {
            final boolean TRACE = false;
            MethodNode node;
            if (TRACE) {
                node = new MethodNode(Opcodes.ASM7);
                pmv = mv;
                mv = node;
            }
            var args = Type.getArgumentTypes(methodDescriptor);
            if (args.length >= Byte.MAX_VALUE) {
                throw new ABICompilerException("Too many args in @EventLog method", methodName);
            }
            var argsSize = (Type.getArgumentsAndReturnSizes(methodDescriptor)>>2)-1;
            emitEventLogBody(args, argsSize);

            if (TRACE) {
                var asmifier = new ASMifier();
                node.accept(new TraceMethodVisitor(asmifier));
                var pw = new PrintWriter(System.out);
                asmifier.print(pw);
                pw.flush();
                node.accept(pmv);
                mv = pmv;
                pmv = null;
            }
        }
        super.visitEnd();
    }

    private boolean checkIfPublicAndNonStatic(int access) {
        boolean isPublic = (access & Opcodes.ACC_PUBLIC) != 0;
        boolean isStatic = (access & Opcodes.ACC_STATIC) != 0;
        return isPublic && !isStatic;
    }

    private void checkArgumentsAndReturnType() {
        for (Type type : Type.getArgumentTypes(this.methodDescriptor)) {
            if (!isAllowedParamType(type)) {
                throw new ABICompilerException(
                    type.getClassName() + " is not an allowed parameter type", methodName);
            }
        }
        Type returnType = Type.getReturnType(methodDescriptor);
        if (!isAllowedReturnType(returnType)) {
            throw new ABICompilerException(
                returnType.getClassName() + " is not an allowed return type", methodName);
        }
    }

    public Method getCallableMethodInfo() {
        if (isExternal() || isOnInstall()) {
            Type type = Type.getReturnType(this.methodDescriptor);
            int output = getDataType(returnTypeMap, type);
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
            return Method.newFunction(methodName, flags, optionalCount, getMethodParameters(), output, type.getDescriptor());
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
        Method.Parameter[] params;
        if (types.length > 0) {
            params = new Method.Parameter[types.length];
            for (int i = 0; i < types.length; i++) {
                params[i] = new Method.Parameter(
                        paramNames.get(i),
                        types[i].getDescriptor(),
                        getDataType(paramTypeMap, types[i]),
                        optional != null && optional[i]);
            }
        } else {
            params = new Method.Parameter[0];
        }
        return params;
    }

    private int getDataType(Map<String, Integer> map, Type type) {
        Integer dataType = map.get(type.getDescriptor());
        if (dataType == null) {
            throw new ABICompilerException("Unsupported type: " + type.getDescriptor(), methodName);
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

    public void visitLineNumber(int line, Label start) {
        if (stripLineNumber) {
            return;
        }
        super.visitLineNumber(line, start);
    }
}
