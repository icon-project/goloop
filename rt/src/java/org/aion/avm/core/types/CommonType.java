package org.aion.avm.core.types;

import i.PackageConstants;

/**
 * An enumeration of some types that are used often and for which we have no good way of
 * auto-generating.
 *
 * Each type contains the necessary information so that a hierarchy would be able to directly read
 * the class from here and be able to determine its relationships with other types in the system.
 *
 * In particular, each type listed here contains its own name, its direct super class name, and the
 * names of all the interfaces it directly implements. Note that all names are .-style
 *
 * In addition to this, each type also has the following data associated with it:
 *
 * 1. isInterface: true only if the type is an interface, otherwise false.
 * 2. isException: true only if the type is an exception type (shadow or otherwise), otherwise false.
 * 3. isVirtualMachineErrorOrChildError: true only if the type is VirtualMachineError or else one
 *    of its child error types, otherwise false.
 */
public enum CommonType {

    // The non-exception shadow types as well as the JCL types.
    JAVA_LANG_OBJECT    (false, false,   false, "java.lang.Object",                                                 null,                       null),
    JAVA_LANG_THROWABLE (false, false,   false, "java.lang.Throwable",                                              JAVA_LANG_OBJECT.dotName,   null),
    I_OBJECT            (true,  false,   false, PackageConstants.kInternalDotPrefix + "IObject",                    JAVA_LANG_OBJECT.dotName,   null),
    SHADOW_OBJECT       (false, false,   false, PackageConstants.kShadowDotPrefix + JAVA_LANG_OBJECT.dotName,       null,                       new String[]{ I_OBJECT.dotName}),
    SHADOW_SERIALIZABLE (true,  false,   false, PackageConstants.kShadowDotPrefix + "java.io.Serializable",         null,                       new String[]{ I_OBJECT.dotName}),
    SHADOW_COMPARABLE   (true,  false,   false, PackageConstants.kShadowDotPrefix + "java.lang.Comparable",         null,                       new String[]{ I_OBJECT.dotName}),
    SHADOW_CLONEABLE    (true,  false,   false, PackageConstants.kShadowDotPrefix + "java.lang.Cloneable",          null,                       new String[]{ I_OBJECT.dotName}),
    SHADOW_ENUM         (false, false,   false, PackageConstants.kShadowDotPrefix + "java.lang.Enum",               JAVA_LANG_OBJECT.dotName,   new String[]{ SHADOW_COMPARABLE.dotName, SHADOW_SERIALIZABLE.dotName}),
    I_ARRAY             (true,  false,   false, PackageConstants.kArrayWrapperDotPrefix + "IArray",                 null,                       new String[]{ I_OBJECT.dotName}),
    I_OBJECT_ARRAY      (true,  false,   false, PackageConstants.kInternalDotPrefix + "IObjectArray",               null,                       new String[]{ I_OBJECT.dotName, I_ARRAY.dotName}),
    ARRAY               (false, false,   false, PackageConstants.kArrayWrapperDotPrefix + "Array",                  SHADOW_OBJECT.dotName,      new String[]{ SHADOW_CLONEABLE.dotName, I_ARRAY.dotName}),
    OBJECT_ARRAY        (false, false,   false, PackageConstants.kArrayWrapperDotPrefix + "ObjectArray",            ARRAY.dotName,              new String[]{ I_OBJECT_ARRAY.dotName}),
    ARRAY_ELEMENT       (false, false,   false, PackageConstants.kArrayWrapperDotPrefix + "ArrayElement",           SHADOW_ENUM.dotName,        null),
    BOOLEAN_ARRAY       (false, false,   false, PackageConstants.kArrayWrapperDotPrefix + "BooleanArray",           ARRAY.dotName,              null),
    BYTE_ARRAY          (false, false,   false, PackageConstants.kArrayWrapperDotPrefix + "ByteArray",              ARRAY.dotName,              null),
    CHAR_ARRAY          (false, false,   false, PackageConstants.kArrayWrapperDotPrefix + "CharArray",              ARRAY.dotName,              null),
    DOUBLE_ARRAY        (false, false,   false, PackageConstants.kArrayWrapperDotPrefix + "DoubleArray",            ARRAY.dotName,              null),
    FLOAT_ARRAY         (false, false,   false, PackageConstants.kArrayWrapperDotPrefix + "FloatArray",             ARRAY.dotName,              null),
    INT_ARRAY           (false, false,   false, PackageConstants.kArrayWrapperDotPrefix + "IntArray",               ARRAY.dotName,              null),
    LONG_ARRAY          (false, false,   false, PackageConstants.kArrayWrapperDotPrefix + "LongArray",              ARRAY.dotName,              null),
    SHORT_ARRAY         (false, false,   false, PackageConstants.kArrayWrapperDotPrefix + "ShortArray",             ARRAY.dotName,              null),

