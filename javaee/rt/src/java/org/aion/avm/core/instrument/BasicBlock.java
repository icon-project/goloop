package org.aion.avm.core.instrument;

import org.objectweb.asm.util.Printer;

import java.util.Collections;
import java.util.List;


/**
 * Describes a single basic block within a method.
 * Note that only the opcodeSequence, switchCounts, and allocatedTypes are meant to be immutable.
 * The variable energyCost is mutable, deliberately, to allow for mutation requests.
 */
public class BasicBlock {
    public final List<Integer> opcodeSequence;
    /**
     * Corresponds to the number of switch cases (including default) included in each switch within the block.
     * Note that these are per-switch, NOT per-opcode.  Hence, a block with 10 opcodes, 2 being switches, will only have 2 entries in this list.
     */
    public final List<Integer> switchCases;
    public final List<String> allocatedTypes;
    private long energyCost;

    public BasicBlock(List<Integer> opcodes, List<Integer> switchCases, List<String> allocatedTypes) {
        this.opcodeSequence = Collections.unmodifiableList(opcodes);
        this.switchCases = Collections.unmodifiableList(switchCases);
        this.allocatedTypes = Collections.unmodifiableList(allocatedTypes);
    }

    /**
     * Sets the cost of the block, so that the accounting idiom will be prepended when the block is next serialized.
     * @param energyCost The energy cost.
     */
    public void setEnergyCost(long energyCost) {
        this.energyCost = energyCost;
    }

    /**
     * Called when serializing the block to determine if the accounting idiom should be prepended.
     * @return The energy cost of the block.
     */
    public long getEnergyCost() {
        return this.energyCost;
    }

    @Override
    public String toString() {
        final var builder = new StringBuilder("BasicBlock{");
        opcodeSequence.stream().map(i -> Printer.OPCODES[i]).forEach(s -> builder.append(s).append('\n'));
        builder.append('}');
        return builder.toString();
    }
}
