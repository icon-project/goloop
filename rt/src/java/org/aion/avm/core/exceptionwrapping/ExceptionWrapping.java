package org.aion.avm.core.exceptionwrapping;

import i.RuntimeAssertionError;
import org.aion.avm.core.ClassToolchain;
import org.aion.avm.core.classgeneration.StubGenerator;
import org.aion.avm.core.types.ClassHierarchy;
import org.aion.avm.core.types.GeneratedClassConsumer;
import org.aion.avm.core.types.CommonType;
import i.Helper;
import i.PackageConstants;
import org.objectweb.asm.Label;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;

import java.util.HashSet;
import java.util.Set;


public class ExceptionWrapping extends ClassToolchain.ToolChainClassVisitor {
    private final GeneratedClassConsumer generatedClassesSink;

    private final ClassHierarchy classHierarchy;
    private final Set<String> jclExceptionsSlashStyle;

    public ExceptionWrapping(GeneratedClassConsumer generatedClassesSink, ClassHierarchy classHierarchy) {
        super(Opcodes.ASM6);
        this.generatedClassesSink = generatedClassesSink;
        this.classHierarchy = classHierarchy;
        this.jclExceptionsSlashStyle = fetchAllJavaLangExceptionsSlashStyle();

    }

    @Override
    public void visit(int version, int access, String name, String signature, String superName, String[] interfaces) {
        super.visit(version, access, name, signature, superName, interfaces);
        
        // Note that we want to decide whether or not to generate a wrapper for this class (added to the "generatedClasses" out parameter), at this point.
        // We do this by walking the hierarchy backward until we get to the java.lang and then seeing if we walk to "java.lang.Throwable" before
        // "java.lang.Object".
        String classDotName = name.replaceAll("/", ".");

        if (this.classHierarchy.isDescendantOfClass(classDotName, CommonType.SHADOW_THROWABLE.dotName)) {
            // Generate our handler for this.
            String reparentedName = ExceptionWrapperNameMapper.slashWrapperNameForClassName(name);
            String reparentedSuperName = ExceptionWrapperNameMapper.slashWrapperNameForClassName(superName);
            byte[] wrapperBytes = StubGenerator.generateWrapperClass(reparentedName, reparentedSuperName);
            generatedClassesSink.accept(reparentedSuperName, reparentedName, wrapperBytes);
        }
    }

