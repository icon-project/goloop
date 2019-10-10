package org.aion.avm.tooling.deploy.eliminator;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.objectweb.asm.ClassVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;

public class ClassDependencyVisitor extends ClassVisitor {

    private final String classSlashName;
    private String superSlashName;
    private String[] interfaces;
    private final List<MethodDependencyVisitor> methodVisitors = new ArrayList<>();
    private final Map<String, MethodInfo> methodMap = new HashMap<>();
    private final List<MethodInfo> alwaysReachables = new ArrayList<>();
    private boolean isInterface;
    private boolean isAbstract;

    public ClassDependencyVisitor(String classSlashName) {
        super(Opcodes.ASM6);
        this.classSlashName = classSlashName;
    }

    @Override
    public void visit(int version, int access, String name, String signature, String superName,
        String[] interfaces) {
        this.superSlashName = superName;
        this.interfaces = interfaces;
        this.isInterface = (access & Opcodes.ACC_INTERFACE) != 0;
        this.isAbstract = (access & Opcodes.ACC_ABSTRACT) != 0;
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String descriptor, String signature,
        String[] exceptions) {
        MethodDependencyVisitor mv = new MethodDependencyVisitor(name, descriptor, access,
            super.visitMethod(access, name, descriptor, signature, exceptions));
        methodVisitors.add(mv);
        return mv;

    }

    // We populate our map after having visited all the methods
    @Override
    public void visitEnd() {
        for (MethodDependencyVisitor methodVisitor : methodVisitors) {
            MethodInfo methodInfo = new MethodInfo(methodVisitor.getMethodIdentifier(), methodVisitor.isStatic(), methodVisitor.getMethodsCalled());
            methodMap.put(methodVisitor.getMethodIdentifier(), methodInfo);
            if (isAlwaysReachable(methodInfo.methodIdentifier)) {
                methodInfo.isReachable = true;
                alwaysReachables.add(methodInfo);
            }
        }
        super.visitEnd();
    }

    // These are the methods we flag as "Always Reachable" because we believe they are the only ones that can be called
    // when userclasses escape out of usercode as "Object".
    // It might be safer to mark all methods in Object as always reachable.
    private boolean isAlwaysReachable(String name) {
        return name.equals("<clinit>()V")
            || name.equals("hashCode()I")
            || name.equals("toString()Ljava/lang/String;")
            || name.equals("equals(Ljava/lang/Object;)Z");
    }

    public String getClassSlashName() {
        return classSlashName;
    }

    public String getSuperSlashName() {
        return superSlashName;
    }

    public String[] getInterfaces() {
        return interfaces;
    }

    public Map<String, MethodInfo> getMethodMap() {
        return methodMap;
    }

    public List<MethodInfo> getAlwaysReachables() {
        return alwaysReachables;
    }

    public boolean isInterface() {
        return isInterface;
    }

    public boolean isAbstract() {
        return isAbstract;
    }
}
