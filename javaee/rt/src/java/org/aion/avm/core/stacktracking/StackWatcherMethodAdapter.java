package org.aion.avm.core.stacktracking;

import i.Helper;
import i.RuntimeAssertionError;
import org.objectweb.asm.Label;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;
import org.objectweb.asm.commons.AdviceAdapter;
import org.objectweb.asm.commons.GeneratorAdapter;
import org.objectweb.asm.commons.Method;
import org.objectweb.asm.tree.MethodNode;

import java.util.ArrayList;


/**
 * Created by StackWatcherClassAdapter to instrument methods for deterministic stack overflow detection.
 * This instrumentation involves the calls to the following helper methods:
 * -enterMethod
 * -exitMethod
 * -getCurStackDepth
 * -getCurStackSize
 * -enterCatchBlock
 *
 * The total flow of this is complicated so is worth explaining:
 * 1) Enter method is called on entering method, to increment the stack depth.
 * 2) On exit, exit method is called to decrement this.
 * (those are the obvious cases, exception handling is more complex)
 * 3) If there are any exception handlers in the method, the stack depth is captured into a local variable.
 * 4) In all exception handlers, the stack depth is forced back to the value stored in this local.
 * This means that the depth is reset to the same value when entering the method or returning to it via exception handler.
 */
class StackWatcherMethodAdapter extends AdviceAdapter {
    private int stackDepthLocalVariableIndex = -1;
    private int stackSizeLocalVariableIndex = -1;
    private int maxLocals = 0;
    private int maxStack = 0;
    private int tryCatchBlockCount = 0;

    // These values represent the upper bound of additional locals & stack space our instrumented code
    // uses. The ClassWriter overwrites the max-locals and max-stack in the end since we always specify
    // COMPUTE_FRAMES; however, intermediate stages of the pipeline may try to perform verification of
    // the stack shape etc. prior to the ClassWriter recomputing these values, and so we safely pass
    // off these upper bounds to satisfy any intermediate checks.
    // See AKI-108 for more details.
    private static final int NUM_INSTRUMENTED_LOCALS = 2;
    private static final int NUM_INSTRUMENTED_STACK = 2;

    //List of exception handler code label (aka the start of catch block)
    private ArrayList<Label> catchBlockList = new ArrayList<Label>();

    //JAVA asm Type for later use.
    private Type typeInt = Type.getType(int.class);
    private Type typeHelper = Type.getType("L" + Helper.RUNTIME_HELPER_NAME + ";");

    public StackWatcherMethodAdapter(final GeneratorAdapter mv,
            final int access, final String name, final String desc)
    {
        super(Opcodes.ASM7, mv, access, name, desc);
    }

    public void setMax(MethodNode node, int l, int s){
        this.maxLocals = l;
        this.maxStack = s;
    }

    @Override
    public void visitMaxs(int maxStack, int maxLocals) {
        RuntimeAssertionError.assertTrue(maxStack == this.maxStack);
        RuntimeAssertionError.assertTrue(maxLocals == this.maxLocals);
        super.visitMaxs(maxStack + NUM_INSTRUMENTED_STACK, maxLocals + NUM_INSTRUMENTED_LOCALS);
    }

    public void setTryCatchBlockNum(int l){
        this.tryCatchBlockCount = l;
    }

    @Override
    public void visitCode(){
        super.visitCode();

        // Push the current stack size to operand stack and invoke AVMStackWatcher.enterMethod(int)
        Method m1 = Method.getMethod("void enterMethod(int)");
        visitLdcInsn(this.maxLocals + this.maxStack);
        invokeStatic(typeHelper, m1);

        // If current method has at least one try catch block, we need to generate a StackWatcher stamp.
        if (this.tryCatchBlockCount > 0){
            //invoke AVMStackWatcher.getCurStackDepth() and put the result into local variable
            Method m2 = Method.getMethod("int getCurStackDepth()");
            invokeStatic(typeHelper, m2);
            this.stackDepthLocalVariableIndex = newLocal(typeInt);
            storeLocal(this.stackDepthLocalVariableIndex, typeInt);

            //invoke AVMStackWatcher.getCurStackSize() and put the result into local variable
            Method m3 = Method.getMethod("int getCurStackSize()");
            invokeStatic(typeHelper, m3);
            this.stackSizeLocalVariableIndex = newLocal(typeInt);
            storeLocal(this.stackSizeLocalVariableIndex, typeInt);
        }
    }

    @Override
    protected void onMethodExit(int opcode){
        // Push the current stack size to operand stack and invoke AVMStackWatcher.exitMethod(int)
        Method m1 = Method.getMethod("void exitMethod(int)");
        visitLdcInsn(this.maxLocals + this.maxStack);
        invokeStatic(typeHelper, m1);
    }


    @Override
    public void visitTryCatchBlock(Label start, Label end, Label handler, String type){
        // visitTryCatchBlock is guaranteed to be called before the visits of its labels.
        // we keep track of all exception handlers, so we can instrument them when they are visited.
        catchBlockList.add(handler);
        mv.visitTryCatchBlock(start, end, handler, type);
    }

    @Override
    public void visitLabel(Label label){
        mv.visitLabel(label);
        // We instrument the code (start of catch block) if the label we are visiting is an exception handler
        if (catchBlockList.contains(label)){
            // Load the stamp from LVT
            loadLocal(this.stackDepthLocalVariableIndex, typeInt);
            loadLocal(this.stackSizeLocalVariableIndex, typeInt);
            Method m1 = Method.getMethod("void enterCatchBlock(int, int)");
            invokeStatic(typeHelper, m1);
        }
    }
}
