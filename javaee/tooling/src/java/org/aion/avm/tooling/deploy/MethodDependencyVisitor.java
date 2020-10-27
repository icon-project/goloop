package org.aion.avm.tooling.deploy;

import org.objectweb.asm.Label;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;
import org.objectweb.asm.signature.SignatureReader;
import org.objectweb.asm.signature.SignatureVisitor;

public class MethodDependencyVisitor extends MethodVisitor {

    private final DependencyCollector dependencyCollector;
    private final SignatureVisitor signatureVisitor;
    private final boolean preserveDebugInfo;

    public MethodDependencyVisitor(MethodVisitor mv, SignatureVisitor signatureVisitor, DependencyCollector dependencyCollector, boolean preserveDebugInfo) {
        super(Opcodes.ASM7, mv);
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
    public void visitLdcInsn(Object value) {
        if (value instanceof Type) {
            Type t = (Type) value;
            String desc = null;
            if (t.getSort() == Type.OBJECT) {
                desc = t.getDescriptor();
            } else if (t.getSort() == Type.ARRAY) {
                desc = t.getElementType().getDescriptor();
            }
            if (desc != null) {
                dependencyCollector.addDescriptor(desc);
            }
        }
        super.visitLdcInsn(value);
    }

    @Override
    public void visitLineNumber(int line, Label start) {
        if (preserveDebugInfo)
            super.visitLineNumber(line, start);
    }
}
