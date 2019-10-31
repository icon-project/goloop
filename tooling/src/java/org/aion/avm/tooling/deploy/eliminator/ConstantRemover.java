package org.aion.avm.tooling.deploy.eliminator;

import org.aion.avm.userlib.abi.ABIDecoder;
import org.aion.avm.userlib.abi.ABIEncoder;
import org.aion.avm.userlib.abi.ABIException;
import org.aion.avm.userlib.abi.ABIStreamingEncoder;
import org.aion.avm.utilities.JarBuilder;
import org.aion.avm.utilities.Utilities;
import org.objectweb.asm.*;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Map;
import java.util.jar.JarInputStream;

/**
 *  Removes the ABIException messages (String constants) from ABIEncoder, ABIDecoder, ABIStreamingEncoder
 *  These messages are not useful once the contract has been deployed, but they consume energy as String constants
 */
public class ConstantRemover {

    private static List<String> classesToVisit = new ArrayList<>(Arrays.asList(
            ABIEncoder.class.getName(),
            ABIDecoder.class.getName(),
            ABIStreamingEncoder.class.getName()));

    public static byte[] removeABIExceptionMessages(byte[] jarBytes) throws IOException {
        JarInputStream jarReader = new JarInputStream(new ByteArrayInputStream(jarBytes), true);
        Map<String, byte[]> classMap = Utilities.extractClasses(jarReader, Utilities.NameStyle.DOT_NAME);
        for(String className : classesToVisit){
            if(classMap.containsKey(className)){
                classMap.put(className, updateABIException(classMap.get(className)));
            }
        }

        String mainClassName = (Utilities.extractMainClassName(jarReader, Utilities.NameStyle.DOT_NAME));
        byte[] mainClassBytes = classMap.get(mainClassName);
        classMap.remove(mainClassName);
        return JarBuilder.buildJarForExplicitClassNamesAndBytecode(mainClassName, mainClassBytes, classMap);
    }

    private static byte[] updateABIException(byte[] classBytecode) {
        ClassReader cr = new ClassReader(classBytecode);
        ClassWriter writer = new ClassWriter(ClassWriter.COMPUTE_FRAMES | ClassWriter.COMPUTE_MAXS);

        cr.accept(new ClassVisitor(Opcodes.ASM5, writer) {
            @Override
            public MethodVisitor visitMethod(int access, String name, String descriptor, String signature, String[] exceptions) {
                String ABIExceptionClassName = Utilities.fulllyQualifiedNameToInternalName(ABIException.class.getName());

                MethodVisitor visitor = super.visitMethod(access, name, descriptor, signature, exceptions);
                return new MethodVisitor(Opcodes.ASM6, visitor) {
                    @Override
                    public void visitLdcInsn(Object value) {
                        // remove string literals
                        if (!(value instanceof String)) {
                            super.visitLdcInsn(value);
                        }
                    }

                    @Override
                    public void visitMethodInsn(int opcode, String owner, String name, String descriptor, boolean isInterface) {
                        if (opcode == Opcodes.INVOKESPECIAL && owner.equals(ABIExceptionClassName)) {
                            if(!descriptor.equals("(Ljava/lang/String;)V")){
                                throw new AssertionError("Unexpected ABIException descriptor.");
                            }
                            super.visitMethodInsn(opcode, owner, name, "()V", isInterface);
                        } else {
                            super.visitMethodInsn(opcode, owner, name, descriptor, isInterface);
                        }
                    }
                };
            }
        }, 0);
        return writer.toByteArray();
    }
}
