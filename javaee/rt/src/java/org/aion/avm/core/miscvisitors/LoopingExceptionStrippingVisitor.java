package org.aion.avm.core.miscvisitors;

import java.util.ArrayList;
import java.util.List;

import org.aion.avm.core.ClassToolchain;
import org.objectweb.asm.Label;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;


/**
 * This visitor was added for issue-314 to strip out exception handler ranges which are handled within themselves (potentially
 * looping internally) or are handled earlier than when they occur (potentially part of a handler cycle).
 * 
 * The class visitor is just a wrapper on a method visitor which does the real work.
 * 
 * We need to check the bytecode offsets of the labels related to the exception handler, which means buffering them until these
 * offsets have been resolved, which happens before visitMaxs.
 */
public class LoopingExceptionStrippingVisitor extends ClassToolchain.ToolChainClassVisitor {
    public LoopingExceptionStrippingVisitor() {
        super(Opcodes.ASM6);
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String descriptor, String signature, String[] exceptions) {
        MethodVisitor underlying = super.visitMethod(access, name, descriptor, signature, exceptions);
        return new ExceptionMethodVisitor(underlying);
    }


    private static class ExceptionMethodVisitor extends MethodVisitor {
        // The buffer of try-catch data while we wait for the Label offsets to be resolved.
        private final List<TryCatchBlock> labels;
        
        public ExceptionMethodVisitor(MethodVisitor methodVisitor) {
            super(Opcodes.ASM6, methodVisitor);
            this.labels = new ArrayList<>();
        }
        
        @Override
        public void visitTryCatchBlock(Label start, Label end, Label handler, String type) {
            this.labels.add(new TryCatchBlock(start, end, handler, type));
        }
        
        @Override
        public void visitMaxs(int maxStack, int maxLocals) {
            // We must handle try-catch blocks before we call visitMaxs so this is the latest point we can process the labels.
            for (TryCatchBlock block : this.labels) {
                int handlerIndex = block.handler.getOffset();
                if (handlerIndex < block.end.getOffset()) {
                    // This is a backward exception, so strip this.
                } else {
                    // This is a forward exception, so we can include it.
                    super.visitTryCatchBlock(block.start, block.end, block.handler, block.type);
                }
            }
            super.visitMaxs(maxStack, maxLocals);
        }
        
    }


    /**
     * Container of the data related to a try-catch block, while sitting in the buffer.
     */
    private static class TryCatchBlock {
        public final Label start;
        public final Label end;
        public final Label handler;
        public final String type;
        
        public TryCatchBlock(Label start, Label end, Label handler, String type) {
            this.start = start;
            this.end = end;
            this.handler = handler;
            this.type = type;
        }
    }
}
