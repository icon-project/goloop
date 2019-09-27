package org.aion.avm.core.exceptionwrapping;

import i.PackageConstants;
import i.RuntimeAssertionError;


/**
 * Maps back and forth between exception wrapper and underlying class names.
 * Note that this operates on /-style names, only.
 */
public class ExceptionWrapperNameMapper {
    public static String slashWrapperNameForClassName(String className) {
        // NOTE:  These are "/-style" names.
        RuntimeAssertionError.assertTrue(-1 == className.indexOf("."));
        
        return PackageConstants.kExceptionWrapperSlashPrefix + className;
    }

    public static String slashClassNameForWrapperName(String wrapperName) {
        // NOTE:  These are "/-style" names.
        RuntimeAssertionError.assertTrue(-1 == wrapperName.indexOf("."));
        // This should only be called on wrapper names.
        RuntimeAssertionError.assertTrue(wrapperName.startsWith(PackageConstants.kExceptionWrapperSlashPrefix));
        
        return wrapperName.substring(PackageConstants.kExceptionWrapperSlashPrefix.length());
    }

    public static String dotClassNameForWrapperName(String wrapperName) {
        // NOTE:  These are ".-style" names.
        RuntimeAssertionError.assertTrue(-1 == wrapperName.indexOf("/"));
        // This should only be called on wrapper names.
        RuntimeAssertionError.assertTrue(wrapperName.startsWith(PackageConstants.kExceptionWrapperDotPrefix));

        return wrapperName.substring(PackageConstants.kExceptionWrapperDotPrefix.length());
    }

    public static boolean isExceptionWrapper(String typeName) {
        // NOTE:  These are "/-style" names.
        RuntimeAssertionError.assertTrue(-1 == typeName.indexOf("."));
        return typeName.startsWith(PackageConstants.kExceptionWrapperSlashPrefix);
    }

    public static boolean isExceptionWrapperDotName(String typeName) {
        // NOTE:  These are ".-dot" names.
        RuntimeAssertionError.assertTrue(-1 == typeName.indexOf("/"));
        return typeName.startsWith(PackageConstants.kExceptionWrapperDotPrefix);
    }

}
