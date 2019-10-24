package org.aion.avm.core.stacktracking;

import org.aion.avm.core.ClassToolchain;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.commons.GeneratorAdapter;
import org.objectweb.asm.tree.MethodNode;


/**
 * This visitor listens to the methods being read and then passes them to the StackWatcherMethodAdapter to be instrumented for stack overflow protection.
 * That other visitor does most of the work while this one is only used to check some of the bounds of the method, first.
 */
public class StackWatcherClassAdapter extends ClassToolchain.ToolChainClassVisitor {
    public StackWatcherClassAdapter() {
        super(Opcodes.ASM6);
    }

    @Override
    public MethodVisitor visitMethod(final int access, final String name,
            final String desc, final String signature, final String[] exceptions)
    {
        MethodVisitor mv = cv.visitMethod(access, name, desc, signature, exceptions);
        GeneratorAdapter ga = new GeneratorAdapter(mv, access, name, desc);
        StackWatcherMethodAdapter ma = new StackWatcherMethodAdapter(ga, access, name, desc);

        // Wrap the method adapter into a method node to access method information.
        return new MethodNode(Opcodes.ASM6, access, name, desc, signature, exceptions)
        {
            @Override
            public void visitEnd() {
                ma.setTryCatchBlockNum(this.tryCatchBlocks.size());
                ma.setMax(this, this.maxLocals, this.maxStack);
                this.accept(ma);
            }
        };
    }
}
