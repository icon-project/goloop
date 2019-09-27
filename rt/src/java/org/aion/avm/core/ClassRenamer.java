package org.aion.avm.core;

import java.util.HashSet;
import java.util.Set;
import org.aion.avm.ArrayUtil;
import org.aion.avm.NameStyle;
import org.aion.avm.core.arraywrapping.ArrayNameMapper;
import org.aion.avm.core.rejection.RejectedClassException;
import org.aion.avm.core.types.CommonType;
import i.PackageConstants;
import i.RuntimeAssertionError;

public final class ClassRenamer {
    // ClassRenamer doesn't make use of it, but a convenient way to pass this value around.
    public final boolean preserveDebuggability;

    // The naming convention of the names in these sets will be the same as indicated by style.
    private NameStyle style;
    private Set<String> preRenameJclExceptions;
    private Set<String> preRenameUserClasses;

    // All of our permitted classes, these are set false if any are forbidden in the constructor.
    private boolean jclClassesPermitted = true;
    private boolean apiClassesPermitted = true;
    private boolean exceptionWrappersPermitted = true;
    private boolean preciseArraysPermitted = true;
    private boolean unifyingArraysPermitted = true;
    private boolean userDefinedClassesPermitted = true;

    // All of the prefixes we use, since we can statically determine these in the constructor.
    private String exceptionWrapperPrefix;
    private String postRenameApiPrefix;
    private String preRenameApiPrefix;
    private String shadowPrefix;
    private String userPrefix;

    public enum ClassCategory { JCL, API, EXCEPTION_WRAPPER, PRECISE_ARRAY, UNIFYING_ARRAY, USER}

    public enum NameCategory { PRE_RENAME, POST_RENAME }

    public enum ArrayType { PRECISE_TYPE, UNIFYING_TYPE, NOT_ARRAY }

    /**
     * Constructs a new class renamer.
     *
     * @param preserveDebuggability Whether debug mode is enabled or not.
     * @param style Whether the class names are dot- or slash-style.
     * @param jclExceptions The list of all JCL exception classes.
     * @param jclExceptionsCategory Whether the provided JCL exceptions are pre- or post-rename.
     * @param userDefinedClasses The list of all user-defined classes.
     * @param userDefinedClassesCategory Whether the provided user classes are pre- or post-rename.
     * @param prohibitedClasses Any class categories which the rename should never expect to see.
     */
    public ClassRenamer(boolean preserveDebuggability, NameStyle style, Set<String> jclExceptions, NameCategory jclExceptionsCategory,
        Set<String> userDefinedClasses, NameCategory userDefinedClassesCategory, Set<ClassCategory> prohibitedClasses) {

        RuntimeAssertionError.assertTrue(style != null);
        RuntimeAssertionError.assertTrue(jclExceptions != null);
        RuntimeAssertionError.assertTrue(jclExceptionsCategory != null);
        RuntimeAssertionError.assertTrue(userDefinedClasses != null);
        RuntimeAssertionError.assertTrue(userDefinedClassesCategory != null);

        // Collect the provided JCL exceptions.
        if (jclExceptionsCategory == NameCategory.POST_RENAME) {
            String prefix = (style == NameStyle.DOT_NAME) ? PackageConstants.kShadowDotPrefix : PackageConstants.kShadowSlashPrefix;
            this.preRenameJclExceptions = toSimplePreRenameSet(jclExceptions, prefix, style);
        } else {
            this.preRenameJclExceptions = new HashSet<>(jclExceptions);
        }

        // Collect the provided user-defined classes.
        if (userDefinedClassesCategory == NameCategory.POST_RENAME) {
            String prefix = (style == NameStyle.DOT_NAME) ? PackageConstants.kUserDotPrefix : PackageConstants.kUserSlashPrefix;
            prefix = (preserveDebuggability) ? "" : prefix;
            this.preRenameUserClasses = toSimplePreRenameSet(userDefinedClasses, prefix, style);
        } else {
            this.preRenameUserClasses = new HashSet<>(userDefinedClasses);
        }

        // Record all of the prohibited classes.
        setProhibitions(prohibitedClasses);

        // Set the prefixes up.
        if (style == NameStyle.DOT_NAME) {
            this.exceptionWrapperPrefix = PackageConstants.kExceptionWrapperDotPrefix;
            this.postRenameApiPrefix = PackageConstants.kShadowApiDotPrefix;
            this.preRenameApiPrefix = PackageConstants.kPublicApiDotPrefix;
            this.shadowPrefix = PackageConstants.kShadowDotPrefix;
            this.userPrefix = (preserveDebuggability) ? "" : PackageConstants.kUserDotPrefix;
        } else {
            this.exceptionWrapperPrefix = PackageConstants.kExceptionWrapperSlashPrefix;
            this.postRenameApiPrefix = PackageConstants.kShadowApiSlashPrefix;
            this.preRenameApiPrefix = PackageConstants.kPublicApiSlashPrefix;
            this.shadowPrefix = PackageConstants.kShadowSlashPrefix;
            this.userPrefix = (preserveDebuggability) ? "" : PackageConstants.kUserSlashPrefix;
        }

        this.style = style;
        this.preserveDebuggability = preserveDebuggability;
    }

