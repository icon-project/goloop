package org.aion.avm.core.types;

import org.aion.avm.core.util.Helpers;
import i.RuntimeAssertionError;
import org.objectweb.asm.Attribute;
import org.objectweb.asm.ClassVisitor;
import org.objectweb.asm.Opcodes;

/**
 * A class visitor whose purpose is to produce an appropriate {@link ClassInformation} object to
 * represent the class it is made to visit.
 */
public class ClassInfoVisitor extends ClassVisitor {
    private ClassInformation classInfo;
    private boolean isRenamed;

    /**
     * Constructs a new class visitor.
     *
     * If {@code isRenamed == true} then we interpret all classes we visit as post-rename classes.
     * Otherwise, we consider them pre-rename classes.
     *
     * Note that this visitor does not perform any renaming at all. It simply creates pre- or post-
     * rename class info objects accordingly.
     */
    public ClassInfoVisitor(boolean isRenamed) {
        super(Opcodes.ASM6);
        this.isRenamed = isRenamed;
    }

    @Override
    public void visit(int version, int access, String name, String signature, String superName, String[] interfaces) {
        String parentQualifiedName = (superName == null) ? null : Helpers.internalNameToFulllyQualifiedName(superName);
        String[] interfaceQualifiedNames = toQualifiedNames(interfaces);
        boolean isInterface = Opcodes.ACC_INTERFACE == (access & Opcodes.ACC_INTERFACE);

        // Mark the class info correctly depending on whether or not this is renamed.
        if (this.isRenamed) {
            this.classInfo = ClassInformation.postRenameInfoFor(isInterface, Helpers.internalNameToFulllyQualifiedName(name), parentQualifiedName, interfaceQualifiedNames);
        } else {
            this.classInfo = ClassInformation.preRenameInfoFor(isInterface, Helpers.internalNameToFulllyQualifiedName(name), parentQualifiedName, interfaceQualifiedNames);
        }
    }

    @Override
    public void visitSource(String source, String debug) {
        super.visitSource(source, debug);
    }

    @Override
    public void visitAttribute(Attribute attribute) {
        super.visitAttribute(attribute);
    }


    public ClassInformation getClassInfo() {
        RuntimeAssertionError.assertTrue(this.classInfo != null);
        return this.classInfo;
    }

    /**
     * Returns the class names as qualified (dot) names if classNames is not null and not empty.
     *
     * Returns null if classNames is null or empty.
     */
    private static String[] toQualifiedNames(String[] classNames) {
        if (classNames == null) {
            return null;
        }

        String[] qualifiedNames = new String[classNames.length];
        for (int i = 0; i < classNames.length; i++) {
            qualifiedNames[i] = Helpers.internalNameToFulllyQualifiedName(classNames[i]);
        }
        return qualifiedNames;
    }
}
