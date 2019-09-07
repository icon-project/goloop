/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package foundation.icon.ee.tooling.abi;

import org.aion.avm.tooling.abi.ABIUtils;
import org.objectweb.asm.AnnotationVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;

import java.util.StringJoiner;

public class ABICompilerMethodVisitor extends MethodVisitor {
    private int access;
    private String methodName;
    private String methodDescriptor;
    private boolean isExternal = false;
    private boolean isOnInstall = false;
    private boolean isPayable = false;
    private boolean isEventLog = false;

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
        if (Type.getType(descriptor).getClassName().equals(External.class.getName())) {
            if (!isPublic) {
                throw new ABICompilerException("@External methods must be public", methodName);
            }
            if (!isStatic) {
                throw new ABICompilerException("@External methods must be static", methodName);
            }
            checkArgumentsAndReturnType();
            isExternal = true;
            // TODO: process readonly flag
            return null;
        } else if (Type.getType(descriptor).getClassName().equals(OnInstall.class.getName())) {
            if (!isPublic) {
                throw new ABICompilerException("@OnInstall methods must be public", methodName);
            }
            if (!isStatic) {
                throw new ABICompilerException("@OnInstall methods must be static", methodName);
            }
            isOnInstall = true;
            return null;
        } else if (Type.getType(descriptor).getClassName().equals(Payable.class.getName())) {
            if (!isPublic) {
                throw new ABICompilerException("@Payable methods must be public", methodName);
            }
            if (!isStatic) {
                throw new ABICompilerException("@Payable methods must be static", methodName);
            }
            if (Type.getReturnType(methodDescriptor) != Type.VOID_TYPE) {
                throw new ABICompilerException(
                    "Method annotated @Payable must have void return type", methodName);
            }
            if (Type.getArgumentTypes(methodDescriptor).length != 0) {
                throw new ABICompilerException(
                    "Method annotated @Payable cannot take arguments", methodName);
            }
            isPayable = true;
            return null;
        } else if (Type.getType(descriptor).getClassName().equals(EventLog.class.getName())) {
            if (!isStatic) {
                throw new ABICompilerException("@EventLog methods must be static", methodName);
            }
            isEventLog = true;
            return null;
        }
        return super.visitAnnotation(descriptor, visible);
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

    public boolean isExternal() {
        return isExternal;
    }

    public boolean isOnInstall() {
        return isOnInstall;
    }

    public boolean isPayable() {
        return isPayable;
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
