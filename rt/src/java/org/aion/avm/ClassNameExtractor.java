package org.aion.avm;

import i.PackageConstants;

public class ClassNameExtractor {

    public static String getOriginalClassName(String internalName) {
        if (internalName.startsWith(PackageConstants.kArrayWrapperDotPrefix)) {
            return getArrayClassFromWrapper(internalName);
        } else {
            return removeInternalPrefix(internalName);
        }
    }

    public static boolean isPostRenameClassDotStyle(String className) {
        return className.startsWith(PackageConstants.kShadowDotPrefix)
            || className.startsWith(PackageConstants.kUserDotPrefix)
            || className.startsWith(PackageConstants.kShadowApiDotPrefix)
            || className.startsWith(PackageConstants.kArrayWrapperDotPrefix)
            || className.startsWith(PackageConstants.kExceptionWrapperDotPrefix)
            || className.startsWith(PackageConstants.kInternalDotPrefix);
    }

    private static String removeInternalPrefix(String className) {
        if (className.startsWith(PackageConstants.kShadowDotPrefix)) {
            return className.substring(PackageConstants.kShadowDotPrefix.length());
        } else if (className.startsWith(PackageConstants.kUserDotPrefix)) {
            return className.substring(PackageConstants.kUserDotPrefix.length());
        } else if (className.startsWith(PackageConstants.kShadowApiDotPrefix)) {
            return className.substring(PackageConstants.kShadowApiDotPrefix.length());
        } else {
            return className;
        }
    }

    private static String getArrayClassFromWrapper(String className) {
        String arrayName = ArrayClassNameMapper.getOriginalNameFromWrapper(className.replaceAll("\\.", "/"));
        String name;
        int dimension = 0;
        if (arrayName != null) {
            // if the shared map contains the className, change the name to fully qualified name
            // otherwise get the prefix and className from transformed className
            name = arrayName.replaceAll("/", ".");
            dimension = getDimension(name, '[');
        } else if(className.startsWith(PackageConstants.kArrayWrapperDotPrefix)){
            name = className.substring(PackageConstants.kArrayWrapperDotPrefix.length());
            dimension = getDimension(name, '$');
        } else {
            throw new AssertionError("Unexpected array wrapper class name.");
        }
        String transformedTokens = new String(new char[dimension]).replace("\0", "[");
        return transformedTokens + getElementName(name.substring(dimension));
    }

    private static int getDimension(String desc, char prefix){
        int d = 0;
        while (desc.charAt(d) == prefix) {
            d++;
        }
        return d;
    }

    private static String getElementName(String internalName) {
        if (internalName.startsWith("L")) {
            return "L" + removeInternalPrefix(internalName.substring(1)) + ";";
        } else {
            return removeInternalPrefix(internalName);
        }
    }
}



