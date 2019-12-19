package org.aion.avm.core.instrument;

import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassVisitor;
import org.objectweb.asm.ClassWriter;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;


/**
 * A arraywrapper over our common ASM routines.
 *
 * This class has no explicit design, as it is still evolving.
 */
public class ClassRewriter  {
    /**
     * Rewrites the given class, changing the named method by calling replacer.  Note that this will still succeed
     * even if the method is not found.
     *
     * @param classBytes The raw bytes of the class to modify.
     * @param methodName The method to replace.
     * @param replacer The callback to invoke to build the replacement method.
     * @return The raw bytes of the updated class.
     */
    public static byte[] rewriteOneMethodInClass(byte[] classBytes, String methodName, IMethodReplacer replacer, int computeFrameFlag) {
        ClassWriter cw = new ClassWriter(computeFrameFlag);
        FullClassVisitor adapter = new FullClassVisitor(cw, methodName, replacer);

        ClassReader cr = new ClassReader(classBytes);
        cr.accept(adapter, ClassReader.SKIP_DEBUG);

        return cw.toByteArray();
    }


    /**
     * A helper class used internally, by rewriteOneMethodInClass.
     */
    private static class FullClassVisitor extends ClassVisitor implements Opcodes {
        private final String methodName;
        private final IMethodReplacer replacer;

        public FullClassVisitor(ClassVisitor cv, String methodName, IMethodReplacer replacer) {
            super(Opcodes.ASM6, cv);
            this.methodName = methodName;
            this.replacer = replacer;
        }

        @Override
        public MethodVisitor visitMethod(int access, String name, String descriptor, String signature, String[] exceptions) {
            MethodVisitor resultantVisitor = null;
            if (this.methodName.equals(name)) {
                // This is the method we want to replace.
                // Just pass in a null signature, instead of updating it (JVM spec 4.3.4: "This kind of type information is needed to support reflection and debugging, and by a Java compiler").
                MethodVisitor originalVisitor = super.visitMethod(access & ~ACC_NATIVE, name, descriptor, null, exceptions);
                ReplacedMethodVisitor replacedVisitor = new ReplacedMethodVisitor(originalVisitor, this.replacer);

                // Note that we need to explicitly call the visitCode on the replaced visitory if we have converted it from native to bytecode.
                if (0 != (access & ACC_NATIVE)) {
                    replacedVisitor.visitCode();
                }
                resultantVisitor = replacedVisitor;
            } else {
                // In this case, we basically just want to pass this through.
                // Just pass in a null signature, instead of updating it (JVM spec 4.3.4: "This kind of type information is needed to support reflection and debugging, and by a Java compiler").
                resultantVisitor = super.visitMethod(access, name, descriptor, null, exceptions);
            }
            return resultantVisitor;
        }
    }


    /**
     * A helper class used internally, by FullClassVisitor.
     */
    private static class ReplacedMethodVisitor extends MethodVisitor implements Opcodes {
        private final MethodVisitor target;
        private final IMethodReplacer replacer;

        public ReplacedMethodVisitor(MethodVisitor target, IMethodReplacer replacer) {
            super(Opcodes.ASM6, null);
            this.target = target;
            this.replacer = replacer;
        }

        @Override
        public void visitCode() {
            this.replacer.populatMethod(this.target);
        }
    }


    /**
     * The interface we call back into to actually build the replacement bytecode for a method.
     * Note that this will probably evolve since it is currently a pretty leaky abstraction:  pushes MethodVisitor knowledge and responsibility to
     * implementation.
     */
    public static interface IMethodReplacer {
        void populatMethod(MethodVisitor visitor);
    }
}
