package org.aion.avm.core.instrument;

import org.aion.avm.core.ClassToolchain;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.tree.MethodNode;
import i.RuntimeAssertionError;

import java.util.List;
import java.util.Map;


/**
 * Responsible for handling metering of individual code blocks within user code.
 * Internally, this activity is applied on a per-method basis, creating sub-visitors which perform the read and write operations:
 * -BlockBuildingMethodVisitor:  reads the bytecode, finding the basic blocks within each method, and using this to construct a billing total (algorithm internal to this class)
 * -ChargeEnergyInjectionVisitor:  reads the bytecode again, using the basic blocks from the previous step to inject a billing call at the beginning of each block
 */
public class ClassMetering extends ClassToolchain.ToolChainClassVisitor {
    private Map<String, Integer> objectSizes;
    private final BytecodeFeeScheduler bytecodeFeeScheduler;

    public ClassMetering(Map<String, Integer> objectSizes) {
        super(Opcodes.ASM6);

        this.objectSizes = objectSizes;
        
        // Note that we construct the fee scheduler, internally.
        this.bytecodeFeeScheduler = new BytecodeFeeScheduler();
        this.bytecodeFeeScheduler.initialize();
    }

    public MethodVisitor visitMethod(
            final int access,
            final String name,
            final String descriptor,
            final String signature,
            final String[] exceptions) {
        // Capture the visitor which actually constitutes the pipeline - we will need to do another pass before this one.
        MethodVisitor realVisitor = super.visitMethod(access, name, descriptor, signature, exceptions);
        
        // We use a MethodNode since we want to capture the bytecode to walk it twice.
        // The actual final write is done on the realVisitor.
        return new MethodNode(Opcodes.ASM6, access, name, descriptor, signature, exceptions) {
            @Override
            public void visitEnd() {
                // Let the superclass do what it wants to finish this.
                super.visitEnd();

                // Create the read-only visitor and use it to extract the block data.
                BlockBuildingMethodVisitor readingVisitor = new BlockBuildingMethodVisitor();
                this.accept(readingVisitor);
                List<BasicBlock> blocks = readingVisitor.getBlockList();

                // Apply the block fee for the bytecodes it contains.
                for (BasicBlock block : blocks) {
                    long feeForBlock = calculateBlockFee(block);
                    block.setEnergyCost(feeForBlock);
                }

                // We can now build the injection visitor over the real visitor, and accept it in order to add the instrumentation.
                ChargeEnergyInjectionVisitor instrumentingVisitor = new ChargeEnergyInjectionVisitor(realVisitor, blocks);
                this.accept(instrumentingVisitor);
            }
        };
    }

    /**
     * Walks the opcodes in a given block, returning the total fee they will cost the block.
     * Note:  This was original implemented as part of FeeChargingCodeInjector.
     * 
     * @param block A code block.
     * @return The block fee.
     */
    private long calculateBlockFee(BasicBlock block) {
        long blockFee = 0;
        long heapSize = 0;

        // Sum up the bytecode fee in the code block
        for (Integer opcode : block.opcodeSequence) {
            blockFee += this.bytecodeFeeScheduler.getFee(opcode);
        }

        // Sum up the static allocation size, in bytes
        for (String allocationType : block.allocatedTypes) {
            if (this.objectSizes != null) {
                if (this.objectSizes.containsKey(allocationType)) {
                    heapSize += this.objectSizes.get(allocationType);
                }else{
                    throw RuntimeAssertionError.unreachable("Class metering size dict missing : " + allocationType);
                }
            }
        }
        // Apply the heap size cost model.  Note that this is a linear function of the heap allocation size.
        blockFee += heapSize * BytecodeFeeScheduler.BytecodeEnergyLevels.MEMORY.getVal();

        return blockFee;
    }
}
