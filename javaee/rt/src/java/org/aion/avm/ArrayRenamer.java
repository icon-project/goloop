package org.aion.avm;

import i.PackageConstants;
import i.RuntimeAssertionError;

public final class ArrayRenamer {
    private static final String BASIC_DOT_WRAPPER = PackageConstants.kArrayWrapperDotPrefix;
    private static final String BASIC_SLASH_WRAPPER = PackageConstants.kArrayWrapperSlashPrefix;
    private static final String UNIFYING_DOT_WRAPPER = PackageConstants.kArrayWrapperUnifyingDotPrefix;
    private static final String UNIFYING_SLASH_WRAPPER = PackageConstants.kArrayWrapperUnifyingSlashPrefix;

    /**
     * Returns {@code name} but with the array wrapper prefix prepended to it.
     *
     * This is the basic array wrapper prefix, not the prefix used for unifying-type object arrays.
     * Use the {@code prependUnifyingArrayWrapperPrefix()} method for that.
     *
     * @param style The naming convention of name.
     * @param name The name to prepend.
     * @return name prepended with the array wrapper prefix.
     */
    public static String prependArrayWrapperPrefix(NameStyle style, String name) {
        RuntimeAssertionError.assertTrue(style != null);
        RuntimeAssertionError.assertTrue(name != null);
        return ((style == NameStyle.DOT_NAME) ? BASIC_DOT_WRAPPER : BASIC_SLASH_WRAPPER) + name;
    }

    /**
     * Returns {@code name} but with the unifying-type object array wrapper prefix prepended to it.
     *
     * This is not the basic array wrapper prefix. Use the {@code prependArrayWrapperPrefix()}
     * method for that.
     *
     * @param style The naming convention of name.
     * @param name The name to prepend.
     * @return name prepended with the unifying-type object array wrapper prefix.
     */
    public static String prependUnifyingArrayWrapperPrefix(NameStyle style, String name) {
        RuntimeAssertionError.assertTrue(style != null);
        RuntimeAssertionError.assertTrue(name != null);
        return ((style == NameStyle.DOT_NAME) ? UNIFYING_DOT_WRAPPER : UNIFYING_SLASH_WRAPPER) + name;
    }

    /**
     * Returns the provided underlyingType prepended with a prefix that denotes the following:
     *
     * 1. the dimension of the array is equal to {@code dimension}.
     * 2. the array is a 'concrete type' object array.
     *
     * @param style The naming convention of name.
     * @param underlyingType The underlying type to be wrapped in the array.
     * @param dimension The dimension of the array.
     * @return underlyingType prepended with the concrete-type object array wrapper prefix denoting dimensionality.
     */
    public static String wrapAsConcreteObjectArray(NameStyle style, String underlyingType, int dimension) {
        RuntimeAssertionError.assertTrue(underlyingType != null);

        if (dimension > 0) {
            String unwrappedName = stringOfSameCharacter('$', dimension) + "L" + underlyingType;
            return prependArrayWrapperPrefix(style, unwrappedName);
        } else {
            throw RuntimeAssertionError.unreachable("Expected dimension of at least 1: " + dimension);
        }
    }

    /**
     * Returns the provided underlyingType prepended with a prefix that denotes the following:
     *
     * 1. the dimension of the array is equal to {@code dimension}.
     * 2. the array is a 'unifying type' object array.
     *
     * @param style The naming convention of name.
     * @param underlyingType The underlying type to be wrapped in the array.
     * @param dimension The dimension of the array.
     * @return underlyingType prepended with the unifying-type object array wrapper prefix denoting dimensionality.
     */
    public static String wrapAsUnifyingObjectArray(NameStyle style, String underlyingType, int dimension) {
        RuntimeAssertionError.assertTrue(underlyingType != null);

        if (dimension > 0) {
            String unwrappedName = stringOfSameCharacter('_', dimension) + "L" + underlyingType;
            return prependUnifyingArrayWrapperPrefix(style, unwrappedName);
        } else {
            throw RuntimeAssertionError.unreachable("Expected dimension of at least 1: " + dimension);
        }
    }

