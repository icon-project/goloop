package org.aion.avm.core;

import org.aion.avm.core.miscvisitors.StringConstantCollectorVisitor;
import org.aion.avm.utilities.Utilities;

import i.Helper;
import i.PackageConstants;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassWriter;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;

import java.util.*;


/**
 * A helper class used to construct the bytecode for the constant class we will inject into the user's DApp.
 */
public class ConstantClassBuilder {
    private static final String kClinitName = "<clinit>";
    private static final int kClinitAccess = Opcodes.ACC_STATIC;
    private static final String kClinitDescriptor = "()V";

    private static final String kPostRenameStringDescriptor = "L" + PackageConstants.kShadowSlashPrefix + "java/lang/String;";
    private static final String kWrapStringMethodName = "wrapAsString";
    private static final String kWrapStringMethodDescriptor = "(Ljava/lang/String;)L" + PackageConstants.kShadowSlashPrefix + "java/lang/String;";

    /**
     * Builds the bytecode for the constant class which can replace the constants found in rawUserClasses.
     * 
     * @param constantClassName The name of the class to create.
     * @param rawUserClasses All the classes provided by the user, in no particular order.
     * @return The bytecode and contents of this constant class.
     */
    public static ConstantClassInfo buildConstantClassBytecodeForClasses(String constantClassName, Collection<byte[]> rawUserClasses) {
        // First, we want to coalesce all string constants in the DApp - avoids duplication, makes the specification far simpler, and avoids limitations of string constants in interfaces.
        StringConstantCollectorVisitor stringConstantCollector = new StringConstantCollectorVisitor();
        for (byte[] bytecode : rawUserClasses) {
            // Note that this pass is read-only but the ClassToolchain.Builder requires the inclusion of a writer (we don't use the bytecode so just use the default one).
            new ClassToolchain.Builder(bytecode, ClassReader.SKIP_DEBUG)
                    .addNextVisitor(stringConstantCollector)
                    .addWriter(new ClassWriter(0))
                    .build()
                    .runAndGetBytecode();
        }
        
        // Generate the constant class and the constant mapping function.
        List<String> sortedStringConstants = stringConstantCollector.sortedStringConstants();
        return ConstantClassBuilder.generateStringConstantClass(constantClassName, sortedStringConstants);
    }

    /**
     * Builds the bytecode for the constant class which contains the given sortedStringConstants.
     * 
     * @param constantClassName The name of the class to create.
     * @param sortedStringConstants The string constants to contain, in the order they should be assigned statics.
     * @return The bytecode and contents of this constant class.
     */
    public static ConstantClassInfo generateConstantClassForTest(String constantClassName, List<String> sortedStringConstants) {
        return ConstantClassBuilder.generateStringConstantClass(constantClassName, sortedStringConstants);
    }


    private static ConstantClassInfo generateStringConstantClass(String className, List<String> sortedStringConstants) {
        Map<String, String> constantToFieldMap = new HashMap<>();
        ClassWriter out = new ClassWriter(ClassWriter.COMPUTE_FRAMES);
        
        int classVersion = 54;
        // (note that this class doesn't deal with floats, but we add the STRICT option so some general tests are happy).
        int classAccess = Opcodes.ACC_PUBLIC | Opcodes.ACC_SUPER | Opcodes.ACC_STRICT;
        // We ignore generics, so null signature.
        String signature = null;
        // We implement no interfaces.
        String[] interfaces = new String[0];
        out.visit(classVersion, classAccess, className, signature, Utilities.fulllyQualifiedNameToInternalName(Object.class.getName()), interfaces);
        
        // Generate the static fields - not final since we need to load these from storage.
        int fieldAccess = Opcodes.ACC_PUBLIC | Opcodes.ACC_STATIC;
        for (int i = 0; i < sortedStringConstants.size(); ++i) {
            String staticName = "const_" + i;
            out.visitField(fieldAccess, staticName, kPostRenameStringDescriptor, null, null);
        }
        
        // Generate the <clinit>
        MethodVisitor clinitVisitor = out.visitMethod(kClinitAccess, kClinitName, kClinitDescriptor, null, null);
        // Prepend the ldc+invokestatic+putstatic setup of the synthesized constants.
        for (int i = 0; i < sortedStringConstants.size(); ++i) {
            String staticName = "const_" + i;
            String constant = sortedStringConstants.get(i);
            constantToFieldMap.put(constant, staticName);
            
            // load constant
            clinitVisitor.visitLdcInsn(constant);

            // wrap as shadow string
            clinitVisitor.visitMethodInsn(Opcodes.INVOKESTATIC, Helper.RUNTIME_HELPER_NAME, kWrapStringMethodName, kWrapStringMethodDescriptor, false);

            // set the field
            clinitVisitor.visitFieldInsn(Opcodes.PUTSTATIC, className, staticName, kPostRenameStringDescriptor);
        }
        clinitVisitor.visitInsn(Opcodes.RETURN);
        clinitVisitor.visitMaxs(1, 0);
        clinitVisitor.visitEnd();
        
        out.visitEnd();
        byte[] bytecode = out.toByteArray();
        return new ConstantClassInfo(bytecode, Collections.unmodifiableMap(constantToFieldMap));
    }


    public static class ConstantClassInfo {
        public final byte[] bytecode;
        public final Map<String, String> constantToFieldMap;
        
        private ConstantClassInfo(byte[] bytecode, Map<String, String> constantToFieldMap) {
            this.bytecode = bytecode;
            this.constantToFieldMap = constantToFieldMap;
        }
    }
}
