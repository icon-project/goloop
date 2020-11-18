/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package foundation.icon.ee.tooling.abi;

import foundation.icon.ee.struct.StructDB;
import foundation.icon.ee.types.Method;
import org.aion.avm.utilities.Utilities;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassVisitor;
import org.objectweb.asm.ClassWriter;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

public class ABICompilerClassVisitor extends ClassVisitor {
    private final List<ABICompilerMethodVisitor> methodVisitors = new ArrayList<>();
    private List<Method> callableInfo = new ArrayList<>();
    private final Map<String, byte[]> classMap;
    private final StructDB structDB;
    private final boolean stripLineNumber;

    public ABICompilerClassVisitor(ClassWriter cw, Map<String, byte[]> classMap,
            StructDB structDB, boolean stripLineNumber) {
        super(Opcodes.ASM7, cw);
        this.classMap = classMap;
        this.structDB = structDB;
        this.stripLineNumber = stripLineNumber;
    }

    public List<Method> getCallableInfo() {
        return callableInfo;
    }

    @Override
    public void visit(int version, int access, String name, String signature, String superName, String[] interfaces) {
        if (superName != null && !superName.equals("java/lang/Object")) {
            String superClassName = Utilities.internalNameToFullyQualifiedName(superName);
            byte[] classBytes = classMap.get(superClassName);
            if (classBytes == null) {
                throw new ABICompilerException("Cannot find super class: " + superName);
            }
            ClassReader reader = new ClassReader(classBytes);
            ClassWriter classWriter = new ClassWriter(ClassWriter.COMPUTE_MAXS);
            ABICompilerClassVisitor superVisitor = new ABICompilerClassVisitor(classWriter, classMap, structDB, stripLineNumber);
            reader.accept(superVisitor, 0);
            callableInfo = superVisitor.getCallableInfo();
            classBytes = classWriter.toByteArray();
            classMap.replace(superClassName, classBytes);
        }
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
                super.visitMethod(access, name, descriptor, signature, exceptions), structDB, stripLineNumber);
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
        Set<String> currentCallables = new HashSet<>();
        Set<String> currentEvents = new HashSet<>();
        Set<String> superCallables = callableInfo.stream()
                .filter(m -> m.getType() == Method.MethodType.FUNCTION || m.getType() == Method.MethodType.FALLBACK)
                .map(Method::getName)
                .collect(Collectors.toSet());
        Set<String> superEvents = callableInfo.stream()
                .filter(m -> m.getType() == Method.MethodType.EVENT)
                .map(Method::getName)
                .collect(Collectors.toSet());

        for (ABICompilerMethodVisitor mv : methodVisitors) {
            final String methodName = mv.getMethodName();
            if (mv.isExternal()) {
                if (currentCallables.contains(methodName)) {
                    throw new ABICompilerException("Multiple @External methods with the same name", methodName);
                }
                currentCallables.add(methodName);
                if (superCallables.contains(methodName)) {
                    Method mth = callableInfo.stream()
                            .filter(m -> m.getName().equals(methodName))
                            .findFirst().orElse(null);
                    if (mth != null && !mth.equals(mv.getCallableMethodInfo())) {
                        throw new ABICompilerException("Re-define a @External method with a different flag", methodName);
                    }
                    callableInfo.remove(mth);
                }
                callableInfo.add(mv.getCallableMethodInfo());
            } else if (mv.isOnInstall()) {
                if (foundOnInstall) {
                    throw new ABICompilerException("Multiple public <init> methods", methodName);
                }
                foundOnInstall = true;
                callableInfo.removeIf(m -> m.getName().equals(methodName));
                callableInfo.add(mv.getCallableMethodInfo());
            } else if (mv.isEventLog()) {
                if (currentEvents.contains(methodName)) {
                    throw new ABICompilerException("Multiple @EventLog methods with the same name", methodName);
                }
                currentEvents.add(methodName);
                if (superEvents.contains(methodName)) {
                    Method mth = callableInfo.stream()
                            .filter(m -> m.getName().equals(methodName))
                            .findFirst().orElse(null);
                    if (mth != null && mth.getIndexed() != mv.getCallableMethodInfo().getIndexed()) {
                        throw new ABICompilerException("Re-define a @EventLog method with a different indexed", methodName);
                    }
                    callableInfo.remove(mth);
                }
                callableInfo.add(mv.getCallableMethodInfo());
            } else if (mv.isFallback()) {
                if (mv.isPayable()) {
                    callableInfo.removeIf(m -> m.getName().equals(methodName));
                    callableInfo.add(mv.getCallableMethodInfo());
                } else if (superCallables.contains(methodName)) {
                    throw new ABICompilerException("Invalid fallback method re-definition", methodName);
                }
            } else {
                if (superCallables.contains(methodName) || superEvents.contains(methodName)) {
                    throw new ABICompilerException("Re-define a method without annotation", methodName);
                }
            }
        }
    }
}
