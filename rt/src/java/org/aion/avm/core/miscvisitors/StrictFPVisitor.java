package org.aion.avm.core.miscvisitors;

import org.aion.avm.core.ClassToolchain;
import org.objectweb.asm.Opcodes;

import static org.objectweb.asm.Opcodes.ACC_STRICT;

public class StrictFPVisitor extends ClassToolchain.ToolChainClassVisitor {

    public StrictFPVisitor() {
        super(Opcodes.ASM6);
    }

    @Override
    public void visit(int version, int access, String name, String signature, String superName, String[] interfaces) {
        super.visit(version, access | ACC_STRICT, name, signature, superName, interfaces);
    }
}