    /**
     * Returns the post-rename name of the provided pre-rename class name. The naming style of the
     * given name is expected to be the same style that this class was initialized with, and the
     * returned name will be in this same style.
     *
     * This method will throw a {@link RejectedClassException#nonWhiteListedClass(String)} if the
     * given name does not match any of our pre-rename name checks.
     *
     * If an array name is given then {@code arrayType} will determine whether or not the post-rename
     * name will be a precise or unifying type name.
     *
     * This method does not handle exception wrapping! Use: {@code toExceptionWrapper()}.
     *
     * NOTE: this method will rename java.lang.Object to shadow Object!
     *
     * @param preRenameClassName The pre-rename class name to be renamed.
     * @param arrayType Whether to produce a precise or unifying array type.
     * @return the post-rename version of the given name.
     */
    public String toPostRenameOrRejectClass(String preRenameClassName, ArrayType arrayType) {
        return toPostRenameInternal(preRenameClassName, arrayType, true);
    }

    /**
     * Returns the post-rename name of the provided pre-rename class name. The naming style of the
     * given name is expected to be the same style that this class was initialized with, and the
     * returned name will be in this same style.
     *
     * This method will throw a {@link RuntimeAssertionError} if the given name is determined not to
     * be a pre-rename class name.
     *
     * If an array name is given then {@code arrayType} will determine whether or not the post-rename
     * name will be a precise or unifying type name.
     *
     * This method does not handle exception wrapping! Use: {@code toExceptionWrapper()}.
     *
     * NOTE: this method will rename java.lang.Object to shadow Object!
     *
     * @param name The pre-rename class name to be renamed.
     * @param arrayType Whether to produce a precise or unifying array type.
     * @return the post-rename version of the given name.
     */
    public String toPostRename(String name, ArrayType arrayType) {
        return toPostRenameInternal(name, arrayType, false);
    }


    private String toPostRenameInternal(String preRenameClassName, ArrayType arrayType, boolean allowClassRejection) {
        RuntimeAssertionError.assertTrue(!preRenameClassName.contains((this.style == NameStyle.DOT_NAME) ? "/" : "."));
        RuntimeAssertionError.assertTrue(arrayType != null);

        if (isPreRenameUserClass(preRenameClassName)) {
            return toPostRenameUserDefinedClass(preRenameClassName);
        } else if (isPreRenameArray(preRenameClassName)) {
            return toPostRenameArray(preRenameClassName, arrayType);
        } else if (isPreRenameApiClass(preRenameClassName)) {
            return toPostRenameApiClass(preRenameClassName);
        } else if (isPreRenameJclClass(preRenameClassName)) {
            return toPostRenameJclClass(preRenameClassName);
        } else {
            if (allowClassRejection) {
                throw RejectedClassException.nonWhiteListedClass(preRenameClassName);
            } else {
                throw RuntimeAssertionError.unreachable("Expected a pre-rename class name: " + preRenameClassName);
            }
        }
    }

    /**
     * Returns the pre-rename name of the provided post-rename name. The naming style of the given
     * name is expected to be the same style that this class was initialized with.
     *
     * @param postRename The post-rename class name to be renamed.
     * @return the pre-rename version of the given name.
     */
    public String toPreRename(String postRename) {
        RuntimeAssertionError.assertTrue(!postRename.contains((this.style == NameStyle.DOT_NAME) ? "/" : "."));

        if (isPostRenameUserClass(postRename)) {
            return toPreRenameUserDefinedClass(postRename);
        } else if (isIObject(postRename)) {
            return getJavaLangObject();
        } else if (isExceptionWrapper(postRename)) {
            return toPreRenameExceptionWrapper(postRename);
        } else if (isPostRenameArray(postRename)) {
            return toPreRenameArray(postRename);
        } else if (isPostRenameApiClass(postRename)) {
            return toPreRenameApiClass(postRename);
        } else if (isPostRenameJclClass(postRename)) {
            return toPreRenameJclClass(postRename);
        } else {
            throw RuntimeAssertionError.unreachable("Expected a post-rename class name: " + postRename);
        }
    }

