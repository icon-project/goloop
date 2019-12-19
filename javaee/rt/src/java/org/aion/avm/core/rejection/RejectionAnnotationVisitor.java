package org.aion.avm.core.rejection;

import org.objectweb.asm.AnnotationVisitor;
import org.objectweb.asm.Opcodes;


/**
 * Filters out annotations since we don't use them.
 */
public class RejectionAnnotationVisitor extends AnnotationVisitor {
    public RejectionAnnotationVisitor() {
        super(Opcodes.ASM6);
    }

    @Override
    public void visit(String name, Object value) {
        // Filter.
    }

    @Override
    public void visitEnum(String name, String descriptor, String value) {
        // Filter.
    }

    @Override
    public AnnotationVisitor visitAnnotation(String name, String descriptor) {
        // Filter with this, since we are stateless.
        return this;
    }

    @Override
    public AnnotationVisitor visitArray(String name) {
        // Filter with this, since we are stateless.
        return this;
    }
}
