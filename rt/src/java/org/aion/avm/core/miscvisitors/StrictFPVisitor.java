package org.aion.avm.core.miscvisitors;

import org.aion.avm.core.ClassToolchain;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;


/**
 * This visitor is the simplest one we have.  All it does is apply the "strictfp" modifier to every method found.
 * Note that, although the Java language interprets this modifier being attached to a class, at the class file level,
 * it is a per-method modifier.
 * 
 * AKI-418:  This application is mostly being done for completeness, as it appears as though there may no longer
 * be hardware on which the JVM provides a "non-strict" mode.  Taking the x86 architecture as an example, using the
 * x87 FPU would produce non-strict values since it had a greater precision than IEEE 754.  However, the JVM now
 * implements floating-point operations via SSE, on that architecture, which natively exposes IEEE 754.
 * However, we still apply the flag since it is required to be strictly correct in all cases.
 */
public class StrictFPVisitor extends ClassToolchain.ToolChainClassVisitor {
    public StrictFPVisitor() {
        super(Opcodes.ASM6);
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String descriptor, String signature, String[] exceptions) {
        // Just apply the strict flag to the method and return the standard visitor.
        // Note that abstract methods cannot also be strict (section 4.6 of JVM spec).
        int accessFlags = (Opcodes.ACC_ABSTRACT != (access & Opcodes.ACC_ABSTRACT))
                ? (access | Opcodes.ACC_STRICT)
                : access;
        return super.visitMethod(accessFlags, name, descriptor, signature, exceptions);
    }
}
