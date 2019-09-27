package org.aion.avm.core.rejection;

import java.nio.charset.StandardCharsets;

import org.aion.avm.core.ClassToolchain;
import org.aion.avm.core.miscvisitors.NamespaceMapper;
import org.aion.avm.core.miscvisitors.PreRenameClassAccessRules;
import i.PackageConstants;
import i.RuntimeAssertionError;
import org.objectweb.asm.AnnotationVisitor;
import org.objectweb.asm.Attribute;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.ModuleVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.TypePath;


/**
 * Does a simple read-only pass over the loaded class, ensuring it isn't doing anything it isn't allowed to do:
 * -uses bytecode in blacklist
 * -references class not in whitelist
 * -overrides methods which we will not support as the user may expect
 * 
 * When a violation is detected, throws the RejectedClassException.
 */
public class RejectionClassVisitor extends ClassToolchain.ToolChainClassVisitor {
    // This will probably change, in the future, but we currently will only parse Java10 (version 54) classes.
    private static final int SUPPORTED_CLASS_VERSION = 54;
    /*
     * This limit could probably be larger, since it really just needs to account for a type name length in 1 byte:
     * -probably 255 - 1 (for "L" prefix) - 3 (maximum array dimensions)
     * However, we want to leave this additional space for some potential future uses and we also don't expect that
     * these types will usually be longer than a few bytes long (API type references will probably be the longest)
     * because deployment tooling to shrink these identifiers reduces the deployment cost the user pays.
     */
    private static final int MAX_UTF8_NAME_LENGTH = 127;

    // The names of the classes that the user defined in their JAR (note:  this does NOT include interfaces).
    private final PreRenameClassAccessRules preRenameClassAccessRules;
    private final NamespaceMapper namespaceMapper;
    private final boolean preserveDebuggability;

    public RejectionClassVisitor(PreRenameClassAccessRules preRenameClassAccessRules, NamespaceMapper namespaceMapper, boolean preserveDebuggability) {
        super(Opcodes.ASM6);
        this.preRenameClassAccessRules = preRenameClassAccessRules;
        this.namespaceMapper = namespaceMapper;
        this.preserveDebuggability = preserveDebuggability;
    }

    @Override
    public void visit(int version, int access, String name, String signature, String superName, String[] interfaces) {
        // Make sure that this is the version we can understand.
        if (SUPPORTED_CLASS_VERSION != version) {
            RejectedClassException.unsupportedClassVersion(version);
        }
        if (name.getBytes(StandardCharsets.UTF_8).length > MAX_UTF8_NAME_LENGTH) {
            RejectedClassException.nameTooLong(name);
        }
        if (!this.preRenameClassAccessRules.canUserSubclass(superName)) {
            RejectedClassException.restrictedSuperclass(name, superName);
        }
        if(name.startsWith(PackageConstants.kPublicApiSlashPrefix)){
            RejectedClassException.unsupportedPackageName(name);
        }

        // Null the signature, since we don't use it and don't want to make sure it is safe.
        super.visit(version, access, name, null, superName, interfaces);
    }

    @Override
    public void visitSource(String source, String debug) {
        // Filter this.
    }

    @Override
    public ModuleVisitor visitModule(String name, int access, String version) {
        throw RuntimeAssertionError.unimplemented("AKI-106: This is never called");
    }

    @Override
    public AnnotationVisitor visitAnnotation(String descriptor, boolean visible) {
        // Filter this.
        return new RejectionAnnotationVisitor();
    }

    @Override
    public AnnotationVisitor visitTypeAnnotation(int typeRef, TypePath typePath, String descriptor, boolean visible) {
        // Filter this.
        return new RejectionAnnotationVisitor();
    }

    @Override
    public void visitAttribute(Attribute attribute) {
        // "Non-standard attributes" are not supported, so filter them.
    }

    @Override
    public FieldVisitor visitField(int access, String name, String descriptor, String signature, Object value) {
        // Note that the "value" field is only used for statics and can't be an object other than a String so we are safe with that.
        
        // Null the signature, since we don't use it and don't want to make sure it is safe.
        FieldVisitor fieldVisitor = super.visitField(access, name, descriptor, null, value);
        return new RejectionFieldVisitor(fieldVisitor);
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String descriptor, String signature, String[] exceptions) {
        // Check that they aren't trying to override a forbidden method.
        // finalize() is forbidden - should we check only the empty args descriptor or all methods with this name?
        if ("finalize".equals(name)) {
            RejectedClassException.forbiddenMethodOverride(name);
        }
        
        // Check that this method isn't synchronized, since we don't allow monitor operations.
        if (0 != (Opcodes.ACC_SYNCHRONIZED & access)) {
            RejectedClassException.invalidMethodFlag(name, "ACC_SYNCHRONIZED");
        }
        
        // Null the signature, since we don't use it and don't want to make sure it is safe.
        MethodVisitor mv = super.visitMethod(access, name, descriptor, null, exceptions);
        return new RejectionMethodVisitor(mv, this.preRenameClassAccessRules, this.namespaceMapper, this.preserveDebuggability);
    }
}
