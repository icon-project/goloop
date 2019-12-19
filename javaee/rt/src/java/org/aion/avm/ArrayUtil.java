package org.aion.avm;

import java.util.Set;
import java.util.regex.Pattern;
import java.util.stream.Collectors;
import java.util.stream.Stream;
import i.PackageConstants;
import i.RuntimeAssertionError;

public final class ArrayUtil {
    private static final String DOT_PREFIX = PackageConstants.kArrayWrapperDotPrefix;
    private static final String SLASH_PREFIX = PackageConstants.kArrayWrapperSlashPrefix;

    private static final Pattern PRE_RENAME_PRIMITIVE_1D = Pattern.compile("^\\[[IBZCFSJD]$");
    private static final Pattern PRE_RENAME_PRIMITIVE_MD = Pattern.compile("^\\[{2,}[IBZCFSJD]$");
    private static final Pattern PRE_RENAME_OBJECT = Pattern.compile("^\\[+L.+");
    private static final Pattern POST_RENAME_CONCRETE_DOT = Pattern.compile("^" + PackageConstants.kArrayWrapperDotPrefix + "\\$+L.+");
    private static final Pattern POST_RENAME_CONCRETE_SLASH = Pattern.compile("^" + PackageConstants.kArrayWrapperSlashPrefix + "\\$+L.+");
    private static final Pattern POST_RENAME_UNIFYING_DOT = Pattern.compile("^" + PackageConstants.kArrayWrapperUnifyingDotPrefix + "_+L.+");
    private static final Pattern POST_RENAME_UNIFYING_SLASH = Pattern.compile("^" + PackageConstants.kArrayWrapperUnifyingSlashPrefix + "_+L.+");

    private static Set<String> postRenamePrimitiveSimpleNames = Stream.of(
        a.IntArray.class.getSimpleName(),
        a.LongArray.class.getSimpleName(),
        a.ByteArray.class.getSimpleName(),
        a.BooleanArray.class.getSimpleName(),
        a.ShortArray.class.getSimpleName(),
        a.DoubleArray.class.getSimpleName(),
        a.FloatArray.class.getSimpleName(),
        a.CharArray.class.getSimpleName())
        .collect(Collectors.toSet());

    private static Set<Character> primitiveSignifiers = Stream.of('I', 'J', 'Z', 'B', 'S', 'D', 'F', 'C').collect(Collectors.toSet());

    public static boolean isPreRenameSingleDimensionPrimitiveArray(String array) {
        RuntimeAssertionError.assertTrue(array != null);
        return PRE_RENAME_PRIMITIVE_1D.matcher(array).matches();
    }

    public static boolean isPreRenameMultiDimensionPrimitiveArray(String array) {
        RuntimeAssertionError.assertTrue(array != null);
        return PRE_RENAME_PRIMITIVE_MD.matcher(array).matches();
    }

    public static boolean isPreRenamePrimitiveArray(String array) {
        return isPreRenameSingleDimensionPrimitiveArray(array) || isPreRenameMultiDimensionPrimitiveArray(array);
    }

    public static boolean isPreRenameObjectArray(String array) {
        RuntimeAssertionError.assertTrue(array != null);
        return PRE_RENAME_OBJECT.matcher(array).matches();
    }

    public static boolean isPreRenameArray(String array) {
        return isPreRenamePrimitiveArray(array) || isPreRenameObjectArray(array);
    }

    public static boolean isPostRenameSingleDimensionPrimitiveArray(NameStyle style, String array) {
        String unwantedCharacter = (style == NameStyle.DOT_NAME) ? "/" : ".";
        RuntimeAssertionError.assertTrue(!array.contains(unwantedCharacter));

        String prefix = (style == NameStyle.DOT_NAME) ? DOT_PREFIX : SLASH_PREFIX;

        if (!array.startsWith(prefix)) {
            return false;
        }

        String strippedName = array.substring(prefix.length());
        return postRenamePrimitiveSimpleNames.contains(strippedName);
    }