    /**
     * Returns the given exception but as an exception wrapper. The naming style of the given name
     * is expected to be the same style that this class was initialized with, and the returned name
     * will be in this same style.
     *
     * This method does not actually check that the given exception is in fact an exception!
     *
     * It is the responsibility of the caller to ensure an exception type is provided!
     *
     * @param exception The exception to wrap.
     * @return the exception wrapper for the given exception.
     */
    public String toExceptionWrapper(String exception) {
        RuntimeAssertionError.assertTrue(!exception.contains((this.style == NameStyle.DOT_NAME) ? "/" : "."));
        return this.exceptionWrapperPrefix + exception;
    }

    /**
     * Returns {@code true} only if name is a pre-rename class name. Otherwise false.
     *
     * @param name The name whose pre- or post-ness is to be determined.
     * @return whether or not the name is pre-rename.
     */
    public boolean isPreRename(String name) {
        RuntimeAssertionError.assertTrue(!name.contains((this.style == NameStyle.DOT_NAME) ? "/" : "."));

        // In debug mode pre- and post-rename user classes are the same, this preliminary check catches this case.
        if (isPostRenameUserClass(name)) {
            return false;
        }

        return isPreRenameUserClass(name) || isPreRenameArray(name) || isPreRenameApiClass(name) || isPreRenameJclClass(name);
    }

    //<--------------------------------RENAMING METHODS-------------------------------------------->

    private String getJavaLangObject() {
        return (this.style == NameStyle.DOT_NAME) ? CommonType.JAVA_LANG_OBJECT.dotName : CommonType.JAVA_LANG_OBJECT.dotName.replaceAll("\\.", "/");
    }

    /**
     * Assumes the class is a user class!
     */
    private String toPostRenameUserDefinedClass(String preRenameUserDefinedClass) {
        if (this.userDefinedClassesPermitted) {
            return this.userPrefix + preRenameUserDefinedClass;
        } else {
            throw RuntimeAssertionError.unreachable("User-defined classes are prohibited: " + preRenameUserDefinedClass);
        }
    }

    /**
     * Assumes the class is a JCL class!
     */
    private String toPostRenameJclClass(String preRenameJclClass) {
        if (this.jclClassesPermitted) {
            return this.shadowPrefix + preRenameJclClass;
        } else {
            throw RuntimeAssertionError.unreachable("JCL classes are prohibited: " + preRenameJclClass);
        }
    }

    /**
     * Assumes the class is an API class!
     */
    private String toPostRenameApiClass(String preRenameApiClass) {
        if (this.apiClassesPermitted) {
            return this.postRenameApiPrefix + preRenameApiClass;
        } else {
            throw RuntimeAssertionError.unreachable("Api classes are prohibited: " + preRenameApiClass);
        }
    }

    /**
     * Assumes the class is an array!
     */
    private String toPostRenameArray(String preRenameArray, ArrayType arrayType) {
        if (arrayType == ArrayType.PRECISE_TYPE) {
            if (this.preciseArraysPermitted) {
                String arraySlashName = (this.style == NameStyle.DOT_NAME) ? preRenameArray.replaceAll("\\.", "/") : preRenameArray;
                String original = ArrayNameMapper.getPreciseArrayWrapperDescriptor(arraySlashName);
                return (this.style == NameStyle.DOT_NAME) ? original.replaceAll("/", "\\.") : original;
            } else {
                throw RuntimeAssertionError.unreachable("Precise-type arrays are prohibited: " + preRenameArray);
            }
        } else if (arrayType == ArrayType.UNIFYING_TYPE) {
            if (this.unifyingArraysPermitted) {
                String arraySlashName = (this.style == NameStyle.DOT_NAME) ? preRenameArray.replaceAll("\\.", "/") : preRenameArray;
                String original = ArrayNameMapper.getUnifyingArrayWrapperDescriptor(arraySlashName);
                return (this.style == NameStyle.DOT_NAME) ? original.replaceAll("/", "\\.") : original;
            } else {
                throw RuntimeAssertionError.unreachable("Unifying-type arrays are prohibited: " + preRenameArray);
            }
        } else {
            throw RuntimeAssertionError.unreachable("Expected a non-array type to be passed in but got: " + preRenameArray);
        }
    }

