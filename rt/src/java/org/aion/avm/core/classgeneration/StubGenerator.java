package org.aion.avm.core.classgeneration;

import org.aion.avm.core.miscvisitors.NamespaceMapper;
import i.PackageConstants;
import org.objectweb.asm.ClassVisitor;
import org.objectweb.asm.ClassWriter;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;


/**
 * Used to generate simple "stub" classes, dynamically.
 * These classes come in 2 flavours:  wrappers and exceptions.
 * Wrappers have only 1 constructor, taking only 1 "java/lang/Object" parameter.
 * Exceptions, however, have 4 constructors:
 * -empty
 * -String
 * -String, Throwable
 * -Throwable
 * 
 * These classes are useful for cases where we don't need special behaviour, but we do need a type (exception catching, parametric polymorphism, etc).
 * Note that class names here are always in the slash style:  "java/lang/Object".
 */
public class StubGenerator {
    private static final int CLASS_VERSION = 54;
    private static final String INIT_NAME = "<init>";
    private static final String ONE_ARG_DESCRIPTOR = "(Ljava/lang/Object;)V";
    
    /**
     * Generates and returns the bytecode for a wrapper class.
     * 
     * @param name The name of the class to generate.
     * @param superName The name of the superclass.
     * @return The bytecode for the new class.
     */
    public static byte[] generateWrapperClass(String name, String superName) {
        ClassWriter out = new ClassWriter(ClassWriter.COMPUTE_FRAMES | ClassWriter.COMPUTE_MAXS);
        
        // This class only exists to be a type - the superclasses always do everything.
        // (common access for all classes we generate - public and "super", meaning post-1.1 invokestatic).
        int access = Opcodes.ACC_PUBLIC | Opcodes.ACC_SUPER;
        // We ignore generics, so null signature.
        String signature = null;
        // We implement no interfaces.
        String[] interfaces = new String[0];
        out.visit(CLASS_VERSION, access, name, signature, superName, interfaces);
        
        // Generate the singular constructor.
        MethodVisitor methodVisitor = out.visitMethod(Opcodes.ACC_PUBLIC, INIT_NAME, ONE_ARG_DESCRIPTOR, null, null);
        methodVisitor.visitCode();
        methodVisitor.visitVarInsn(Opcodes.ALOAD, 0);
        methodVisitor.visitVarInsn(Opcodes.ALOAD, 1);
        methodVisitor.visitMethodInsn(Opcodes.INVOKESPECIAL, superName, INIT_NAME, ONE_ARG_DESCRIPTOR, false);
        methodVisitor.visitInsn(Opcodes.RETURN);
        methodVisitor.visitMaxs(2, 2);
        methodVisitor.visitEnd();
        
        // Finish this and dump the bytes.
        out.visitEnd();
        return out.toByteArray();
    }
    
    /**
     * Generates and returns the bytecode for an exception class.
     * 
     * @param name The name of the class to generate.
     * @param superName The name of the superclass.
     * @return The bytecode for the new class.
     */
    public static byte[] generateExceptionClass(String name, String superName) {
        ClassWriter out = new ClassWriter(ClassWriter.COMPUTE_FRAMES | ClassWriter.COMPUTE_MAXS);
        
        // This class only exists to be a type - the superclasses always do everything.
        // (common access for all classes we generate - public and "super", meaning post-1.1 invokestatic).
        int access = Opcodes.ACC_PUBLIC | Opcodes.ACC_SUPER;
        // We ignore generics, so null signature.
        String signature = null;
        // We implement no interfaces.
        String[] interfaces = new String[0];
        out.visit(CLASS_VERSION, access, name, signature, superName, interfaces);
        
        // Generate the constructors.
        populateExceptionConstructors(out, superName);
        
        // Finish this and dump the bytes.
        out.visitEnd();
        return out.toByteArray();
    }
    