    // The shadow exception types.
    SHADOW_THROWABLE                (false, true,   false,  PackageConstants.kShadowDotPrefix + JAVA_LANG_THROWABLE.dotName,                SHADOW_OBJECT.dotName,                      null),
    SHADOW_ERROR                    (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.Error",                          SHADOW_THROWABLE.dotName,                   null),
    SHADOW_VIRTUAL_MACHINE_ERROR    (false, true,   true,   PackageConstants.kShadowDotPrefix + "java.lang.VirtualMachineError",            SHADOW_ERROR.dotName,                       null),
    SHADOW_INTERNAL_ERROR           (false, true,   true,   PackageConstants.kShadowDotPrefix + "java.lang.InternalError",                  SHADOW_VIRTUAL_MACHINE_ERROR.dotName,       null),
    SHADOW_ZIP_ERROR                (false, true,   true,   PackageConstants.kShadowDotPrefix + "java.util.zip.ZipError",                   SHADOW_INTERNAL_ERROR.dotName,              null),
    SHADOW_OUT_OF_MEMORY_ERROR      (false, true,   true,   PackageConstants.kShadowDotPrefix + "java.lang.OutOfMemoryError",               SHADOW_VIRTUAL_MACHINE_ERROR.dotName,       null),
    SHADOW_STACK_OVERFLOW_ERROR     (false, true,   true,   PackageConstants.kShadowDotPrefix + "java.lang.StackOverflowError",             SHADOW_VIRTUAL_MACHINE_ERROR.dotName,       null),
    SHADOW_UNKNOWN_ERROR            (false, true,   true,   PackageConstants.kShadowDotPrefix + "java.lang.UnknownError",                   SHADOW_VIRTUAL_MACHINE_ERROR.dotName,       null),
    SHADOW_ASSERTION_ERROR          (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.AssertionError",                 SHADOW_ERROR.dotName,                       null),
    SHADOW_LINKAGE_ERROR            (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.LinkageError",                   SHADOW_ERROR.dotName,                       null),
    SHADOW_BOOTSTRAP_ERROR          (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.BootstrapMethodError",           SHADOW_LINKAGE_ERROR.dotName,               null),
    SHADOW_CIRCULARITY_ERROR        (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.ClassCircularityError",          SHADOW_LINKAGE_ERROR.dotName,               null),
    SHADOW_CLASS_FORMAT_ERROR       (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.ClassFormatError",               SHADOW_LINKAGE_ERROR.dotName,               null),
    SHADOW_VERSION_ERROR            (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.UnsupportedClassVersionError",   SHADOW_CLASS_FORMAT_ERROR.dotName,          null),
    SHADOW_INITIALIZER_ERROR        (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.ExceptionInInitializerError",    SHADOW_LINKAGE_ERROR.dotName,               null),
    SHADOW_INCOMPATIBLE_CHANGE_ERROR(false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.IncompatibleClassChangeError",   SHADOW_LINKAGE_ERROR.dotName,               null),
    SHADOW_ABSTRACT_METHOD_ERROR    (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.AbstractMethodError",            SHADOW_INCOMPATIBLE_CHANGE_ERROR.dotName,   null),
    SHADOW_ILLEGAL_ACCESS_ERROR     (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.IllegalAccessError",             SHADOW_INCOMPATIBLE_CHANGE_ERROR.dotName,   null),
    SHADOW_INSTANTIATION_ERROR      (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.InstantiationError",             SHADOW_INCOMPATIBLE_CHANGE_ERROR.dotName,   null),
    SHADOW_NO_SUCH_FIELD_ERROR      (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.NoSuchFieldError",               SHADOW_INCOMPATIBLE_CHANGE_ERROR.dotName,   null),
    SHADOW_NO_SUCH_METHOD_ERROR     (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.NoSuchMethodError",              SHADOW_INCOMPATIBLE_CHANGE_ERROR.dotName,   null),
    SHADOW_NO_CLASS_DEF_FOUND_ERROR (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.NoClassDefFoundError",           SHADOW_LINKAGE_ERROR.dotName,               null),
    SHADOW_UNSATISFIED_LINK_ERROR   (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.UnsatisfiedLinkError",           SHADOW_LINKAGE_ERROR.dotName,               null),
    SHADOW_VERIFY_ERROR             (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.VerifyError",                    SHADOW_LINKAGE_ERROR.dotName,               null),
    SHADOW_THREAD_DEATH             (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.ThreadDeath",                    SHADOW_ERROR.dotName,                       null),
    SHADOW_EXCEPTION                (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.Exception",                      SHADOW_THROWABLE.dotName,                   null),
    SHADOW_CLONE_EXCEPTION          (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.CloneNotSupportedException",     SHADOW_EXCEPTION.dotName,                   null),
    SHADOW_INTERRUPTED_EXCEPTION    (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.InterruptedException",           SHADOW_EXCEPTION.dotName,                   null),
    SHADOW_REFLECTIVE_OP_EXCEPTION  (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.ReflectiveOperationException",   SHADOW_EXCEPTION.dotName,                   null),
    SHADOW_CLASS_NOT_FOUND_EXCEPTION(false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.ClassNotFoundException",         SHADOW_REFLECTIVE_OP_EXCEPTION.dotName,     null),
    SHADOW_ILLEGAL_ACCESS_EXCEPTION (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.IllegalAccessException",         SHADOW_REFLECTIVE_OP_EXCEPTION.dotName,     null),
    SHADOW_INSTANTIATION_EXCEPTION  (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.InstantiationException",         SHADOW_REFLECTIVE_OP_EXCEPTION.dotName,     null),
    SHADOW_NO_SUCH_FIELD_EXCEPTION  (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.NoSuchFieldException",           SHADOW_REFLECTIVE_OP_EXCEPTION.dotName,     null),
    SHADOW_NO_SUCH_METHOD_EXCEPTION (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.NoSuchMethodException",          SHADOW_REFLECTIVE_OP_EXCEPTION.dotName,     null),
    SHADOW_RUNTIME_EXCEPTION        (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.RuntimeException",               SHADOW_EXCEPTION.dotName,                   null),
    SHADOW_ARITHMETIC_EXCEPTION     (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.ArithmeticException",            SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_ARRAY_STORE_EXCEPTION    (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.ArrayStoreException",            SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_CLASS_CAST_EXCEPTION     (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.ClassCastException",             SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_ENUM_CONSTANT_EXCEPTION  (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.EnumConstantNotPresentException",SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_ILLEGAL_ARG_EXCEPTION    (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.IllegalArgumentException",       SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_ILLEGAL_THREAD_EXCEPTION (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.IllegalThreadStateException",    SHADOW_ILLEGAL_ARG_EXCEPTION.dotName,       null),
    SHADOW_NUMBER_FORMAT_EXCEPTION  (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.NumberFormatException",          SHADOW_ILLEGAL_ARG_EXCEPTION.dotName,       null),
    SHADOW_ILLEGAL_CALLER_EXCEPTION (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.IllegalCallerException",         SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_ILLEGAL_MONITOR_EXCEPTION(false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.IllegalMonitorStateException",   SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_ILLEGAL_STATE_EXCEPTION  (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.IllegalStateException",          SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_INDEX_BOUND_EXCEPTION    (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.IndexOutOfBoundsException",      SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_ARRAY_BOUND_EXCEPTION    (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.ArrayIndexOutOfBoundsException", SHADOW_INDEX_BOUND_EXCEPTION.dotName,       null),
    SHADOW_STRING_BOUND_EXCEPTION   (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.StringIndexOutOfBoundsException",SHADOW_INDEX_BOUND_EXCEPTION.dotName,       null),
    SHADOW_LAYER_EXCEPTION          (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.LayerInstantiationException",    SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_NEGATIVE_SIZE_EXCEPTION  (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.NegativeArraySizeException",     SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_NULL_POINTER_EXCEPTION   (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.NullPointerException",           SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_SECURITY_EXCEPTION       (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.SecurityException",              SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_NO_TYPE_PRESENT_EXCEPTION(false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.TypeNotPresentException",        SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_UNSUPPORTED_OP_EXCEPTION (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.lang.UnsupportedOperationException",  SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_NO_SUCH_ELEMENT_EXCEPTION(false, true,   false,  PackageConstants.kShadowDotPrefix + "java.util.NoSuchElementException",         SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_BUFF_UNDERFLOW_EXCEPTION (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.nio.BufferUnderflowException",        SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    SHADOW_BUFF_OVERFLOW_EXCEPTION  (false, true,   false,  PackageConstants.kShadowDotPrefix + "java.nio.BufferOverflowException",         SHADOW_RUNTIME_EXCEPTION.dotName,           null),
    ;

    public final boolean isInterface;
    public final boolean isShadowException;
    public final boolean isVirtualMachineErrorOrChildError;
    public final String dotName;
    public final String superClassDotName;
    public final String[] superInterfacesDotNames;    // Caller must never modify this field!

    /**
     * A representation of some type.
     *
     * @param isInterface True indicates this type is an interface.
     * @param isException True indicates this type is a shadow exception type - shadow Throwable or one of its descendants.
     * @param isVirtualMachineErrorOrChildError True indicates this type is VirtualMachineError or one of its descendants.
     * @param dotName The dot-style name of this type.
     * @param parentDotName The dot-style name of this type's concrete super class.
     * @param interfaces The dot-style names of this type's super interfaces.
     */
    CommonType(boolean isInterface, boolean isException, boolean isVirtualMachineErrorOrChildError, String dotName, String parentDotName, String[] interfaces) {
        this.isInterface = isInterface;
        this.isShadowException = isException;
        this.isVirtualMachineErrorOrChildError = isVirtualMachineErrorOrChildError;
        this.dotName = dotName;
        this.superClassDotName = parentDotName;
        this.superInterfacesDotNames = interfaces;
    }

}
