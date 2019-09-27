package org.aion.avm.core.instrument;

import java.util.HashMap;

import org.objectweb.asm.Opcodes;

/**
 * The bytecode fee schedule as designed on wiki page
 * See {@linktourl https://github.com/aionnetworkp/aion_vm/wiki/Java-Bytecode-fee-schedule}
 */
public class BytecodeFeeScheduler {

    /**
     * The bytecode Energy levels as designed on wiki page
     * See {@linktourl https://github.com/aionnetworkp/aion_vm/wiki/Java-Bytecode-fee-schedule}
     */
    public enum BytecodeEnergyLevels {
        // Basic Energy Levels
        ZERO        (0),
        BASE        (2),
        VERYLOW     (3),
        LOW         (5),
        MID         (8),
        HIGH        (15),
        VERYHIGH    (30),

        // Extra Energy Levels based on the gas cost in solidity.
        // Memory access in AVM is slightly more expensive
        MACCESS     (5),
        // based on the cost for (EQ/LT/GT + ISZERO + JUMPI + JUMPDEST + PUSH)
        FLOWCONTROL (20),
        CREATION    (40),

        // Memory Usage Energy Level
        MEMORY      (3),
        ;

        private final int val;

        BytecodeEnergyLevels(int val) {
            this.val = val;
        }

        public int getVal() {
            return val;
        }
    }

    /**
     * The bytecode fee info, including the Energy levels, alpha, delta and the static fee.
     */
    private class BytecodeFeeInfo {
        BytecodeEnergyLevels nrgLvl;
        BytecodeEnergyLevels extraNrgLvl;
        int delta;    // number of the items removed from the stack
        int alpha;    // number of the additional items placed on the stack
        long fee;     // the static fee of this bytecode, generally including the computation cost and stack memory cost, assuming that the heap memory cost is added dynamically.

        /**
         * Constructor.
         * @param nrgLvl a BytecodeEnergyLevels enum regarding the basic Energy level
         * @param extraNrgLvl a BytecodeEnergyLevels enum regarding the extra Energy level
         * @param delta number of the items this bytecode removes from the stack
         * @param alpha number of the additional items this bytecode places on the stack
         * @param fee the static fee of this bytecode
         */
        private BytecodeFeeInfo(BytecodeEnergyLevels nrgLvl,
                                BytecodeEnergyLevels extraNrgLvl,
                                int delta,
                                int alpha,
                                int fee) {
            this.nrgLvl = nrgLvl;
            this.extraNrgLvl = extraNrgLvl;
            this.delta = delta;
            this.alpha = alpha;
            this.fee = fee;
        }

        /**
         * return the Energy level.
         */
        public BytecodeEnergyLevels getNrgLvl() {
            return nrgLvl;
        }

        /**
         * return the extra Energy level.
         */
        public BytecodeEnergyLevels getExtraNrgLvl() {
            return extraNrgLvl;
        }

        /**
         * return the Delta.
         */
        public int getDelta() {
            return delta;
        }

        /**
         * return the Alpha.
         */
        public int getAlpha() {
            return alpha;
        }

        /**
         * return the fee.
         */
        public long getFee() {
            return fee;
        }

        private void setFee(long fee) {
            if (fee >= 0) {
                this.fee = fee;
            }
            else {
                throw new IllegalArgumentException("Bytecode fee cannot be negative.");
            }
        }
    }

    /**
     * A hashmap that stores the fee info for each bytecode.
     */
    private HashMap<Integer, BytecodeFeeInfo> feeScheduleMap;

    /**
     * Constructor.
     */
    public BytecodeFeeScheduler() {
    }

