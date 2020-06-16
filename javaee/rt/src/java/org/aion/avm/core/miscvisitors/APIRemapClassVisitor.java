package org.aion.avm.core.miscvisitors;

import org.aion.avm.core.ClassToolchain;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;

public class APIRemapClassVisitor extends ClassToolchain.ToolChainClassVisitor {

    public APIRemapClassVisitor() {
        super(Opcodes.ASM7);
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String desc, String signature, String[] exceptions) {
        MethodVisitor mv = super.visitMethod(access, name, desc, signature, exceptions);
        return new MethodVisitor(Opcodes.ASM7, mv) {
            @Override
            public void visitMethodInsn(
                    int opcode,
                    String owner,
                    String name,
                    String descriptor,
                    boolean isInterface) {
                if (opcode==Opcodes.INVOKESTATIC &&
                        owner.equals("s/java/util/Map") &&
                        name.equals("avm_ofEntries") &&
                        descriptor.equals("(Lw/_Ls/java/util/Map$Entry;)Ls/java/util/Map;") &&
                        isInterface) {
                    descriptor = "(Li/IObjectArray;)Ls/java/util/Map;";
                }
                super.visitMethodInsn(opcode, owner, name, descriptor, isInterface);
            }
        };
    }
}