    /**
     * Assumes the class is a user class!
     */
    private String toPreRenameUserDefinedClass(String postRenameUserDefinedClass) {
        if (this.userDefinedClassesPermitted) {
            return postRenameUserDefinedClass.substring(this.userPrefix.length());
        } else {
            throw RuntimeAssertionError.unreachable("User-defined classes are prohibited: " + postRenameUserDefinedClass);
        }
    }

    /**
     * Assumes the class is a JCL class!
     */
    private String toPreRenameJclClass(String postRenameJclClass) {
        if (this.jclClassesPermitted) {
            return postRenameJclClass.substring(this.shadowPrefix.length());
        } else {
            throw RuntimeAssertionError.unreachable("JCL classes are prohibited: " + postRenameJclClass);
        }
    }

    /**
     * Assumes the class is an API class!
     */
    private String toPreRenameApiClass(String postRenameApiClass) {
        if (this.apiClassesPermitted) {
            return postRenameApiClass.substring(this.postRenameApiPrefix.length());
        } else {
            throw RuntimeAssertionError.unreachable("Api classes are prohibited: " + postRenameApiClass);
        }
    }

    /**
     * Assumes the class is an array!
     */
    private String toPreRenameArray(String postRenameArray) {
        if (ArrayUtil.isSpecialPostRenameArray(this.style, postRenameArray)) {

            String arrayDotName = (this.style == NameStyle.DOT_NAME) ? postRenameArray : postRenameArray.replaceAll("/", "\\.");
            String original = toPreRenameSpecialArrayDotName(arrayDotName);
            return (this.style == NameStyle.DOT_NAME) ? original : original.replaceAll("/", "\\.");

        } else if (ArrayUtil.isPostRenameConcreteTypeObjectArray(this.style, postRenameArray)) {
            if (this.preciseArraysPermitted) {
                String slashName = (this.style == NameStyle.DOT_NAME) ? postRenameArray.replaceAll("\\.", "/") : postRenameArray;
                String original = ArrayNameMapper.getOriginalNameOf(slashName);
                return (this.style == NameStyle.DOT_NAME) ? original.replaceAll("/", "\\.") : original;
            } else {
                throw RuntimeAssertionError.unreachable("Precise-type arrays are prohibited: " + postRenameArray);
            }
        } else if (ArrayUtil.isPostRenameUnifyingTypeObjectArray(this.style, postRenameArray)) {
            if (this.unifyingArraysPermitted) {
                String slashName = (this.style == NameStyle.DOT_NAME) ? postRenameArray.replaceAll("\\.", "/") : postRenameArray;
                String original = ArrayNameMapper.getOriginalNameOf(slashName);
                return (this.style == NameStyle.DOT_NAME) ? original.replaceAll("/", "\\.") : original;
            } else {
                throw RuntimeAssertionError.unreachable("Unifying-type arrays are prohibited: " + postRenameArray);
            }
        } else if (ArrayUtil.isPostRenamePrimitiveArray(this.style, postRenameArray)) {
            String slashName = (this.style == NameStyle.DOT_NAME) ? postRenameArray.replaceAll("\\.", "/") : postRenameArray;
            String original = ArrayNameMapper.getOriginalNameOf(slashName);
            return (this.style == NameStyle.DOT_NAME) ? original.replaceAll("/", "\\.") : original;
        } else {
            throw RuntimeAssertionError.unreachable("Expected a post-rename array: " + postRenameArray);
        }
    }

    /**
     * Assumes the class is an exception wrapper!
     */
    private String toPreRenameExceptionWrapper(String exceptionWrapper) {
        if (this.exceptionWrappersPermitted) {
            return exceptionWrapper.substring(this.exceptionWrapperPrefix.length());
        } else {
            throw RuntimeAssertionError.unreachable("Exception wrappers are prohibited: " + exceptionWrapper);
        }
    }

    /**
     * the 'special' handwritten arrays are: Array, IArray, ObjectArray, IObjectArray
     *
     * There is no way to get the original name of an Array or IArray type: any one of the primitives
     * may have cast to this. This will be an exception as such.
     *
     * Both ObjectArray and IObjectArray will be renamed back to a regular java.lang.Object array.
     */
    private String toPreRenameSpecialArrayDotName(String specialArrayDotName) {
        if (specialArrayDotName.equals(CommonType.ARRAY.dotName) || specialArrayDotName.equals(CommonType.I_ARRAY.dotName)) {
            throw RuntimeAssertionError.unreachable("Ambiguous pre-rename array name, cannot convert: " + specialArrayDotName);
        } else if (specialArrayDotName.equals(CommonType.OBJECT_ARRAY.dotName) || specialArrayDotName.equals(CommonType.I_OBJECT_ARRAY.dotName)) {
            return "[L" + CommonType.JAVA_LANG_OBJECT.dotName;
        } else {
            throw RuntimeAssertionError.unreachable("Expected a special handwritten array: " + specialArrayDotName);
        }
    }

