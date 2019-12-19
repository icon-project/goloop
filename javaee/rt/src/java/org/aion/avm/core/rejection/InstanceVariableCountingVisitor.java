package org.aion.avm.core.rejection;

import org.aion.avm.core.ClassToolchain;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.Opcodes;


/**
 * Counts the number of instance variables declared in each class the user provided.
 * We use this information to enforce limits during DAppCreator.
 */
public class InstanceVariableCountingVisitor extends ClassToolchain.ToolChainClassVisitor {
    private final InstanceVariableCountManager manager;
    private String className;
    private String superClassName;
    private int instanceVariableCount;

    public InstanceVariableCountingVisitor(InstanceVariableCountManager manager) {
        super(Opcodes.ASM6);
        this.manager = manager;
    }

    @Override
    public void visit(int version, int access, String name, String signature, String superName, String[] interfaces) {
        // Just capture the meta-data.
        this.className = name;
        this.superClassName = superName;
        super.visit(version, access, name, signature, superName, interfaces);
    }

    @Override
    public FieldVisitor visitField(int access, String name, String descriptor, String signature, Object value) {
        // If this is an instance field, add it to our count.  We don't restrict static fields.
        if (Opcodes.ACC_STATIC != (Opcodes.ACC_STATIC & access)) {
            this.instanceVariableCount += 1;
        }
        return super.visitField(access, name, descriptor, signature, value);
    }

    @Override
    public void visitEnd() {
        // Just report back.
        this.manager.addCount(this.className, this.superClassName, this.instanceVariableCount);
        super.visitEnd();
    }

}
