package org.aion.avm.core.arraywrapping;

import org.aion.avm.core.ClassToolchain;
import org.aion.avm.core.types.ClassHierarchy;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;


/**
 * This is just the class visitor which passes individual methods to {@link org.aion.avm.core.arraywrapping.ArraysRequiringAnalysisMethodNode}.
 */
public class ArraysRequiringAnalysisClassVisitor extends ClassToolchain.ToolChainClassVisitor {
    private final ClassHierarchy hierarchy;

    public String className;

    public ArraysRequiringAnalysisClassVisitor(ClassHierarchy hierarchy) {
        super(Opcodes.ASM6);
        this.hierarchy = hierarchy;
    }

    @Override
    public void visit(int version, int access, String name, String signature, String superName, String[] interfaces) {
        className = name;
        super.visit(version, access, name, signature, superName, interfaces);
    }

    public MethodVisitor visitMethod(
            final int access,
            final String name,
            final String descriptor,
            final String signature,
            final String[] exceptions) {

        MethodVisitor mv = super.visitMethod(access, name, descriptor, signature, exceptions);

        return new ArraysRequiringAnalysisMethodNode(access, name, descriptor, signature, exceptions, mv, className, this.hierarchy);
    }
}
