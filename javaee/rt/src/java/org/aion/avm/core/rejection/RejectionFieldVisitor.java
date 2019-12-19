package org.aion.avm.core.rejection;

import org.objectweb.asm.AnnotationVisitor;
import org.objectweb.asm.Attribute;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.TypePath;


/**
 * Does a simple read-only pass over the loaded class, ensuring it isn't doing anything it isn't allowed to do:
 * -uses bytecode in blacklist
 * -references class not in whitelist
 * -overrides methods which we will not support as the user may expect
 * 
 * When a violation is detected, throws the RejectedClassException.
 */
public class RejectionFieldVisitor extends FieldVisitor {
    public RejectionFieldVisitor(FieldVisitor visitor) {
        super(Opcodes.ASM6, visitor);
    }

    @Override
    public AnnotationVisitor visitAnnotation(String descriptor, boolean visible) {
        // Filter this.
        return new RejectionAnnotationVisitor();
    }

    @Override
    public AnnotationVisitor visitTypeAnnotation(int typeRef, TypePath typePath, String descriptor, boolean visible) {
        // Filter this.
        return new RejectionAnnotationVisitor();
    }

    @Override
    public void visitAttribute(Attribute attribute) {
        // "Non-standard attributes" are not supported, so filter them.
    }
}