    public static boolean isPostRenameMultiDimensionPrimitiveArray(NameStyle style, String array) {
        String unwantedCharacter = (style == NameStyle.DOT_NAME) ? "/" : ".";
        RuntimeAssertionError.assertTrue(!array.contains(unwantedCharacter));

        String prefix = ((style == NameStyle.DOT_NAME) ? DOT_PREFIX : SLASH_PREFIX);
        if (!array.startsWith(prefix)) {
            return false;
        }

        String strippedName = array.substring(prefix.length());
        int dimension = numberOfLeading('$', strippedName);
        if (dimension < 1) {
            return false;
        }

        char lastChar = strippedName.charAt(strippedName.length() - 1);
        return (strippedName.length() == dimension + 1) && (primitiveSignifiers.contains(lastChar));
    }

    public static boolean isPostRenamePrimitiveArray(NameStyle style, String array) {
        return isPostRenameSingleDimensionPrimitiveArray(style, array) || isPostRenameMultiDimensionPrimitiveArray(style, array);
    }

    public static boolean isPostRenameConcreteTypeObjectArray(NameStyle style, String array) {
        RuntimeAssertionError.assertTrue(style != null);
        RuntimeAssertionError.assertTrue(array != null);
        return (style == NameStyle.DOT_NAME) ? POST_RENAME_CONCRETE_DOT.matcher(array).matches() : POST_RENAME_CONCRETE_SLASH.matcher(array).matches();
    }

    public static boolean isPostRenameUnifyingTypeObjectArray(NameStyle style, String array) {
        RuntimeAssertionError.assertTrue(style != null);
        RuntimeAssertionError.assertTrue(array != null);
        return (style == NameStyle.DOT_NAME) ? POST_RENAME_UNIFYING_DOT.matcher(array).matches() : POST_RENAME_UNIFYING_SLASH.matcher(array).matches();
    }

    public static boolean isPostRenameObjectArray(NameStyle style, String array) {
        return isPostRenameConcreteTypeObjectArray(style, array) || isPostRenameUnifyingTypeObjectArray(style, array);
    }

    public static boolean isPostRenameArray(NameStyle style, String array) {
        return isPostRenamePrimitiveArray(style, array) || isPostRenameObjectArray(style, array) || isSpecialPostRenameArray(style, array);
    }

    public static boolean isSpecialPostRenameArray(NameStyle style, String array) {
        String prefix = (style == NameStyle.DOT_NAME) ? PackageConstants.kArrayWrapperDotPrefix : PackageConstants.kArrayWrapperSlashPrefix;
        String internalPrefix = (style == NameStyle.DOT_NAME) ? PackageConstants.kInternalDotPrefix : PackageConstants.kInternalSlashPrefix;

        if (array.equals(prefix + "ObjectArray") || array.equals(internalPrefix + "IObjectArray") || array.equals(prefix + "Array") || array.equals(prefix + "IArray")) {
            return true;
        } else {
            return false;
        }
    }

    public static boolean isSingleDimensionalPrimitiveArray(NameStyle style, String array) {
        return isPreRenameSingleDimensionPrimitiveArray(array) || isPostRenameSingleDimensionPrimitiveArray(style, array);
    }

    public static boolean isMultiDimensionalPrimitiveArray(NameStyle style, String array) {
        return isPreRenameMultiDimensionPrimitiveArray(array) || isPostRenameMultiDimensionPrimitiveArray(style, array);
    }

    public static boolean isPrimitiveArray(NameStyle style, String array) {
        return isPreRenamePrimitiveArray(array) || isPostRenamePrimitiveArray(style, array);
    }

    public static boolean isObjectArray(NameStyle style, String array) {
        return isPreRenameObjectArray(array) || isPostRenameObjectArray(style, array);
    }

