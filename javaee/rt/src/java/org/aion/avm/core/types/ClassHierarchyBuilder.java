package org.aion.avm.core.types;

import java.util.Set;
import org.aion.avm.core.ClassRenamer;
import org.aion.avm.core.NodeEnvironment;
import org.aion.avm.core.rejection.RejectedClassException;
import i.PackageConstants;

/**
 * A utility for building instances of a class hierarchy.
 *
 * This utility is the preferred way of constructing a hierarchy, especially since it ensures that
 * the verifier gets run and that the hierarchy was not left in a corrupt state.
 */
public final class ClassHierarchyBuilder {
    private ClassHierarchyVerifier verifier;
    private ClassRenamer classRenamer;
    private Set<ClassInformation> userClassInfos;
    private Set<ClassInformation> nonUserClassInfos;

    private boolean addShadowJcl = false;
    private boolean addArrays = false;
    private boolean addExceptions = false;
    private boolean addUserClasses = false;
    private boolean addNonUserClasses = false;

    /**
     * Constructs a new hierarchy that is pre-loaded with all of the shadow JCL classes.
     *
     * Note that preserveDebuggability is only used when adding user-defined classes to the hierarchy.
     * If the hierarchy being constructed contains no user-defined classes, then this value has no
     * impact and can safely be set to anything.
     */
    public ClassHierarchyBuilder() {
        this.verifier = new ClassHierarchyVerifier();
    }

    /**
     * Specifies that all of the shadow JCL classes will be added to the hierarchy.
     *
     * @return this builder.
     */
    public ClassHierarchyBuilder addShadowJcl() {
        this.addShadowJcl = true;
        return this;
    }

    /**
     * Specifies that all of the hand-written array wrapper classes will be added to the hierarchy.
     *
     * @return this builder.
     */
    public ClassHierarchyBuilder addHandwrittenArrayWrappers() {
        this.addArrays = true;
        return this;
    }

    /**
     * Specifies that all of the JCL exception classes will be added to the hierarchy.
     *
     * @return this builder.
     */
    public ClassHierarchyBuilder addPostRenameJclExceptions() {
        this.addExceptions = true;
        return this;
    }

    /**
     * Specifies that all of the classes represented by the provided class infos will be added to
     * the hierarchy as pre-rename user-defined classes.
     *
     * Note that user-defined classes are handled specially. This should be the set of all classes
     * defined uniquely by the user.
     *
     * If this special behaviour is unwanted, use {@code addNonUserDefinedClasses()}.
     *
     * @param classRenamer The class renamer utility.
     * @param classInfos The class infos representing the user-defined classes.
     * @return this builder.
     */
    public ClassHierarchyBuilder addPreRenameUserDefinedClasses(ClassRenamer classRenamer, Set<ClassInformation> classInfos) {
        this.addUserClasses = true;
        this.classRenamer = classRenamer;
        this.userClassInfos = classInfos;
        return this;
    }

    /**
     * Specifies that all of the classes represented by the provided class infos will be added to
     * the hierarchy as post-rename classes.
     *
     * @param classInfos The class infos representing the user-defined classes.
     * @return this builder.
     */
    public ClassHierarchyBuilder addPostRenameNonUserDefinedClasses(Set<ClassInformation> classInfos) {
        this.addNonUserClasses = true;
        this.nonUserClassInfos = classInfos;
        return this;
    }

    /**
     * Constructs the specified hierarchy and runs the verifier against it to ensure that it is
     * a complete hierarchy.
     *
     * Throws an exception if the verifier detects an inconsistency.
     *
     * @return The hierarchy.
     */
    public ClassHierarchy build() {

        // The most efficient way to construct the hierarchy is to begin with the shadow JCL if specified.
        ClassHierarchy hierarchy = (this.addShadowJcl) ? createHierarchyWithShadowJclClasses() : new ClassHierarchy();

        // Add any non-user-defined classes if specified as a set of class infos.
        if (this.addNonUserClasses) {
            addNonUserClasses(hierarchy);
        }

        // Add the user-defined classes if specified.
        if (this.addUserClasses) {
            addUserClasses(hierarchy);
        }

        // Add the handwritten array wrapper classes if specified.
        if (this.addArrays) {
            addHandwrittenArrayWrappersToHierarchy(hierarchy);
        }

        // Add the exception types if specified.
        if (this.addExceptions) {
            addPostRenameJclExceptionTypes(hierarchy);
        }

        // Finally, run the verifier against the hierarchy and if all is good return it.
        HierarchyVerificationResult result = this.verifier.verifyHierarchy(hierarchy);
        if (!result.success) {
            throw new RejectedClassException(result.getError());
        }
        return hierarchy;
    }

    /**
     * Constructs a new hierarchy with all the shadow JCL classes loaded into it.
     */
    private ClassHierarchy createHierarchyWithShadowJclClasses() {
        return NodeEnvironment.singleton.deepCopyOfClassHierarchy();
    }

    /**
     * Adds the handwritten array wrapper classes (as well as any classes referenced by them) to the
     * hierarchy.
     */
    private void addHandwrittenArrayWrappersToHierarchy(ClassHierarchy hierarchy) {
        for (CommonType type : CommonType.values()) {
            if (type.dotName.startsWith(PackageConstants.kArrayWrapperDotPrefix)) {
                hierarchy.add(ClassInformation.postRenameInfofrom(type));
            }
        }

        // IObjectArray is an internal type, not an array wrapper, so it didn't get added above.
        hierarchy.add(ClassInformation.postRenameInfofrom(CommonType.I_OBJECT_ARRAY));
    }

    /**
     * Adds the post-renamed JCL exception classes (as well as any classes referenced by them).
     */
    private void addPostRenameJclExceptionTypes(ClassHierarchy hierarchy) {
        for (CommonType type : CommonType.values()) {
            if (type.isShadowException) {
                hierarchy.addIfAbsent(ClassInformation.postRenameInfofrom(type));
            }
        }
    }

    /**
     * Adds the user-defined classes (as well as any classes referenced by them).
     *
     * This method also takes care of whether or not to rename these classes based on whether or not
     * we are in debug mode.
     */
    private void addUserClasses(ClassHierarchy hierarchy) {
        hierarchy.addPreRenameUserDefinedClasses(this.classRenamer, this.userClassInfos);
    }

    /**
     * Adds the non-user-defined classes.
     *
     * No renaming is performed.
     */
    private void addNonUserClasses(ClassHierarchy hierarchy) {
        for (ClassInformation classInfo : this.nonUserClassInfos) {
            hierarchy.add(classInfo);
        }
    }

}
