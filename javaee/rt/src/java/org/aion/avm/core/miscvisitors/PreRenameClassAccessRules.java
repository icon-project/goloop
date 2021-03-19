package org.aion.avm.core.miscvisitors;

import i.PackageConstants;
import i.RuntimeAssertionError;
import org.aion.avm.core.NodeEnvironment;
import org.aion.avm.utilities.Utilities;
import score.UserRevertException;

import java.util.Set;
import java.util.stream.Collectors;


/**
 * Handles the common, high-level class identification questions asked by UserClassMappingVisitor, RejectionClassVisitor, and RejectionMethodVisitor.
 * As the class name implies, this always acts on pre-renamed classes.
 * Additionally, the entire interface operates on slash-style class names (since these are used within ASM visitors).
 */
public class PreRenameClassAccessRules {
    // This is the hard-coded list of classes, from the JCL, which we allow the user code to subclass.
    private static final Set<String> SUBCLASS_ALLOWLIST_SLASH_NAMES = Set.of(
            Enum.class.getName(),
            Exception.class.getName(),
            Object.class.getName(),
            RuntimeException.class.getName(),
            Throwable.class.getName(),
            UserRevertException.class.getName()
    ).stream().map(Utilities::fullyQualifiedNameToInternalName).collect(Collectors.toSet());

    private final Set<String> userDefinedSlashClassesOnly;
    private final Set<String> userDefinedSlashClassesAndInterfaces;

    public PreRenameClassAccessRules(Set<String> preRenameUserDefinedDotClassesOnly,
                                     Set<String> preRenameUserDefinedDotClassesAndInterfaces) {
        this.userDefinedSlashClassesOnly = preRenameUserDefinedDotClassesOnly.stream()
                .map(Utilities::fullyQualifiedNameToInternalName).collect(Collectors.toSet());
        this.userDefinedSlashClassesAndInterfaces = preRenameUserDefinedDotClassesAndInterfaces.stream()
                .map(Utilities::fullyQualifiedNameToInternalName).collect(Collectors.toSet());
    }

    /**
     * @param slashName The slash-style class name.
     * @return True if user-code can subclass the given class.
     */
    public boolean canUserSubclass(String slashName) {
        RuntimeAssertionError.assertTrue(!slashName.contains("."));
        return internalIsUserDefinedClassOnly(slashName)
                || internalIsJclSubclassAllowlist(slashName);
    }

    /**
     * @param slashName The slash-style class name.
     * @return True if user-code can access (invoke against, for example) the given class.
     */
    public boolean canUserAccessClass(String slashName) {
        RuntimeAssertionError.assertTrue(!slashName.contains("."));
        return internalIsUserDefinedClassOrInterface(slashName)
                || internalIsJclClass(slashName)
                || internalIsArray(slashName)
                || internalIsApiClass(slashName);
    }

    /**
     * @param slashName The slash-style class name.
     * @return True if this is a class or interface defined by the user's code.
     */
    public boolean isUserDefinedClassOrInterface(String slashName) {
        RuntimeAssertionError.assertTrue(!slashName.contains("."));
        return internalIsUserDefinedClassOrInterface(slashName);
    }

    /**
     * @param slashName The slash-style class name.
     * @return True if this is a JCL class which user code can access (will be mapped into shadow space).
     */
    public boolean isJclClass(String slashName) {
        RuntimeAssertionError.assertTrue(!slashName.contains("."));
        return internalIsJclClass(slashName);
    }

    /**
     * @param slashName The slash-style class name.
     * @return True if this class is defined as part of the public API.
     */
    public boolean isApiClass(String slashName) {
        RuntimeAssertionError.assertTrue(!slashName.contains("."));
        return internalIsApiClass(slashName);
    }

    private boolean internalIsUserDefinedClassOnly(String slashName) {
        return this.userDefinedSlashClassesOnly.contains(slashName);
    }

    private boolean internalIsUserDefinedClassOrInterface(String slashName) {
        return this.userDefinedSlashClassesAndInterfaces.contains(slashName);
    }

    private boolean internalIsJclClass(String slashName) {
        return NodeEnvironment.singleton.isClassFromJCL(slashName);
    }

    private boolean internalIsArray(String slashName) {
        return (0 == slashName.indexOf("["));
    }

    private boolean internalIsApiClass(String slashName) {
        return slashName.startsWith(PackageConstants.kPublicApiSlashPrefix);
    }

    private boolean internalIsJclSubclassAllowlist(String slashName) {
        return SUBCLASS_ALLOWLIST_SLASH_NAMES.contains(slashName);
    }
}
