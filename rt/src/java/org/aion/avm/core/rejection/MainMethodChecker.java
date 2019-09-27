package org.aion.avm.core.rejection;

import org.aion.avm.core.ClassToolchain;
import org.objectweb.asm.ClassWriter;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;


/**
 * Does a trivial read-only pass over the class, ensuring that it includes the correct main entry-point method:
 * "public static byte[] main()"
 * 
 * This is used to flag DApps being deployed as invalid if their main class doesn't have a main entry-point.
 */
public class MainMethodChecker extends ClassToolchain.ToolChainClassVisitor {
    private static final int MAIN_MODIFIERS = Opcodes.ACC_PUBLIC | Opcodes.ACC_STATIC;
    private static final String MAIN_METHOD_NAME = "main";
    private static final String MAIN_METHOD_DESCRIPTOR = "()[B";

    public static boolean checkForMain(byte[] bytecode) {
        MainMethodChecker methodChecker = new MainMethodChecker();
        // We don't actually want the output bytecode, in this case.
        try {
            new ClassToolchain.Builder(bytecode, 0)
                .addNextVisitor(methodChecker)
                .addWriter(new ClassWriter(0))
                .build()
                .runAndGetBytecode();
        } catch (Exception e) {
            throw new RejectedClassException("Error when reading main method of main class: " + e.getMessage());
        }
        return methodChecker.didFindMain;
    }


    private boolean didFindMain;

    private MainMethodChecker() {
        super(Opcodes.ASM6);
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String descriptor, String signature, String[] exceptions) {
        if (MAIN_METHOD_NAME.equals(name) && (MAIN_MODIFIERS == (access & MAIN_MODIFIERS)) && MAIN_METHOD_DESCRIPTOR.equals(descriptor)) {
            this.didFindMain = true;
        }
        return super.visitMethod(access, name, descriptor, signature, exceptions);
    }
}