    /**
     * Initialize the fee schedule Hashmap. Add the fee info for each bytecode and calculate the static fee.
     * The bytecode Energy levels, alpha and delta are listed on wiki page
     * See {@linktourl https://github.com/aionnetworkp/aion_vm/wiki/Java-Bytecode-fee-schedule}
     */
    public void initialize() {
        feeScheduleMap = new HashMap<>() {{
            // NOP
            put(Opcodes.NOP, new BytecodeFeeInfo(BytecodeEnergyLevels.ZERO, BytecodeEnergyLevels.ZERO, 0, 0, 0));

            // Load and Store
            // Not in ASM Opcodes but Contants: xLOAD_[0/1/2/3] -- visitor of xLOAD; xSTORE_[0/1/2/3] -- visitor of xSTORE
            put(Opcodes.ILOAD,      new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.LLOAD,      new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.FLOAD,      new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.DLOAD,      new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.ALOAD,      new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.IALOAD,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 0, 1, 0));
            put(Opcodes.LALOAD,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 0, 1, 0));
            put(Opcodes.FALOAD,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 0, 1, 0));
            put(Opcodes.DALOAD,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 0, 1, 0));
            put(Opcodes.AALOAD,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 0, 1, 0));
            put(Opcodes.BALOAD,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 0, 1, 0));
            put(Opcodes.CALOAD,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 0, 1, 0));
            put(Opcodes.SALOAD,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 0, 1, 0));
            put(Opcodes.ISTORE,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 1, 0, 0));
            put(Opcodes.LSTORE,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 1, 0, 0));
            put(Opcodes.FSTORE,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 1, 0, 0));
            put(Opcodes.DSTORE,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 1, 0, 0));
            put(Opcodes.ASTORE,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 1, 0, 0));
            put(Opcodes.IASTORE,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 1, 0, 0));
            put(Opcodes.LASTORE,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 1, 0, 0));
            put(Opcodes.FASTORE,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 1, 0, 0));
            put(Opcodes.DASTORE,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 1, 0, 0));
            put(Opcodes.AASTORE,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 1, 0, 0));
            put(Opcodes.BASTORE,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 1, 0, 0));
            put(Opcodes.CASTORE,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 1, 0, 0));
            put(Opcodes.SASTORE,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 1, 0, 0));

            // Arithmetic
            put(Opcodes.IADD,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.LADD,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.FADD,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.DADD,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.ISUB,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.LSUB,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.FSUB,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.DSUB,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.IMUL,   new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.LMUL,   new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.FMUL,   new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.DMUL,   new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.IDIV,   new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.LDIV,   new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.FDIV,   new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.DDIV,   new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.IREM,   new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.LREM,   new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.FREM,   new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.DREM,   new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.INEG,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.LNEG,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.FNEG,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.DNEG,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.ISHL,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.LSHL,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.ISHR,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.LSHR,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.IUSHR,  new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.LUSHR,  new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.IAND,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.LAND,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.IOR,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.LOR,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.IXOR,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.LXOR,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.IINC,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 0, 0));
            put(Opcodes.LCMP,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.FCMPL,  new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.FCMPG,  new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.DCMPL,  new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));
            put(Opcodes.DCMPG,  new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 1, 0));

            // Type Conversion
            put(Opcodes.I2L,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.I2F,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.I2D,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.L2I,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.L2F,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.L2D,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.F2I,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.F2L,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.F2D,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.D2I,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.D2L,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.D2F,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.I2B,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.I2C,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.I2S,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));

            // Object Creation and Manipulation
            put(Opcodes.GETSTATIC,      new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 0, 1, 0));
            put(Opcodes.PUTSTATIC,      new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 1, 0, 0));
            put(Opcodes.GETFIELD,       new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 0, 1, 0));
            put(Opcodes.PUTFIELD,       new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.MACCESS, 2, 0, 0));
            put(Opcodes.NEW,            new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.CREATION, 0, 1, 0));
            put(Opcodes.NEWARRAY,       new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.CREATION, 1, 1, 0));
            put(Opcodes.ANEWARRAY,      new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.CREATION, 1, 1, 0));
            put(Opcodes.ARRAYLENGTH,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.CHECKCAST,      new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.INSTANCEOF,     new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 1, 0));
            put(Opcodes.MULTIANEWARRAY, new BytecodeFeeInfo(BytecodeEnergyLevels.LOW,     BytecodeEnergyLevels.CREATION, 5, 1, 0)); // delta of 5 is not accurate but bigger than alpha, so the value does not matter.

            // Operand Stack Management
            // Not in ASM Opcodes but Contants: LDC_W and LDC2_w -- visitor of LDC
            put(Opcodes.ACONST_NULL,  new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.ICONST_M1,  new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.ICONST_0,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.ICONST_1,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.ICONST_2,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.ICONST_3,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.ICONST_4,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.ICONST_5,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.LCONST_0,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.LCONST_1,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.FCONST_0,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.FCONST_1,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.FCONST_2,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.DCONST_0,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.DCONST_1,   new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.BIPUSH,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.SIPUSH,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.LDC,        new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.POP,        new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 0, 0));
            put(Opcodes.POP2,       new BytecodeFeeInfo(BytecodeEnergyLevels.BASE,    BytecodeEnergyLevels.ZERO, 1, 0, 0));
            put(Opcodes.DUP,        new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 0, 1, 0));
            put(Opcodes.DUP_X1,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 3, 0));
            put(Opcodes.DUP_X2,     new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 3, 4, 0));
            put(Opcodes.DUP2,       new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 4, 0));
            put(Opcodes.DUP2_X1,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 3, 5, 0));
            put(Opcodes.DUP2_X2,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 4, 6, 0));
            put(Opcodes.SWAP,       new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.ZERO, 2, 2, 0));

            // Control Transfer
            // AVM to reject: jsr, ret, jsr_w
            // Not in ASM Opcodes but Contants: GOTO_W -- visitor of GOTO
            put(Opcodes.IFEQ,         new BytecodeFeeInfo(BytecodeEnergyLevels.BASE, BytecodeEnergyLevels.FLOWCONTROL,    1, 0, 0));
            put(Opcodes.IFNE,         new BytecodeFeeInfo(BytecodeEnergyLevels.BASE, BytecodeEnergyLevels.FLOWCONTROL,    1, 0, 0));
            put(Opcodes.IFLT,         new BytecodeFeeInfo(BytecodeEnergyLevels.BASE, BytecodeEnergyLevels.FLOWCONTROL,    1, 0, 0));
            put(Opcodes.IFGE,         new BytecodeFeeInfo(BytecodeEnergyLevels.BASE, BytecodeEnergyLevels.FLOWCONTROL,    1, 0, 0));
            put(Opcodes.IFGT,         new BytecodeFeeInfo(BytecodeEnergyLevels.BASE, BytecodeEnergyLevels.FLOWCONTROL,    1, 0, 0));
            put(Opcodes.IFLE,         new BytecodeFeeInfo(BytecodeEnergyLevels.BASE, BytecodeEnergyLevels.FLOWCONTROL,    1, 0, 0));
            put(Opcodes.IF_ICMPEQ,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 2, 0, 0));
            put(Opcodes.IF_ICMPNE,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 2, 0, 0));
            put(Opcodes.IF_ICMPLT,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 2, 0, 0));
            put(Opcodes.IF_ICMPGE,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 2, 0, 0));
            put(Opcodes.IF_ICMPGT,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 2, 0, 0));
            put(Opcodes.IF_ICMPLE,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 2, 0, 0));
            put(Opcodes.IF_ACMPEQ,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 2, 0, 0));
            put(Opcodes.IF_ACMPNE,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 2, 0, 0));
            put(Opcodes.GOTO,         new BytecodeFeeInfo(BytecodeEnergyLevels.BASE, BytecodeEnergyLevels.FLOWCONTROL,    0, 0, 0));
            put(Opcodes.TABLESWITCH,  new BytecodeFeeInfo(BytecodeEnergyLevels.LOW, BytecodeEnergyLevels.FLOWCONTROL,     1, 0, 0));
            put(Opcodes.LOOKUPSWITCH, new BytecodeFeeInfo(BytecodeEnergyLevels.LOW, BytecodeEnergyLevels.FLOWCONTROL,     1, 0, 0));
            put(Opcodes.IFNULL,       new BytecodeFeeInfo(BytecodeEnergyLevels.BASE, BytecodeEnergyLevels.FLOWCONTROL,    1, 0, 0));
            put(Opcodes.IFNONNULL,    new BytecodeFeeInfo(BytecodeEnergyLevels.BASE, BytecodeEnergyLevels.FLOWCONTROL,    1, 0, 0));

            // Method Invocation and Return
            put(Opcodes.IRETURN,            new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 5, 1, 0));
            put(Opcodes.LRETURN,            new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 5, 1, 0));
            put(Opcodes.FRETURN,            new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 5, 1, 0));
            put(Opcodes.DRETURN,            new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 5, 1, 0));
            put(Opcodes.ARETURN,            new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 5, 1, 0));
            put(Opcodes.RETURN,             new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 5, 0, 0));
            put(Opcodes.INVOKEVIRTUAL,      new BytecodeFeeInfo(BytecodeEnergyLevels.HIGH, BytecodeEnergyLevels.FLOWCONTROL, 5, 1, 0));
            put(Opcodes.INVOKESPECIAL,      new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 5, 1, 0));
            put(Opcodes.INVOKESTATIC,       new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 5, 1, 0));
            put(Opcodes.INVOKEINTERFACE,    new BytecodeFeeInfo(BytecodeEnergyLevels.HIGH, BytecodeEnergyLevels.FLOWCONTROL, 5, 1, 0));
            put(Opcodes.INVOKEDYNAMIC,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYHIGH, BytecodeEnergyLevels.FLOWCONTROL, 5, 1, 0));

            // Throwing Exceptions
            put(Opcodes.ATHROW,    new BytecodeFeeInfo(BytecodeEnergyLevels.VERYLOW, BytecodeEnergyLevels.FLOWCONTROL, 5, 1, 0));

            // AVM to reject: MONITORENTER, MONITOREXIT
        }};

        // calculate the static fee for each bytecode.
        for (int op : feeScheduleMap.keySet()) {
            BytecodeFeeInfo feeInfo = feeScheduleMap.get(op);

            // believing no overflow here so no need to cast from int to long during the calculation
            long fee = feeInfo.getNrgLvl().getVal() + feeInfo.getExtraNrgLvl().getVal();

            feeInfo.setFee(fee);
        }
    }

    /**
     * return the bytecode fee.
     */
    public long getFee(int op) {
        if (feeScheduleMap.containsKey(op)) {
            return feeScheduleMap.get(op).getFee();
        } else {
            throw new IllegalArgumentException("This bytecode is not in the fee schedule.");
        }
    }
}
