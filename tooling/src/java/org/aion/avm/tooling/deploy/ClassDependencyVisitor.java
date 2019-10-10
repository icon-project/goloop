package org.aion.avm.tooling.deploy;

import org.objectweb.asm.*;
import org.objectweb.asm.signature.SignatureReader;
import org.objectweb.asm.signature.SignatureVisitor;

// Note that inner classes are added as dependencies only if they are referenced
public class ClassDependencyVisitor extends ClassVisitor {

    private final DependencyCollector dependencyCollector;
    private final SignatureVisitor signatureVisitor;
    private boolean preserveDebugInfo;

    public ClassDependencyVisitor(SignatureVisitor signatureVisitor, DependencyCollector dependencyCollector, ClassWriter writer, boolean preserveDebugInfo) {
        super(Opcodes.ASM6, writer);
        this.dependencyCollector = dependencyCollector;
        this.signatureVisitor = signatureVisitor;
        this.preserveDebugInfo = preserveDebugInfo;
    }

    @Override
    public void visit(int version, int access, String name, String signature, String superName, String[] interfaces) {
        // Signature may be null if the class is not a generic one, and does not extend or implement generic classes or interfaces.
        if (signature == null) {
            dependencyCollector.addType(superName);
            if (interfaces != null) {
                for (String i : interfaces)
                    dependencyCollector.addType(i);
            }
        } else {
            new SignatureReader(signature).accept(signatureVisitor);
        }

        super.visit(version, access, name, signature, superName, interfaces);
    }

    @Override
    public FieldVisitor visitField(int access, String name, String descriptor, String signature, Object value) {

        if (signature == null) {
            dependencyCollector.addDescriptor(descriptor);
        } else {
            new SignatureReader(signature).acceptType(signatureVisitor);
        }
        return super.visitField(access, name, descriptor, signature, value);
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String descriptor, String signature, String[] exceptions) {

        if (signature == null) {
            dependencyCollector.addMethodDescriptor(descriptor);
        } else {
            new SignatureReader(signature).accept(signatureVisitor);
        }

        if (exceptions != null) {
            for (String ex : exceptions)
                dependencyCollector.addType(ex);
        }

        MethodVisitor mv = super.visitMethod(access, name, descriptor, signature, exceptions);
        return new MethodDependencyVisitor(mv, signatureVisitor, dependencyCollector, preserveDebugInfo);
    }
}
