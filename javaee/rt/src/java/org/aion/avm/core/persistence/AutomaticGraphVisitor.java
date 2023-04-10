package org.aion.avm.core.persistence;

import org.aion.avm.core.ClassToolchain;
import i.RuntimeAssertionError;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.tree.MethodNode;
import org.objectweb.asm.tree.analysis.Analyzer;
import org.objectweb.asm.tree.analysis.AnalyzerException;
import org.objectweb.asm.tree.analysis.Frame;


/**
 * This visitor is responsible for reshaping the contract code such that our "automatic graph" persistence design can be applied.
 * Specifically, this means the following transformations:
 * 1)  Add a special constructor, which cannot already be present, just calling its superclass counterpart.
 * 2)  Remove "final" from all fields (at least instance fields - we may be able to treat static fields differently).
 * 3)  Prepend all PUTFIELD/GETFIELD instructions with a call to "lazyLoad()" on the receiver object (unless "this" in a constructor).
 * 
 * Note that this transformation doesn't depend on the persistence model being applied.  So long as "lazyLoad()" is a safe no-op,
 * there is no harm in enabling this without the corresponding persistence logic.
 * This should probably be put late in the pipeline since these transformations are substantial, and could change energy and stack
 * accounting in pretty large ways for what are essentially our own implementation details.
 */
public class AutomaticGraphVisitor extends ClassToolchain.ToolChainClassVisitor {
    private static final String CLINIT_NAME = "<clinit>";
    private static final String INIT_NAME = "<init>";
    // The special constructor takes (Void ignore, int readIndex).
    private static final String SPECIAL_CONSTRUCTOR_DESCRIPTOR = "(Ljava/lang/Void;I)V";

    private boolean isInterface;
    private String className;
    private String superClassName;

    public AutomaticGraphVisitor() {
        super(Opcodes.ASM7);
    }

    @Override
    public void visit(int version, int access, String name, String signature, String superName, String[] interfaces) {
        // Note that we don't want to change interfaces - clearly, they have no constructors.
        this.isInterface = (0 != (Opcodes.ACC_INTERFACE & access));
        // We need the class name for the analyzer.
        this.className = name;
        // We just want to extract the superclass name.
        this.superClassName = superName;
        super.visit(version, access, name, signature, superName, interfaces);
    }

    @Override
    public FieldVisitor visitField(int access, String name, String descriptor, String signature, Object value) {
        // Filter out the "final" from all fields.
        // (note that we may way to skip this, for statics, and exclude them from the serialization system).
        int newAccess = (~Opcodes.ACC_FINAL) & access; 
        return super.visitField(newAccess, name, descriptor, signature, value);
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String descriptor, String signature, String[] exceptions) {
        MethodVisitor downstream = super.visitMethod(access, name, descriptor, signature, exceptions);
        MethodVisitor visitor = null;
        
        // There are 3 distinct special-cases we need to handle, here.
        if (CLINIT_NAME.equals(name)) {
            // 1) If this is the <clinit>, we don't want to inject the lazyLoad calls (nothing visible there could be a stub).
            visitor = downstream;
        } else if (INIT_NAME.equals(name)) {
            // 2) If this is an <init> we need to ensure that we don't lazyLoad() the "this" pointer due to outer class references (issue-156).
            visitor = new MethodNode(Opcodes.ASM7, access, name, descriptor, signature, exceptions) {
                @Override
                public void visitEnd() {
                    super.visitEnd();
                    
                    // The MethodNode is fully populated, so we can now analyze it.
                    // We use a custom interpreter which only worries about "this" or "not this".
                    Analyzer<ConstructorThisInterpreter.ThisValue> analyzer = new Analyzer<>(new ConstructorThisInterpreter());
                    try {
                        // We want to tear apart the frames and check if the top of the stack, at each bytecode, is "this" (those cases can be
                        // safely skipped)..
                        Frame<ConstructorThisInterpreter.ThisValue>[] frames = analyzer.analyze(AutomaticGraphVisitor.this.className, this);
                        StackThisTracker tracker = new StackThisTracker(frames);
                        // Tell the LazyLoadingMethodVisitor about these locations where "lazyLoad()" can be skipped and have it process the
                        // method.
                        this.accept(new LazyLoadingMethodVisitor(downstream, tracker));
                    } catch (AnalyzerException e) {
                        // Such an error should have been handled before we got this far.
                        throw RuntimeAssertionError.unexpected(e);
                    }
                }
            };
        } else {
            // 3) Otherwise, insert lazyLoad() calls before any field access.
            visitor = new LazyLoadingMethodVisitor(downstream, null);
        }
        return visitor;
    }

    @Override
    public void visitEnd() {
        // If this isn't an interface, define the special constructor here.
        if (!this.isInterface) {
            // This logic is similar to StubGenerator.
            MethodVisitor methodVisitor = super.visitMethod(Opcodes.ACC_PUBLIC, INIT_NAME, SPECIAL_CONSTRUCTOR_DESCRIPTOR, null, null);
            methodVisitor.visitCode();
            methodVisitor.visitVarInsn(Opcodes.ALOAD, 0);
            methodVisitor.visitVarInsn(Opcodes.ALOAD, 1);
            methodVisitor.visitVarInsn(Opcodes.ILOAD, 2);
            methodVisitor.visitMethodInsn(Opcodes.INVOKESPECIAL, this.superClassName, INIT_NAME, SPECIAL_CONSTRUCTOR_DESCRIPTOR, false);
            methodVisitor.visitInsn(Opcodes.RETURN);
            methodVisitor.visitMaxs(4, 4);
            methodVisitor.visitEnd();
        }
        super.visitEnd();
    }
}
