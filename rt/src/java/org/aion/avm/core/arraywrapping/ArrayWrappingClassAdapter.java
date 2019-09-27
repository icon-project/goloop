package org.aion.avm.core.arraywrapping;

import org.aion.avm.core.ClassToolchain;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;

/**
 * A class visitor that convert array from field/method signature into array wrapper.
 */

public class ArrayWrappingClassAdapter extends ClassToolchain.ToolChainClassVisitor {

    public ArrayWrappingClassAdapter() {
        super(Opcodes.ASM6);
    }

    @Override
    public FieldVisitor visitField(int access,
            java.lang.String name,
            java.lang.String descriptor,
            java.lang.String signature,
            java.lang.Object value)
    {
        // Convert array field to wrapper
        String desc = descriptor;
        if (descriptor.startsWith("[")) {
            desc = "L" + ArrayNameMapper.getUnifyingArrayWrapperDescriptor(descriptor) + ";";
        }

        return super.visitField(access, name, desc, signature, value);
    }

    @Override
    public MethodVisitor visitMethod(
            final int access,
            final String name,
            final String descriptor,
            final String signature,
            final String[] exceptions)
    {

        String desc = ArrayNameMapper.updateMethodDesc(descriptor);

        MethodVisitor mv = super.visitMethod(access, name, desc, signature, exceptions);

        return new ArrayWrappingMethodAdapter(mv, access, name, desc);
    }

}