    public MethodVisitor visitMethod(
            final int access,
            final String name,
            final String descriptor,
            final String signature,
            final String[] exceptions) {

        MethodVisitor mv = super.visitMethod(access, name, descriptor, signature, exceptions);

        return new MethodVisitor(Opcodes.ASM6, mv) {
            // We are going to collect the target labels of the exception handlers, so we can inject our instrumentation.
            private final Set<Label> catchTargets = new HashSet<>();
            // When we match a label as a target, we set this flag to inject our call-out before the first bytecode.
            private boolean isTargetFrame = false;

            @Override
            public void visitLabel(Label label) {
                super.visitLabel(label);

                // Be careful that we could have multiple labels at the same bytecode so saturate to true on match.
                if (this.catchTargets.contains(label)) {
                    this.isTargetFrame = true;
                }
            }
            @Override
            public void visitFrame(int type, int nLocal, Object[] local, int nStack, Object[] stack) {
                super.visitFrame(type, nLocal, local, nStack, stack);

                // If this is the target frame, then this is the last callback before the first bytecode.
                // This means, once we call the super visitFrame, we can now inject the call-out.
                if (this.isTargetFrame) {
                    // We need to build the call to the unwrap helper, followed by the checkcast.
                    // (we know that there is exactly one element on the stack, since this is the beginning of a catch block.
                    String castType = (String) stack[0];
                    String methodName = "unwrapThrowable";

                    // SHADOW NOTES:
                    // Note that this java/lang/Throwable MUST not be rewritten as another type since that is literally what is on the operand
                    // stack at the beginning of an exception handler.
                    // On the other hand, the returned java/lang/Object _should_ be rewritten as the shadow type, but we are safe if it isn't.
                    String methodDescriptor = "(Ljava/lang/Throwable;)L" + PackageConstants.kShadowSlashPrefix + "java/lang/Object;";

                    // The call-out will actually return the base object class in our environment so we need to cast.
                    super.visitMethodInsn(Opcodes.INVOKESTATIC, Helper.RUNTIME_HELPER_NAME, methodName, methodDescriptor, false);

                    // The cast will give us exactly what the previous path thought was the common class of any multi-catch options.
                    // (without this, the verifier will complain about the operand stack containing the wrong type when used).
                    // Note that this type also needs to be translated into our namespace, since we won't be giving the user direct access to
                    // real exceptions.
                    boolean isPreRenameJavaException = jclExceptionsSlashStyle.contains(castType);

                    // We only expect to find java.lang.Throwable by this point when it is synthesized in finally blocks.
                    // Everything else should have already been renamed.
                    if (isPreRenameJavaException) {
                        RuntimeAssertionError.assertTrue(castType.equals("java/lang/Throwable"));
                    }

                    String mappedCastType = isPreRenameJavaException
                            ? PackageConstants.kShadowSlashPrefix + castType
                            : castType;
                    super.visitTypeInsn(Opcodes.CHECKCAST, mappedCastType);
                }
                this.isTargetFrame = false;
            }
            @Override
            public void visitTryCatchBlock(Label start, Label end, Label handler, String type) {
                // Our special-handling is only for non-finally blocks.
                if (null != type) {
                    if (type.startsWith(PackageConstants.kShadowSlashPrefix)) {
                        // If this was trying to catch anything the shadow space, catch the underlying type (since the JVM might generate it)
                        // as well as our wrapper of this type (in case the user threw it from their code).
                        String strippedClass = type.substring(PackageConstants.kShadowSlashPrefix.length());
                        super.visitTryCatchBlock(start, end, handler, strippedClass);
                        String wrapperType = ExceptionWrapperNameMapper.slashWrapperNameForClassName(type);
                        super.visitTryCatchBlock(start, end, handler, wrapperType);
                    } else {
                        // This is user-defined (or should have been stripped, earlier) so replace it with the appropriate wrapper type.
                        String wrapperType = ExceptionWrapperNameMapper.slashWrapperNameForClassName(type);
                        super.visitTryCatchBlock(start, end, handler, wrapperType);
                    }
                } else {
                    // Finally, just pass through.
                    super.visitTryCatchBlock(start, end, handler, type);
                }

                // We also need to record the handler label so we know it is a catch block, in visitLabel, above.
                this.catchTargets.add(handler);
            }
            @Override
            public void visitInsn(int opcode) {
                // If this is athrow, we need to wrap whatever we had with an actual exception (after shadowing, nothing we get here will be an
                // actual exception - even java/lang exceptions will be exposed to the user's code as some sort of stub object from our domain).
                // This must happen BEFORE the athrow opcode is emitted, for obvious reasons.
                if (Opcodes.ATHROW == opcode) {
                    String methodName = "wrapAsThrowable";

                    // SHADOW NOTES:
                    // Note that this java/lang/Throwable MUST NOT be rewritten as another type since that is literally what must be on the
                    // operand stack when we call athrow.
                    // On the other hand, the java/lang/Object _should_ be rewritten as the shadow type, but we are safe if it isn't.
                    String methodDescriptor = "(L" + PackageConstants.kShadowSlashPrefix + "java/lang/Object;)Ljava/lang/Throwable;";

                    // The call-out will actually return the base object class in our environment so we need to cast.
                    super.visitMethodInsn(Opcodes.INVOKESTATIC, Helper.RUNTIME_HELPER_NAME, methodName, methodDescriptor, false);
                }
                super.visitInsn(opcode);
            }
        };
    }

    private Set<String> fetchAllJavaLangExceptionsSlashStyle() {
        Set<String> javaLangExceptions = new HashSet<>();
        for (CommonType type : CommonType.values()) {
            if (type.isShadowException) {
                javaLangExceptions.add(type.dotName.substring(PackageConstants.kShadowDotPrefix.length()).replaceAll("\\.", "/"));
            }
        }
        return javaLangExceptions;
    }
}
