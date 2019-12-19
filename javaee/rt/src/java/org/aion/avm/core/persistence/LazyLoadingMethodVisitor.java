package org.aion.avm.core.persistence;

import org.aion.avm.core.util.DescriptorParser;
import org.aion.avm.utilities.Utilities;

import i.RuntimeAssertionError;
import org.objectweb.asm.Handle;
import org.objectweb.asm.Label;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;


/**
 * Walks the method code, replace prepending a call to "lazyLoad()" on any GETFIELD/PUTFIELD bytecodes.
 * Note that there are special-cases:
 * -"&lt;clinit&gt;" - no re-writing is done here since nothing visible at this point could be a stub (this
 *  visitor isn't created in those cases).
 * -"&lt;init&gt;" - extra analysis is done to determine where the "this" pointer is since we can't call
 *  lazyLoad() on it (fields can be accessed before "this" has been initialized).
 * 
 * It may be possible to expand this control flow analysis to also avoid redundant lazyLoad() calls in all
 * methods, but this is a later consideration.
 */
public class LazyLoadingMethodVisitor extends MethodVisitor {
    private static final String SHADOW_OBJECT_NAME = Utilities.fulllyQualifiedNameToInternalName(s.java.lang.Object.class.getName());
    private static final String LAZY_LOAD_NAME = "lazyLoad";
    private static final String LAZY_LOAD_DESCRIPTOR = "()V";

    private final StackThisTracker tracker;
    // The offset of the next instruction into the canSafelySkip array.  Usually, this is just bytecodes but labels, frames, and line number entries
    // are also counted.
    private int frameOffset;

    public LazyLoadingMethodVisitor(MethodVisitor visitor, StackThisTracker tracker) {
        super(Opcodes.ASM6, visitor);
        this.tracker = tracker;
    }

    @Override
    public void visitFieldInsn(int opcode, String owner, String name, String descriptor) {
        checkInjectLazyLoad(opcode, descriptor);
        super.visitFieldInsn(opcode, owner, name, descriptor);
        this.frameOffset += 1;
    }

    @Override
    public void visitInsn(int opcode) {
        super.visitInsn(opcode);
        this.frameOffset += 1;
    }

    @Override
    public void visitIntInsn(int opcode, int operand) {
        super.visitIntInsn(opcode, operand);
        this.frameOffset += 1;
    }

    @Override
    public void visitVarInsn(int opcode, int var) {
        super.visitVarInsn(opcode, var);
        this.frameOffset += 1;
    }

    @Override
    public void visitTypeInsn(int opcode, String type) {
        super.visitTypeInsn(opcode, type);
        this.frameOffset += 1;
    }

    @Override
    public void visitMethodInsn(int opcode, String owner, String name, String descriptor, boolean isInterface) {
        super.visitMethodInsn(opcode, owner, name, descriptor, isInterface);
        this.frameOffset += 1;
    }

    @Override
    public void visitInvokeDynamicInsn(String name, String descriptor, Handle bootstrapMethodHandle, Object... bootstrapMethodArguments) {
        super.visitInvokeDynamicInsn(name, descriptor, bootstrapMethodHandle, bootstrapMethodArguments);
        this.frameOffset += 1;
    }

    @Override
    public void visitJumpInsn(int opcode, Label label) {
        super.visitJumpInsn(opcode, label);
        this.frameOffset += 1;
    }

    @Override
    public void visitLdcInsn(Object value) {
        super.visitLdcInsn(value);
        this.frameOffset += 1;
    }

    @Override
    public void visitIincInsn(int var, int increment) {
        super.visitIincInsn(var, increment);
        this.frameOffset += 1;
    }

    @Override
    public void visitTableSwitchInsn(int min, int max, Label dflt, Label... labels) {
        super.visitTableSwitchInsn(min, max, dflt, labels);
        this.frameOffset += 1;
    }

    @Override
    public void visitLookupSwitchInsn(Label dflt, int[] keys, Label[] labels) {
        super.visitLookupSwitchInsn(dflt, keys, labels);
        this.frameOffset += 1;
    }

    @Override
    public void visitMultiANewArrayInsn(String descriptor, int numDimensions) {
        super.visitMultiANewArrayInsn(descriptor, numDimensions);
        this.frameOffset += 1;
    }

