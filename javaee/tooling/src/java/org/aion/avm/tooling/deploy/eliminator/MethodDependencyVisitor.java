package org.aion.avm.tooling.deploy.eliminator;

import org.objectweb.asm.Handle;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;

import java.util.HashSet;
import java.util.Set;

public class MethodDependencyVisitor extends MethodVisitor {

    private final String methodIdentifier;
    private final Set<MethodInvocation> methodsCalled = new HashSet<>();
    private final boolean isStatic;

    public MethodDependencyVisitor(String methodName, String methodDescriptor, int access,
                                   MethodVisitor mv) {
        super(Opcodes.ASM7, mv);
        // the concatenation of name + descriptor is a unique identifier for every method in a class
        this.methodIdentifier = methodName + methodDescriptor;
        this.isStatic = (access & Opcodes.ACC_STATIC) != 0;
    }

    @Override
    public void visitMethodInsn(int opcode, String owner, String name, String desc, boolean itf) {
        methodsCalled.add(new MethodInvocation(owner, name + desc, opcode));
    }

    @Override
    public void visitInvokeDynamicInsn(String name, String desc, Handle bsm, Object... bsmArgs) {
        if (name.equals("apply") || name.equals("run")) {
            //TODO: Add tests to confirm assumptions like length >= 2, [1] is the Handle, etc.
            Handle ownerHandle = (Handle) bsmArgs[1];
            methodsCalled.add(new MethodInvocation(ownerHandle.getOwner(),
                    ownerHandle.getName() + ownerHandle.getDesc(), Opcodes.INVOKEDYNAMIC));
        } else if (!name.equals("makeConcatWithConstants")) {
            throw new UnsupportedOperationException("Unsupported invokedynamic instruction: " + name);
        }
    }

    public Set<MethodInvocation> getMethodsCalled() {
        return methodsCalled;
    }

    public String getMethodIdentifier() {
        return methodIdentifier;
    }

    public boolean isStatic() {
        return isStatic;
    }
}