    public static boolean isArray(NameStyle style, String array) {
        return isPreRenameArray(array) || isPostRenameArray(style, array);
    }

    /**
     * Returns the dimension of the provided pre-rename primitive array.
     */
    public static int dimensionOfPreRenamePrimitiveArray(String array) {
        RuntimeAssertionError.assertTrue(isPreRenamePrimitiveArray(array));
        return numberOfLeading('[', array);
    }

    /**
     * Returns the dimension of the provided pre-rename object array.
     */
    public static int dimensionOfPreRenameObjectArray(String array) {
        RuntimeAssertionError.assertTrue(isPreRenameObjectArray(array));
        return numberOfLeading('[', array);
    }

    public static int dimensionOfPreRenameArray(String array) {
        if (isPreRenamePrimitiveArray(array)) {
            return dimensionOfPreRenamePrimitiveArray(array);
        } else if (isPreRenameObjectArray(array)) {
            return dimensionOfPreRenameObjectArray(array);
        } else {
            throw RuntimeAssertionError.unreachable("Expected pre-rename array: " + array);
        }
    }

    private static int dimensionOfPostRenameMultiDimensionPrimitiveArray(NameStyle style, String array) {
        String unwrappedArray = ArrayRenamer.stripArrayWrapperPrefix(style, array);
        return numberOfLeading('$', unwrappedArray);
    }

    public static int dimensionOfPostRenamePrimitiveArray(NameStyle style, String array) {
        if (isPostRenameSingleDimensionPrimitiveArray(style, array)) {
            return 1;
        } else if (isPostRenameMultiDimensionPrimitiveArray(style, array)) {
            return dimensionOfPostRenameMultiDimensionPrimitiveArray(style, array);
        } else {
            throw RuntimeAssertionError.unreachable("Expected post-rename primitive array: " + array);
        }
    }

    private static int dimensionOfPostRenameConcreteTypeObjectArray(NameStyle style, String array) {
        String unwrappedArray = ArrayRenamer.stripArrayWrapperPrefix(style, array);
        return numberOfLeading('$', unwrappedArray);
    }

    private static int dimensionOfPostRenameUnifyingTypeObjectArray(NameStyle style, String array) {
        String unwrappedArray = ArrayRenamer.stripArrayWrapperPrefix(style, array);
        return numberOfLeading('_', unwrappedArray);
    }

    public static int dimensionOfPostRenameObjectArray(NameStyle style, String array) {
        if (isPostRenameConcreteTypeObjectArray(style, array)) {
            return dimensionOfPostRenameConcreteTypeObjectArray(style, array);
        } else if (isPostRenameUnifyingTypeObjectArray(style, array)) {
            return dimensionOfPostRenameUnifyingTypeObjectArray(style, array);
        } else {
            throw RuntimeAssertionError.unreachable("Expected post-rename object array: " + array);
        }
    }

    public static int dimensionOfPostRenameArray(NameStyle style, String array) {
        if (isPostRenamePrimitiveArray(style, array)) {
            return dimensionOfPostRenamePrimitiveArray(style, array);
        } else if (isPostRenameObjectArray(style, array)) {
            return dimensionOfPostRenameObjectArray(style, array);
        } else {
            throw RuntimeAssertionError.unreachable("Expected post-rename array: " + array);
        }
    }

    public static int dimensionOfArray(NameStyle style, String array) {
        if (isPreRenameArray(array)) {
            return dimensionOfPreRenameArray(array);
        } else if (isPostRenameArray(style, array)) {
            return dimensionOfPostRenameArray(style, array);
        } else {
            throw RuntimeAssertionError.unreachable("Expected array: " + array);
        }
    }

    private static int numberOfLeading(char leadingChar, String string) {
        int count = 0;

        for (int i = 0; i < string.length(); i++) {
            if (string.charAt(i) != leadingChar) {
                return count;
            }
            count++;
        }

        return count;
    }
}