    //<-----------------------------PRE-RENAME CLASS DETECTION METHODS----------------------------->

    private boolean isPreRenameUserClass(String className) {
        return this.preRenameUserClasses.contains(className);
    }

    private boolean isPreRenameJclClass(String className) {
        return isPreRenameJclNonException(className) || isPreRenameJclException(className);
    }

    private boolean isPreRenameApiClass(String className) {
        return className.startsWith(this.preRenameApiPrefix);
    }

    private boolean isPreRenameArray(String className) {
        return ArrayUtil.isPreRenameArray(className);
    }

    private boolean isPreRenameJclException(String className) {
        return this.preRenameJclExceptions.contains(className);
    }

    private boolean isPreRenameJclNonException(String className) {
        String classSlashName = (this.style == NameStyle.DOT_NAME) ? className.replaceAll("\\.", "/") : className;
        return NodeEnvironment.singleton.isClassFromJCL(classSlashName);
    }

    //<-----------------------------POST-RENAME CLASS DETECTION METHODS---------------------------->

    private boolean isIObject(String className) {
        String iObject = (this.style == NameStyle.DOT_NAME) ? CommonType.I_OBJECT.dotName : CommonType.I_OBJECT.dotName.replaceAll("\\.", "/");
        return iObject.equals(className);
    }

    private boolean isPostRenameUserClass(String className) {
        int prefixLength = Math.min(className.length(), this.userPrefix.length());
        return this.preRenameUserClasses.contains(className.substring(prefixLength));
    }

    private boolean isPostRenameJclClass(String className) {
        int prefixLength = Math.min(className.length(), this.shadowPrefix.length());
        return isPostRenameJclNonException(className, prefixLength) || isPostRenameJclException(className, prefixLength);
    }

    private boolean isPostRenameApiClass(String className) {
        return className.startsWith(this.postRenameApiPrefix);
    }

    private boolean isExceptionWrapper(String className) {
        return className.startsWith(this.exceptionWrapperPrefix);
    }

    private boolean isPostRenameArray(String className) {
        return ArrayUtil.isPostRenameArray(this.style, className);
    }

    private boolean isPostRenameJclNonException(String className, int prefixLength) {
        String classSlashName = (this.style == NameStyle.DOT_NAME) ? className.replaceAll("\\.", "/") : className;
        return NodeEnvironment.singleton.isClassFromJCL(classSlashName.substring(prefixLength));
    }

    private boolean isPostRenameJclException(String className, int prefixLength) {
        return this.preRenameJclExceptions.contains(className.substring(prefixLength));
    }

    /**
     * Returns the same set as postRenameSet except with all strings having the given prefix removed
     * from them.
     *
     * The outputStyle is the naming style that the output set will be named in!
     *
     * It is the caller's responsibility to ensure this prefix actually exists in each name in the set.
     */
    private Set<String> toSimplePreRenameSet(Set<String> postRenameSet, String prefix, NameStyle outputStyle) {
        Set<String> preRenameSet = new HashSet<>();
        for (String postRename : postRenameSet) {
            String preRenameName = postRename.substring(prefix.length());
            preRenameName = (outputStyle == NameStyle.DOT_NAME) ? preRenameName.replaceAll("/", "\\.") : preRenameName.replaceAll("\\.", "/");
            preRenameSet.add(preRenameName);
        }
        return preRenameSet;
    }

    private void setProhibitions(Set<ClassCategory> prohibitedClasses) {
        if (prohibitedClasses != null) {
            for (ClassCategory prohibitedClass : prohibitedClasses) {
                switch (prohibitedClass) {
                    case JCL:
                        this.jclClassesPermitted = false;
                        break;
                    case API:
                        this.apiClassesPermitted = false;
                        break;
                    case EXCEPTION_WRAPPER:
                        this.exceptionWrappersPermitted = false;
                        break;
                    case PRECISE_ARRAY:
                        this.preciseArraysPermitted = false;
                        break;
                    case UNIFYING_ARRAY:
                        this.unifyingArraysPermitted = false;
                        break;
                    case USER:
                        this.userDefinedClassesPermitted = false;
                        break;
                    default: throw RuntimeAssertionError.unreachable("Unexpected prohibited class: " + prohibitedClass);
                }
            }
        }
    }
}
