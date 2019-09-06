package org.aion.avm.tooling.abi;

import org.objectweb.asm.AnnotationVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;

import java.util.StringJoiner;

public class ABICompilerMethodVisitor extends MethodVisitor {
    private int access;
    private String methodName;
    private String methodDescriptor;
    private boolean isCallable = false;
    private boolean isFallback = false;

    public boolean isCallable() {
        return isCallable;
    }

    // Should only be called on public static methods
    public String getPublicStaticMethodSignature() {
        String signature = "";

        StringJoiner arguments = new StringJoiner(", ");
        for (Type type : Type.getArgumentTypes(this.methodDescriptor)) {
            arguments.add(ABIUtils.shortenClassName(type.getClassName()));
        }
        String returnType = Type.getReturnType(this.methodDescriptor).getClassName();
        signature = ("public ")
                + ("static ")
                + ABIUtils.shortenClassName(returnType) + " "
                + this.methodName + "("
                + arguments.toString()
                + ")";
        return signature;
    }

    public ABICompilerMethodVisitor(int access, String methodName, String methodDescriptor, MethodVisitor mv) {
        super(Opcodes.ASM6, mv);
        this.access = access;
        this.methodName = methodName;
        this.methodDescriptor = methodDescriptor;
    }

    @Override
    public AnnotationVisitor visitAnnotation(String descriptor, boolean visible) {
        boolean isPublic = (this.access & Opcodes.ACC_PUBLIC) != 0;
        boolean isStatic = (this.access & Opcodes.ACC_STATIC) != 0;
        if(Type.getType(descriptor).getClassName().equals(Callable.class.getName())) {
            if (!isPublic) {
                throw new ABICompilerException("@Callable methods must be public", methodName);
            }
            if (!isStatic) {
                throw new ABICompilerException("@Callable methods must be static", methodName);
            }
            checkArgumentsAndReturnType();
            isCallable = true;
            return null;
        } else if (Type.getType(descriptor).getClassName().equals(Fallback.class.getName())) {
            if (!isStatic) {
                throw new ABICompilerException("Fallback function must be static", methodName);
            }
            if (Type.getReturnType(methodDescriptor) != Type.VOID_TYPE) {
                throw new ABICompilerException(
                    "Function annotated @Fallback must have void return type", methodName);
            }
            if (Type.getArgumentTypes(methodDescriptor).length != 0) {
                throw new ABICompilerException(
                    "Function annotated @Fallback cannot take arguments", methodName);
            }
            isFallback = true;
            return null;
        } else if (Type.getType(descriptor).getClassName().equals(Initializable.class.getName())) {
            throw new ABICompilerException(
                "Methods cannot be annotated @Initializable", methodName);
        } else {
            return super.visitAnnotation(descriptor, visible);
        }
    }

    private void checkArgumentsAndReturnType() {
        for (Type type : Type.getArgumentTypes(this.methodDescriptor)) {
            if(!ABIUtils.isAllowedType(type)) {
                throw new ABICompilerException(
                    type.getClassName() + " is not an allowed parameter type", methodName);
            }
        }
        Type returnType = Type.getReturnType(methodDescriptor);
        if(!ABIUtils.isAllowedType(returnType) && returnType != Type.VOID_TYPE) {
            throw new ABICompilerException(
                returnType.getClassName() + " is not an allowed return type", methodName);
        }
    }

    public boolean isFallback() {
        return isFallback;
    }

    public String getMethodName() {
        return methodName;
    }

    public String getDescriptor() {
        return methodDescriptor;
    }
}