    /**
     * Generates and returns the bytecode for a "legacy-style" exception class.  That is, the kind which have their own getCause and getException implementations.
     * In our implementation, we will strictly wrap the super-class, and append a getException method.
     * 
     * @param name The name of the class to generate.
     * @param superName The name of the superclass.
     * @return The bytecode for the new class.
     */
    public static byte[] generateLegacyExceptionClass(String name, String superName) {
        ClassWriter out = new ClassWriter(ClassWriter.COMPUTE_FRAMES | ClassWriter.COMPUTE_MAXS);
        
        // This class only exists to be a type - the superclasses always do everything.
        // (common access for all classes we generate - public and "super", meaning post-1.1 invokestatic).
        int access = Opcodes.ACC_PUBLIC | Opcodes.ACC_SUPER;
        // We ignore generics, so null signature.
        String signature = null;
        // We implement no interfaces.
        String[] interfaces = new String[0];
        out.visit(CLASS_VERSION, access, name, signature, superName, interfaces);
        
        // Generate the constructors.
        populateExceptionConstructors(out, superName);
        
        // Generate the getException method.
        {
            // NOTE:  These methods need to exist with the UserClassMappingVisitor.mapMethodName("" prefixes, since the shadow library is post-shadow.
            String returnThrowable = "()L" + PackageConstants.kShadowSlashPrefix + "java/lang/Throwable;";
            MethodVisitor methodVisitor = out.visitMethod(Opcodes.ACC_PUBLIC, NamespaceMapper.mapMethodName("getException"), returnThrowable, null, null);
            methodVisitor.visitCode();
            methodVisitor.visitVarInsn(Opcodes.ALOAD, 0);
            methodVisitor.visitMethodInsn(Opcodes.INVOKEVIRTUAL, name, NamespaceMapper.mapMethodName("getCause"), returnThrowable, false);
            methodVisitor.visitInsn(Opcodes.ARETURN);
            methodVisitor.visitMaxs(1, 1);
            methodVisitor.visitEnd();
        }
        
        // Finish this and dump the bytes.
        out.visitEnd();
        return out.toByteArray();
    }
    
    
    private static void populateExceptionConstructors(ClassVisitor out, String superName) {
        MethodVisitor methodVisitor = null;
        // Generate the () constructor.
        {
            String noArgDescriptor = "()V";
            methodVisitor = out.visitMethod(Opcodes.ACC_PUBLIC, INIT_NAME, noArgDescriptor, null, null);
            methodVisitor.visitCode();
            methodVisitor.visitVarInsn(Opcodes.ALOAD, 0);
            methodVisitor.visitMethodInsn(Opcodes.INVOKESPECIAL, superName, INIT_NAME, noArgDescriptor, false);
            methodVisitor.visitInsn(Opcodes.RETURN);
            methodVisitor.visitMaxs(1, 1);
            methodVisitor.visitEnd();
        }
        
        // Generate the (String) constructor.
        {
            String oneStringDescriptor = "(L" + PackageConstants.kShadowSlashPrefix + "java/lang/String;)V";
            methodVisitor = out.visitMethod(Opcodes.ACC_PUBLIC, INIT_NAME, oneStringDescriptor, null, null);
            methodVisitor.visitCode();
            methodVisitor.visitVarInsn(Opcodes.ALOAD, 0);
            methodVisitor.visitVarInsn(Opcodes.ALOAD, 1);
            methodVisitor.visitMethodInsn(Opcodes.INVOKESPECIAL, superName, INIT_NAME, oneStringDescriptor, false);
            methodVisitor.visitInsn(Opcodes.RETURN);
            methodVisitor.visitMaxs(2, 2);
            methodVisitor.visitEnd();
        }
        
        // Generate the (String, Throwable) constructor.
        {
            String stringThrowableDescriptor = "(L" + PackageConstants.kShadowSlashPrefix + "java/lang/String;L" + PackageConstants.kShadowSlashPrefix + "java/lang/Throwable;)V";
            methodVisitor = out.visitMethod(Opcodes.ACC_PUBLIC, INIT_NAME, stringThrowableDescriptor, null, null);
            methodVisitor.visitCode();
            methodVisitor.visitVarInsn(Opcodes.ALOAD, 0);
            methodVisitor.visitVarInsn(Opcodes.ALOAD, 1);
            methodVisitor.visitVarInsn(Opcodes.ALOAD, 2);
            methodVisitor.visitMethodInsn(Opcodes.INVOKESPECIAL, superName, INIT_NAME, stringThrowableDescriptor, false);
            methodVisitor.visitInsn(Opcodes.RETURN);
            methodVisitor.visitMaxs(3, 3);
            methodVisitor.visitEnd();
        }
        
        // Generate the (Throwable) constructor.
        {
            String oneThrowableDescriptor = "(L" + PackageConstants.kShadowSlashPrefix + "java/lang/Throwable;)V";
            methodVisitor = out.visitMethod(Opcodes.ACC_PUBLIC, INIT_NAME, oneThrowableDescriptor, null, null);
            methodVisitor.visitCode();
            methodVisitor.visitVarInsn(Opcodes.ALOAD, 0);
            methodVisitor.visitVarInsn(Opcodes.ALOAD, 1);
            methodVisitor.visitMethodInsn(Opcodes.INVOKESPECIAL, superName, INIT_NAME, oneThrowableDescriptor, false);
            methodVisitor.visitInsn(Opcodes.RETURN);
            methodVisitor.visitMaxs(2, 2);
            methodVisitor.visitEnd();
        }
        
        // Generate the deserialization constructor:  "(Ljava/lang/Void;I)V".
        {
            String deserializationDescriptor = "(Ljava/lang/Void;I)V";
            methodVisitor = out.visitMethod(Opcodes.ACC_PUBLIC, INIT_NAME, deserializationDescriptor, null, null);
            methodVisitor.visitCode();
            methodVisitor.visitVarInsn(Opcodes.ALOAD, 0);
            methodVisitor.visitVarInsn(Opcodes.ALOAD, 1);
            methodVisitor.visitVarInsn(Opcodes.ILOAD, 2);
            methodVisitor.visitMethodInsn(Opcodes.INVOKESPECIAL, superName, INIT_NAME, deserializationDescriptor, false);
            methodVisitor.visitInsn(Opcodes.RETURN);
            methodVisitor.visitMaxs(3, 3);
            methodVisitor.visitEnd();
        }
    }
}
