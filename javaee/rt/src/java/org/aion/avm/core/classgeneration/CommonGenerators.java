package org.aion.avm.core.classgeneration;

import java.util.Arrays;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;

import org.aion.avm.ClassNameExtractor;
import org.aion.avm.core.types.CommonType;
import i.PackageConstants;
import i.RuntimeAssertionError;


/**
 * Contains some of the common constants and code-generation idioms used in various tests and/or across the system, in general.
 */
public class CommonGenerators {
    // There doesn't appear to be any way to enumerate these classes in the existing class loader (even though they are part of java.lang)
    // so we will list the names of all the classes we need and assemble them that way.
    // We should at least be able to use the original Throwable's classloader to look up the subclasses (again, since they are in java.lang).
    // Note:  "java.lang.VirtualMachineError" and children are deliberately absent from this since user code can never see them.
    public static final String[] kExceptionClassNames = Arrays.stream(CommonType.values())
        .filter((type) -> (type.isShadowException && !type.isVirtualMachineErrorOrChildError && !type.dotName.equals(CommonType.SHADOW_THROWABLE.dotName)))
        .map((type) -> (ClassNameExtractor.getOriginalClassName(type.dotName)))
        .toArray(String[]::new);

    // We don't generate the shadows for these ones since we have hand-written them (but wrappers are still required).
    public static final Set<String> kHandWrittenExceptionClassNames = Set.of(new String[] {
        ClassNameExtractor.getOriginalClassName(CommonType.SHADOW_ERROR.dotName),
        ClassNameExtractor.getOriginalClassName(CommonType.SHADOW_ASSERTION_ERROR.dotName),
        ClassNameExtractor.getOriginalClassName(CommonType.SHADOW_EXCEPTION.dotName),
        ClassNameExtractor.getOriginalClassName(CommonType.SHADOW_RUNTIME_EXCEPTION.dotName),
        ClassNameExtractor.getOriginalClassName(CommonType.SHADOW_ENUM_CONSTANT_EXCEPTION.dotName),
        ClassNameExtractor.getOriginalClassName(CommonType.SHADOW_NO_TYPE_PRESENT_EXCEPTION.dotName),
        ClassNameExtractor.getOriginalClassName(CommonType.SHADOW_NO_SUCH_ELEMENT_EXCEPTION.dotName),
    });

    // We generate "legacy-style exception" shadows for these ones (and wrappers are still required).
    public static final Set<String> kLegacyExceptionClassNames = Set.of(new String[] {
        ClassNameExtractor.getOriginalClassName(CommonType.SHADOW_INITIALIZER_ERROR.dotName),
        ClassNameExtractor.getOriginalClassName(CommonType.SHADOW_CLASS_NOT_FOUND_EXCEPTION.dotName),
    });

    // Record the parent class of each generated class. This information is needed by the heap size calculation.
    // Both class names are in the shadowed version.
    public static Map<String, String> parentClassMap;

    // Note that the calculating allocation sizes for generated exceptions (JCLAndAPIHeapInstanceSize}) assumes that
    // generated exceptions do not declare any fields, and do not inherit from an Exception class with fields other than Throwable.
    // Thus, the default allocation size for each class is 32 bytes.
    public static Map<String, byte[]> generateShadowJDK() {
        Map<String, byte[]> shadowJDK = new HashMap<>();

        Map<String, byte[]> shadowException = generateShadowException();

        shadowJDK.putAll(shadowException);

        return shadowJDK;
    }

    public static Map<String, byte[]> generateShadowException() {
        Map<String, byte[]> generatedClasses = new HashMap<>();
        parentClassMap = new HashMap<>();
        for (String className : kExceptionClassNames) {
            // We need to look this up to find the superclass.
            String superclassName = null;
            try {
                superclassName = Class.forName(className).getSuperclass().getName();
            } catch (ClassNotFoundException e) {
                // We are operating on built-in exception classes so, if these are missing, there is something wrong with the JDK.
                throw RuntimeAssertionError.unexpected(e);
            }
            
            // Generate the shadow.
            if (!kHandWrittenExceptionClassNames.contains(className)) {
                // Note that we are currently listing the shadow "java.lang." directly, so strip off the redundant "java.lang."
                // (this might change in the future).
                String shadowName = PackageConstants.kShadowDotPrefix + className;
                String shadowSuperName = PackageConstants.kShadowDotPrefix + superclassName;
                byte[] shadowBytes = null;
                if (kLegacyExceptionClassNames.contains(className)) {
                    // "Legacy" exception.
                    shadowBytes = generateLegacyExceptionClass(shadowName, shadowSuperName);
                } else {
                    // "Standard" exception.
                    shadowBytes = generateExceptionClass(shadowName, shadowSuperName);
                }
                
                generatedClasses.put(shadowName, shadowBytes);

                parentClassMap.put(shadowName, shadowSuperName);
            }
            
            // Generate the wrapper.
            String wrapperName = PackageConstants.kExceptionWrapperDotPrefix + PackageConstants.kShadowDotPrefix + className;
            String wrapperSuperName = PackageConstants.kExceptionWrapperDotPrefix + PackageConstants.kShadowDotPrefix + superclassName;
            byte[] wrapperBytes = generateWrapperClass(wrapperName, wrapperSuperName);
            generatedClasses.put(wrapperName, wrapperBytes);
        }
        return generatedClasses;
    }

    private static byte[] generateWrapperClass(String mappedName, String mappedSuperName) {
        String slashName = mappedName.replaceAll("\\.", "/");
        String superSlashName = mappedSuperName.replaceAll("\\.", "/");
        return StubGenerator.generateWrapperClass(slashName, superSlashName);
    }

    private static byte[] generateExceptionClass(String mappedName, String mappedSuperName) {
        String slashName = mappedName.replaceAll("\\.", "/");
        String superSlashName = mappedSuperName.replaceAll("\\.", "/");
        return StubGenerator.generateExceptionClass(slashName, superSlashName);
    }

    private static byte[] generateLegacyExceptionClass(String mappedName, String mappedSuperName) {
        String slashName = mappedName.replaceAll("\\.", "/");
        String superSlashName = mappedSuperName.replaceAll("\\.", "/");
        return StubGenerator.generateLegacyExceptionClass(slashName, superSlashName);
    }
}
