package org.aion.avm.core.persistence;

import java.util.List;
import java.util.stream.Collectors;

import i.RuntimeAssertionError;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;
import org.objectweb.asm.tree.AbstractInsnNode;
import org.objectweb.asm.tree.analysis.AnalyzerException;
import org.objectweb.asm.tree.analysis.BasicInterpreter;
import org.objectweb.asm.tree.analysis.BasicValue;
import org.objectweb.asm.tree.analysis.Interpreter;
import org.objectweb.asm.tree.analysis.Value;


/**
 * Used to decide where to inject "lazyLoad()" calls.  Specifically, this is used to determine where the "this" pointer is, in a constructor,
 * to avoid injecting the call against that object since field accesses can be made prior to initialization (see issue-156).
 * Internally, this sits on top of the BasicInterpreter, as it only needs to add a piece of meta-data on top of that logic.
 * 
 * WARNING:  This implementation assumes that the first call to newValue(), within a constructor, is made for the "this" pointer.
 */
public class ConstructorThisInterpreter extends Interpreter<ConstructorThisInterpreter.ThisValue> {
    private final BasicInterpreter underlying;
    private boolean isNextThis;

    public ConstructorThisInterpreter() {
        super(Opcodes.ASM6);
        this.underlying = new BasicInterpreter();
        // NOTE:  This is based on the assumption that a non-static method's first newValue() call is for "this".
        this.isNextThis = true;
    }

    @Override
    public ThisValue newValue(Type type) {
        ThisValue result = null;
        BasicValue original = this.underlying.newValue(type);
        if (null != original) {
            if (this.isNextThis) {
                // WARNING:  This is where we assume that we are being asked to describe the "this".
                result = ThisValue.createThis(original);
                this.isNextThis = false;
            } else {
                result = ThisValue.createNotThis(original);
            }
        }
        return result;
    }

    @Override
    public ThisValue newOperation(AbstractInsnNode insn) throws AnalyzerException {
        // These create new values so it obviously isn't this.
        BasicValue basic = this.underlying.newOperation(insn);
        return ThisValue.createNotThis(basic);
    }

    @Override
    public ThisValue copyOperation(AbstractInsnNode insn, ThisValue value) throws AnalyzerException {
        // We are just moving the value around so it maintains is this/not_this state.
        boolean isThis = value.isThis;
        BasicValue basic = this.underlying.copyOperation(insn, value.underlying);
        return isThis
                ? ThisValue.createThis(basic)
                : ThisValue.createNotThis(basic);
    }

    @Override
    public ThisValue unaryOperation(AbstractInsnNode insn, ThisValue value) throws AnalyzerException {
        // These create a different result (possibly null), so isn't this.
        BasicValue basic = this.underlying.unaryOperation(insn, value.underlying);
        return (null != basic)
                ? ThisValue.createNotThis(basic)
                : null;
    }

    @Override
    public ThisValue binaryOperation(AbstractInsnNode insn, ThisValue value1, ThisValue value2) throws AnalyzerException {
        // These create a different result (possibly null), so isn't this.
        BasicValue basic = this.underlying.binaryOperation(insn, value1.underlying, value2.underlying);
        return (null != basic)
                ? ThisValue.createNotThis(basic)
                : null;
    }

    @Override
    public ThisValue ternaryOperation(AbstractInsnNode insn, ThisValue value1, ThisValue value2, ThisValue value3) throws AnalyzerException {
        // These create a different result (possibly null), so isn't this.
        BasicValue basic = this.underlying.ternaryOperation(insn, value1.underlying, value2.underlying, value3.underlying);
        return (null != basic)
                ? ThisValue.createNotThis(basic)
                : null;
    }

    @Override
    public ThisValue naryOperation(AbstractInsnNode insn, List<? extends ThisValue> values) throws AnalyzerException {
        // A new type is created, so definitely not this.
        List<BasicValue> basics = values.stream().map((value) -> value.underlying).collect(Collectors.toList());
        BasicValue basic = this.underlying.naryOperation(insn, basics);
        return (null != basic)
                ? ThisValue.createNotThis(basic)
                : null;
    }

    @Override
    public void returnOperation(AbstractInsnNode insn, ThisValue value, ThisValue expected) throws AnalyzerException {
        this.underlying.returnOperation(insn, value.underlying, expected.underlying);
    }

    @Override
    public ThisValue merge(ThisValue value1, ThisValue value2) {
        boolean isThis = value1.isThis && value2.isThis;
        BasicValue basic = this.underlying.merge(value1.underlying, value2.underlying);
        return isThis
                ? ThisValue.createThis(basic)
                : ThisValue.createNotThis(basic);
    }


    /**
     * The Value we want to use is just an additional piece of meta-data (is this "this") sitting on top of the BasicValue.
     * Note that the "equals()" implementation of Value is incredibly important!  If this is missing, infinite loops can
     * occur in the Analyzer since it assumes it isn't making progress (every time it merges 2 types, there is a change).
     */
    public static class ThisValue implements Value {
        public static ThisValue createThis(BasicValue underlying) {
            return new ThisValue(true, underlying);
        }
        
        public static ThisValue createNotThis(BasicValue underlying) {
            return new ThisValue(false, underlying);
        }
        
        
        public final boolean isThis;
        public final BasicValue underlying;
        
        private ThisValue(boolean isThis, BasicValue underlying) {
            RuntimeAssertionError.assertTrue(null != underlying);
            this.isThis = isThis;
            this.underlying = underlying;
        }
        @Override
        public int getSize() {
            return this.underlying.getSize();
        }
        @Override
        public int hashCode() {
            return this.underlying.hashCode();
        }
        @Override
        public boolean equals(Object obj) {
            boolean isEqual = (this == obj);
            if (!isEqual && (obj instanceof ThisValue)) {
                ThisValue other = (ThisValue) obj;
                isEqual = (this.isThis == other.isThis) && this.underlying.equals(other.underlying);
            }
            return isEqual;
        }
    }
}