    /**
     * Returns array but with the array wrapper prefix stripped from it. Note that if the prefix is
     * the unifying-type array prefix or the basic prefix it does not matter, either one is eligible
     * for this method and either one will be removed if supplied.
     *
     * @param style The naming convention of name.
     * @param array The array whose wrapper prefix is to be stripped.
     * @return array without the array wrapper prefix.
     */
    public static String stripArrayWrapperPrefix(NameStyle style, String array) {
        RuntimeAssertionError.assertTrue(style != null);
        RuntimeAssertionError.assertTrue(array != null);

        String basicPrefix = (style == NameStyle.DOT_NAME) ? BASIC_DOT_WRAPPER : BASIC_SLASH_WRAPPER;
        String unifyingPrefix = (style == NameStyle.DOT_NAME) ? UNIFYING_DOT_WRAPPER : UNIFYING_SLASH_WRAPPER;

        // Check the unifying prefix first since the basic is a prefix itself of the unifying prefix!
        if (array.startsWith(unifyingPrefix)) {
            return array.substring(unifyingPrefix.length());
        } else if (array.startsWith(basicPrefix)) {
            return array.substring(basicPrefix.length());
        } else {
            throw RuntimeAssertionError.unreachable("Expected an array wrapper: " + array);
        }
    }

    /**
     * Returns name of the object that the given array is an array of. That is, if array is an array
     * of some type T then this method returns T (more precisely, the name of T).
     *
     * This is not equivalent to stripping the array wrapper prefix, since the stripped name may
     * still contain meta-data (for example, some leading tokens that describe the dimensionality
     * of the array) relating to the underlying type, whereas this method simply returns the
     * underlying type itself.
     *
     * Note that array must be an array wrapper (that is, a post-rename array).
     *
     * @param style The naming convention of name.
     * @param array The array whose underlying type's name is to be returned.
     * @return the underlying array type's name.
     */
    public static String getPostRenameObjectArrayWrapperUnderlyingTypeName(NameStyle style, String array) {
        RuntimeAssertionError.assertTrue(ArrayUtil.isPostRenameObjectArray(style, array));

        int dimension = ArrayUtil.dimensionOfPostRenameObjectArray(style, array);
        String strippedName = stripArrayWrapperPrefix(style, array);

        // Remove the leading dimension-characters and the 'L' character that follows them.
        return strippedName.substring(dimension + 1);
    }

    /**
     * Returns the name of the object that the given array is an array of. That is, if array is an
     * array of some type T then this method returns T (more precisely, the name of T).
     *
     * @param array The pre-rename object array.
     * @return the underlying type.
     */
    public static String getPreRenameObjectArrayWrapperUnderlyingTypeName(String array) {
        RuntimeAssertionError.assertTrue(ArrayUtil.isPreRenameObjectArray(array));

        int dimension = ArrayUtil.dimensionOfPreRenameObjectArray(array);

        // The +1 is to account for the 'L' preceding the type name.
        return array.substring(dimension + 1);
    }

    /**
     * Returns a pre-rename object array that is an array of the given base type, with the given
     * dimensionality.
     *
     * @param baseType The base type to make an array of.
     * @param dimension The dimension of the array.
     * @return the pre-rename object array.
     */
    public static String prependPreRenameObjectArrayPrefix(String baseType, int dimension) {
        String leadingDimensionChars = stringOfSameCharacter('[', dimension);
        return leadingDimensionChars + "L" + baseType;
    }

    /**
     * Returns a string of length {@code length}, each of whose characters is the same: namely, each
     * character is {@code character}.
     */
    private static String stringOfSameCharacter(char character, int length) {
        if (character == '$') {
            return new String(new char[length]).replaceAll("\0", "\\$");
        } else if (character == '[') {
            return new String(new char[length]).replaceAll("\0", "\\[");
        } else {
            return new String(new char[length]).replaceAll("\0", String.valueOf(character));
        }
    }
}
