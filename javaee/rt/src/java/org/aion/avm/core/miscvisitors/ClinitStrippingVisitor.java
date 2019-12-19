package org.aion.avm.core.miscvisitors;

import org.aion.avm.core.ClassToolchain;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;


/**
 * A visitor which purely exists to strip the &lt;clinit&gt; method from any class it is given.
 * We do this so that we can avoid redundant constant initialization when loaded classes for which we already have a complete persistence record.
 * NOTE:  This is NOT part of the standard tool-chain and is run only after the initial invocation for deployment, before we save the jar for
 * later CALL invocations.
 * See issue-134 for more details on this design.
 */
public class ClinitStrippingVisitor extends ClassToolchain.ToolChainClassVisitor {
    private static final String kClinitName = "<clinit>";

    public ClinitStrippingVisitor() {
        super(Opcodes.ASM6);
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String descriptor, String signature, String[] exceptions) {
        return (kClinitName.equals(name))
                ? null
                : super.visitMethod(access, name, descriptor, signature, exceptions)
        ;
    }
}
