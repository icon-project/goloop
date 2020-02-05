/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package foundation.icon.ee.tooling.abi;

import foundation.icon.ee.types.Method;
import org.objectweb.asm.ClassVisitor;
import org.objectweb.asm.ClassWriter;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

public class ABICompilerClassVisitor extends ClassVisitor {
    private List<ABICompilerMethodVisitor> methodVisitors = new ArrayList<>();
    private List<ABICompilerMethodVisitor> callableMethodVisitors = new ArrayList<>();
    private List<Method> callableInfo = new ArrayList<>();
    private boolean stripLineNumber;

    public ABICompilerClassVisitor(ClassWriter cw, boolean stripLineNumber) {
        super(Opcodes.ASM6, cw);
        this.stripLineNumber = stripLineNumber;
    }

    public List<Method> getCallableInfo() {
        return callableInfo;
    }

    public List<ABICompilerMethodVisitor> getCallableMethodVisitors() {
        return callableMethodVisitors;
    }

    @Override
    public void visit(int version, int access, java.lang.String name, java.lang.String signature, java.lang.String superName, java.lang.String[] interfaces) {
        super.visit(version, access, name, signature, superName, interfaces);
    }

    @Override
    public FieldVisitor visitField(
            int access, String name, String descriptor, String signature, Object value) {
        return new ABICompilerFieldVisitor(access, name, descriptor,
                super.visitField(access, name, descriptor, signature, value));
    }

    @Override
    public MethodVisitor visitMethod(
            int access, String name, String descriptor, String signature, String[] exceptions) {
        if (name.equals("main") && ((access & Opcodes.ACC_PUBLIC) != 0)) {
            throw new ABICompilerException("main method cannot be defined", name);
        }
        ABICompilerMethodVisitor mv = new ABICompilerMethodVisitor(access, name, descriptor,
                super.visitMethod(access, name, descriptor, signature, exceptions), stripLineNumber);
        methodVisitors.add(mv);
        return mv;
    }

    @Override
    public void visitEnd() {
        postProcess();
        super.visitEnd();
    }

    private void postProcess() {
        boolean foundOnInstall = false;
        Set<String> callableNames = new HashSet<>();
        Set<String> eventsNames = new HashSet<>();
        for (ABICompilerMethodVisitor mv : methodVisitors) {
            if (mv.isExternal()) {
                if (callableNames.contains(mv.getMethodName())) {
                    throw new ABICompilerException("Multiple @External methods with the same name", mv.getMethodName());
                }
                callableNames.add(mv.getMethodName());
                callableInfo.add(mv.getCallableMethodInfo());
                callableMethodVisitors.add(mv);
            } else if (mv.isOnInstall()) {
                if (foundOnInstall) {
                    throw new ABICompilerException("Multiple onInstall methods", mv.getMethodName());
                }
                foundOnInstall = true;
                callableInfo.add(mv.getCallableMethodInfo());
                callableMethodVisitors.add(mv);
            } else if (mv.isEventLog()) {
                if (eventsNames.contains(mv.getMethodName())) {
                    throw new ABICompilerException("Multiple @EventLog methods with the same name", mv.getMethodName());
                }
                eventsNames.add(mv.getMethodName());
                callableInfo.add(mv.getCallableMethodInfo());
            } else if (mv.isFallback()) {
                callableInfo.add(mv.getCallableMethodInfo());
            }
        }
    }
}
