package org.aion.avm.core.persistence;

import i.RuntimeAssertionError;
import org.objectweb.asm.tree.analysis.Frame;


/**
 * Used to answer questions of where the "this" pointer is, on the stack.  Specifically, the 2 top stack slots (for GETFIELD/PUTFIELD).
 */
public class StackThisTracker {
    private final Frame<ConstructorThisInterpreter.ThisValue>[] frames;

    public StackThisTracker(Frame<ConstructorThisInterpreter.ThisValue>[] frames) {
        RuntimeAssertionError.assertTrue(null != frames);
        this.frames = frames;
    }

    public boolean isThisTargetOfGet(int index) {
        Frame<ConstructorThisInterpreter.ThisValue> frame = this.frames[index];
        // Note that we will treat this as safe, even on invalid bytecode (we handle the stack underflow).
        int size = frame.getStackSize();
        return (size > 0)
                ? frame.getStack(size - 1).isThis
                : false;
    }

    public boolean isThisTargetOfPut(int index) {
        Frame<ConstructorThisInterpreter.ThisValue> frame = this.frames[index];
        // Note that we will treat this as safe, even on invalid bytecode (we handle the stack underflow).
        int size = frame.getStackSize();
        return (size > 1)
                ? frame.getStack(size - 2).isThis
                : false;
    }

    public int getFrameCount() {
        return this.frames.length;
    }
}