    @Override
    public void visitFrame(int type, int nLocal, Object[] local, int nStack, Object[] stack) {
        super.visitFrame(type, nLocal, local, nStack, stack);
        this.frameOffset += 1;
    }

    @Override
    public void visitLabel(Label label) {
        super.visitLabel(label);
        this.frameOffset += 1;
    }

    @Override
    public void visitLineNumber(int line, Label start) {
        super.visitLineNumber(line, start);
        this.frameOffset += 1;
    }

    @Override
    public void visitEnd() {
        super.visitEnd();
        RuntimeAssertionError.assertTrue((null == this.tracker) || (this.frameOffset == this.tracker.getFrameCount()));
    }


    /**
     * NOTE:  All calls to instruction visitation routines are made against super, directly, since we do frame offset accounting within our overrides
     * and that offset only applies to incoming bytecodes, not outgoing ones.
     * 
     * @param opcode The opcode.
     * @param descriptor The type descriptor of the field to which the opcode is applied.
     */
    private void checkInjectLazyLoad(int opcode, String descriptor) {
        // If this is a PUTFIELD or GETFIELD, we want to call "lazyLoad()":
        // -PUTIFELD:  DUP2, POP, INVOKEVIRTUAL
        // -GETIFELD:  DUP, INVOKEVIRTUAL
        if ((Opcodes.PUTFIELD == opcode) && ((null == this.tracker) || !this.tracker.isThisTargetOfPut(this.frameOffset))) {
            // We need to see how big this type is since double and long need a far more complex dance.
            if ((1 == descriptor.length()) && ((DescriptorParser.LONG == descriptor.charAt(0)) || (DescriptorParser.DOUBLE == descriptor.charAt(0)))) {
                // Here, the stack looks like: ... OBJECT, VAR1, VAR2 (top)
                // Where we need:  ... OBJECT, VAR1, VAR2, OBJECT (top)
                // This is multiple stages:
                // DUP2_X1: ... VAR1, VAR2, OBJECT, VAR1, VAR2 (top)
                super.visitInsn(Opcodes.DUP2_X1);
                // POP2: ... VAR1, VAR2, OBJECT (top)
                super.visitInsn(Opcodes.POP2);
                // DUP: ... VAR1, VAR2, OBJECT, OBJECT (top)
                super.visitInsn(Opcodes.DUP);
                // INOKE: ... VAR1, VAR2, OBJECT (top)
                super.visitMethodInsn(Opcodes.INVOKEVIRTUAL, SHADOW_OBJECT_NAME, LAZY_LOAD_NAME, LAZY_LOAD_DESCRIPTOR, false);
                // DUP_X2: ... OBJECT, VAR1, VAR2, OBJECT (top)
                super.visitInsn(Opcodes.DUP_X2);
                // POP: ... OBJECT, VAR1, VAR2 (top)
                super.visitInsn(Opcodes.POP);
            } else {
                // Here, the stack looks like: ... OBJECT, VAR, (top)
                // Where we need:  ... OBJECT, VAR, OBJECT (top)
                // Stages:
                // DUP2: ... OBJECT, VAR, OBJECT, VAR (top)
                super.visitInsn(Opcodes.DUP2);
                // POP: ... OBJECT, VAR, OBJECT (top)
                super.visitInsn(Opcodes.POP);
                // INOKE: ... OBJECT, VAR (top)
                super.visitMethodInsn(Opcodes.INVOKEVIRTUAL, SHADOW_OBJECT_NAME, LAZY_LOAD_NAME, LAZY_LOAD_DESCRIPTOR, false);
            }
        } else if ((Opcodes.GETFIELD == opcode) && ((null == this.tracker) || !this.tracker.isThisTargetOfGet(this.frameOffset))) {
            // Here, the stack looks like: ... OBJECT, (top)
            // Where we need:  ... OBJECT, OBJECT (top)
            super.visitInsn(Opcodes.DUP);
            super.visitMethodInsn(Opcodes.INVOKEVIRTUAL, SHADOW_OBJECT_NAME, LAZY_LOAD_NAME, LAZY_LOAD_DESCRIPTOR, false);
        }
    }
}
