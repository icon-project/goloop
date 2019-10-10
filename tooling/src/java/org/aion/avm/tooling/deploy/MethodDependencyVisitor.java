package org.aion.avm.tooling.deploy;

import org.objectweb.asm.*;
import org.objectweb.asm.signature.SignatureReader;
import org.objectweb.asm.signature.SignatureVisitor;

public class MethodDependencyVisitor extends MethodVisitor {

    private final DependencyCollector dependencyCollector;
    private final SignatureVisitor signatureVisitor;
    private boolean preserveDebugInfo;

    public MethodDependencyVisitor(MethodVisitor mv, SignatureVisitor signatureVisitor, DependencyCollector dependencyCollector, boolean preserveDebugInfo) {
        super(Opcodes.ASM6, mv);
        this.dependencyCollector = dependencyCollector;
        this.signatureVisitor = signatureVisitor;
        this.preserveDebugInfo = preserveDebugInfo;
    }

    @Override
    public void visitTypeInsn(int opcode, String type) {
        //A type instruction is an instruction that takes the internal name of a class as parameter.
        dependencyCollector.addType(type);
        super.visitTypeInsn(opcode, type);
    }

    @Override
    public void visitFieldInsn(int opcode, String owner, String name, String descriptor) {
        // Assumes the type of the field will be visited once the owner class is visited
        dependencyCollector.addType(owner);
        super.visitFieldInsn(opcode, owner, name, descriptor);
    }

    @Override
    public void visitMethodInsn(int opcode, String owner, String name, String descriptor, boolean isInterface) {
        // Assumes the desc of the method will be visited once the owner class is visited
        dependencyCollector.addType(owner);
        super.visitMethodInsn(opcode, owner, name, descriptor, isInterface);
    }

    @Override
    public void visitTryCatchBlock(Label start, Label end, Label handler, String type) {
        dependencyCollector.addType(type);
        super.visitTryCatchBlock(start, end, handler, type);
    }

    @Override
    public void visitMultiANewArrayInsn(String descriptor, int numDimensions) {
        dependencyCollector.addDescriptor(descriptor);
        super.visitMultiANewArrayInsn(descriptor, numDimensions);
    }

    @Override
    public void visitLocalVariable(String name, String descriptor, String signature, Label start, Label end, int index) {
        if (signature == null) {
            dependencyCollector.addDescriptor(descriptor);
        } else {
            new SignatureReader(signature).acceptType(signatureVisitor);
        }

        if (preserveDebugInfo)
            super.visitLocalVariable(name, descriptor, signature, start, end, index);
    }

    @Override
    public void visitLineNumber(int line, Label start) {
        if (preserveDebugInfo)
            super.visitLineNumber(line, start);
    }
}
