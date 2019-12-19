package org.aion.avm.core.instrument;

import org.objectweb.asm.Handle;
import org.objectweb.asm.Label;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;


/**
 * Collects information regarding BasicBlocks within a method.
 * Specifically, this refers to the opcodes, switch cases, and allocated types within a given block.
 * 
 * Note that this was adapted from the ClassRewriter.BlockMethodReader.
 */
public class BlockBuildingMethodVisitor extends MethodVisitor {
    private final List<BasicBlock> buildingList;
    private List<Integer> currentBuildingBlock;
    private List<Integer> currentBlockSwitches;
    private List<String> currentAllocationList;

    public BlockBuildingMethodVisitor() {
        super(Opcodes.ASM6);
        this.buildingList = new ArrayList<>();
        
        // We need to enter our initial state in the constructor since this method may be abstract and have no code.
        this.currentBuildingBlock = new ArrayList<>();
        this.currentBlockSwitches = new ArrayList<>();
        this.currentAllocationList = new ArrayList<>();
    }

    public List<BasicBlock> getBlockList() {
        return Collections.unmodifiableList(this.buildingList);
    }

    @Override
    public void visitEnd() {
        // This is called after all the code has been walked, so seal the final block.
        handleLabel();
        this.currentBuildingBlock = null;
        this.currentBlockSwitches = null;
        this.currentAllocationList = null;
    }
    @Override
    public void visitFieldInsn(int opcode, String owner, String name, String descriptor) {
        this.currentBuildingBlock.add(opcode);
    }
    @Override
    public void visitIincInsn(int var, int increment) {
        this.currentBuildingBlock.add(Opcodes.IINC);
    }
    @Override
    public void visitInsn(int opcode) {
        this.currentBuildingBlock.add(opcode);
        
        // Note that this could be an athrow, in which case we should handle this as a label.
        // (this, like the jump case, shouldn't normally matter since there shouldn't be unreachable code after it).
        if (Opcodes.ATHROW == opcode) {
            handleLabel();
        }
    }
    @Override
    public void visitIntInsn(int opcode, int operand) {
        this.currentBuildingBlock.add(opcode);
    }
    @Override
    public void visitInvokeDynamicInsn(String name, String descriptor, Handle bootstrapMethodHandle, Object... bootstrapMethodArguments) {
        this.currentBuildingBlock.add(Opcodes.INVOKEDYNAMIC);
    }
    @Override
    public void visitJumpInsn(int opcode, Label label) {
        this.currentBuildingBlock.add(opcode);
        // Jump is the end of a block so emit the label.
        // (note that this is also where if statements show up).
        handleLabel();
    }
    @Override
    public void visitLabel(Label label) {
        handleLabel();
    }
    @Override
    public void visitLdcInsn(Object value) {
        this.currentBuildingBlock.add(Opcodes.LDC);
    }
    @Override
    public void visitLookupSwitchInsn(Label dflt, int[] keys, Label[] labels) {
        this.currentBuildingBlock.add(Opcodes.LOOKUPSWITCH);
        // Count the number of labels plus the default.
        this.currentBlockSwitches.add(labels.length + 1);
        
        // Even though every label is given, there could be unreachable code immediately after.
        handleLabel();
    }
    @Override
    public void visitMethodInsn(int opcode, String owner, String name, String descriptor, boolean isInterface) {
        this.currentBuildingBlock.add(opcode);
    }
    @Override
    public void visitMultiANewArrayInsn(String descriptor, int numDimensions) {
        this.currentBuildingBlock.add(Opcodes.MULTIANEWARRAY);
    }
    @Override
    public void visitTableSwitchInsn(int min, int max, Label dflt, Label... labels) {
        this.currentBuildingBlock.add(Opcodes.TABLESWITCH);
        // Count the number of labels plus the default.
        this.currentBlockSwitches.add(labels.length + 1);
        
        // Even though every label is given, there could be unreachable code immediately after.
        handleLabel();
    }
    @Override
    public void visitTypeInsn(int opcode, String type) {
        this.currentBuildingBlock.add(opcode);
        // If this is a new, att the type to the allocation list for the block.
        if (Opcodes.NEW == opcode) {
            this.currentAllocationList.add(type);
        }
    }
    @Override
    public void visitVarInsn(int opcode, int var) {
        this.currentBuildingBlock.add(opcode);
    }


    /**
     * Called whenever we encounter a label or something we are synthesizing as a label, for block detection purposes.
     */
    private void handleLabel() {
        // Seal the previous block (avoid the case where the block is empty).
        if (!this.currentBuildingBlock.isEmpty()) {
            // Add the block to our finished block list.
            this.buildingList.add(new BasicBlock(this.currentBuildingBlock, this.currentBlockSwitches, this.currentAllocationList));
            // Start the new block.
            this.currentBuildingBlock = new ArrayList<>();
            this.currentBlockSwitches = new ArrayList<>();
            this.currentAllocationList = new ArrayList<>();
        }
    }
}
