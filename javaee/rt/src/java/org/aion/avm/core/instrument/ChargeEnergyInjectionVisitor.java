package org.aion.avm.core.instrument;

import java.util.List;

import i.Helper;
import i.RuntimeAssertionError;
import org.objectweb.asm.Handle;
import org.objectweb.asm.Label;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;


/**
 * The visitor responsible for injecting the "Helper.chargeEnergy(int)" calls.
 *
 * This is done per-block, so the in-order list of BasicBlock objects generated during a pre-pass must be provided.
 *
 * The general approach is to walk over the bytecode, injecting a call-out immediately before the first bytecode
 * which comes after a label (as a BasicBlock is defined as starting after a label).
 *
 * The actual value charged is expected to be set on the BasicBlock elsewhere as this class doesn't know where the
 * number came from, just how to inject the call.
 */
public class ChargeEnergyInjectionVisitor extends MethodVisitor {
    private final List<BasicBlock> blocks;
    private boolean scanningToNewBlockStart;
    private int nextBlockIndexToWrite;

    public ChargeEnergyInjectionVisitor(MethodVisitor target, List<BasicBlock> blocks) {
        super(Opcodes.ASM7, target);
        this.blocks = blocks;
    }

    @Override
    public void visitCode() {
        // We initialize the state machine.
        this.scanningToNewBlockStart = true;
        this.nextBlockIndexToWrite = 0;
        // We also need to tell the writer to advance.
        super.visitCode();
    }
    @Override
    public void visitEnd() {
        // We never have empty blocks, in our implementation, so we should always be done when we reach this point.
        RuntimeAssertionError.assertTrue(this.blocks.size() == this.nextBlockIndexToWrite);
        // Tell the writer we are done.
        super.visitEnd();
    }
    @Override
    public void visitFieldInsn(int opcode, String owner, String name, String descriptor) {
        checkInject();
        super.visitFieldInsn(opcode, owner, name, descriptor);
    }
    @Override
    public void visitIincInsn(int var, int increment) {
        checkInject();
        super.visitIincInsn(var, increment);
    }
    @Override
    public void visitInsn(int opcode) {
        checkInject();
        super.visitInsn(opcode);

        // Note that this could be an athrow, in which case we should handle this as a label.
        // (this, like the jump case, shouldn't normally matter since there shouldn't be unreachable code after it).
        if (Opcodes.ATHROW == opcode) {
            this.scanningToNewBlockStart = true;
        }
    }
    @Override
    public void visitIntInsn(int opcode, int operand) {
        checkInject();
        super.visitIntInsn(opcode, operand);
    }
    @Override
    public void visitJumpInsn(int opcode, Label label) {
        checkInject();
        super.visitJumpInsn(opcode, label);

        // Jump is the end of a block so emit the label.
        // (note that this is also where if statements show up).
        this.scanningToNewBlockStart = true;
    }
    @Override
    public void visitLabel(Label label) {
        // The label means that we found a new block (although there might be several labels before it actually starts)
        // so enter the state machine mode where we are looking for that beginning of a block.
        this.scanningToNewBlockStart = true;
        super.visitLabel(label);
    }
    @Override
    public void visitLdcInsn(Object value) {
        checkInject();
        super.visitLdcInsn(value);
    }
    @Override
    public void visitLookupSwitchInsn(Label dflt, int[] keys, Label[] labels) {
        checkInject();
        super.visitLookupSwitchInsn(dflt, keys, labels);
    }
    @Override
    public void visitMethodInsn(int opcode, String owner, String name, String descriptor, boolean isInterface) {
        checkInject();
        super.visitMethodInsn(opcode, owner, name, descriptor, isInterface);
    }
    @Override
    public void visitMultiANewArrayInsn(String descriptor, int numDimensions) {
        checkInject();
        super.visitMultiANewArrayInsn(descriptor, numDimensions);
    }
    @Override
    public void visitTableSwitchInsn(int min, int max, Label dflt, Label... labels) {
        checkInject();
        super.visitTableSwitchInsn(min, max, dflt, labels);
    }
    @Override
    public void visitTypeInsn(int opcode, String type) {
        checkInject();
        super.visitTypeInsn(opcode, type);
    }
    @Override
    public void visitVarInsn(int opcode, int var) {
        checkInject();
        super.visitVarInsn(opcode, var);
    }
    @Override
    public void visitMaxs(int maxStack, int maxLocals) {
        super.visitMaxs(maxStack, maxLocals);
    }

    @Override
    public void visitInvokeDynamicInsn(String name, String descriptor, Handle bootstrapMethodHandle, Object... bootstrapMethodArguments) {
        checkInject();
        super.visitInvokeDynamicInsn(name, descriptor, bootstrapMethodHandle, bootstrapMethodArguments);
    }

    /**
     * Common state machine advancing call.  Called at every instruction to see if we need to inject and/or advance
     * the state machine.
     */
    private void checkInject() {
        if (this.scanningToNewBlockStart) {
            // We were waiting for this so make sure that this block has some associated cost.
            BasicBlock currentBlock = this.blocks.get(this.nextBlockIndexToWrite);
            long currentEnergyCost = currentBlock.getEnergyCost();
            // We should never encounter a block with a zero/negative cost.
            RuntimeAssertionError.assertTrue(currentEnergyCost > 0L);
            // We track this as a long but report it as an int so prove that this can't overflow.
            RuntimeAssertionError.assertTrue(currentEnergyCost <= (long)Integer.MAX_VALUE);

            // Inject the bytecodes.
            super.visitLdcInsn(Integer.valueOf((int)currentEnergyCost));
            super.visitMethodInsn(Opcodes.INVOKESTATIC, Helper.RUNTIME_HELPER_NAME, "chargeEnergy", "(I)V", false);

            // Reset the state machine for the next block.
            this.scanningToNewBlockStart = false;
            this.nextBlockIndexToWrite += 1;
        }
    }
}
