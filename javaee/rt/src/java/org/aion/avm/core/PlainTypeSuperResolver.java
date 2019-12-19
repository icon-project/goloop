package org.aion.avm.core;

import org.aion.avm.ArrayUtil;
import org.aion.avm.NameStyle;
import org.aion.avm.core.ClassRenamer.ArrayType;
import org.aion.avm.core.exceptionwrapping.ExceptionWrapperNameMapper;
import org.aion.avm.core.types.ClassHierarchy;
import org.aion.avm.core.types.CommonType;
import i.RuntimeAssertionError;

/**
 * A class that is used to determine a tightest common super class of two types, where at least one
 * of the two types must be a 'plain type'.
 *
 * A 'plain type' is defined as any type that is not an array type (pre- or post- rename array) AND
 * is not an exception wrapper.
 *
 * If this resolver as well as {@link ArraySuperResolver} and {@link ExceptionWrapperSuperResolver}
 * are called on any two types then at least one of the resolvers will return a non-null and therefore
 * valid answer.
 */
public final class PlainTypeSuperResolver {
    private final ClassHierarchy classHierarchy;
    private final ClassRenamer classRenamer;

    public PlainTypeSuperResolver(ClassHierarchy classHierarchy, ClassRenamer classRenamer) {
        if (classHierarchy == null) {
            throw new NullPointerException("Cannot construct PlainTypeSuperResolver with null class hierarchy.");
        }
        if (classRenamer == null) {
            throw new NullPointerException("Cannot construct PlainTypeSuperResolver with null class renamer.");
        }
        this.classHierarchy = classHierarchy;
        this.classRenamer = classRenamer;
    }

    /**
     * Returns a tightest common super class of the two given types if at least one of them is a
     * plain type, where a plain type is any type that is NOT an exception wrapper AND NOT an array.
     *
     * Returns null if neither of the two given types are plain types.
     *
     * @param type1dotName The first type.
     * @param type2dotName The second type.
     * @return a tightest common super class or null.
     */
    public String getTightestSuperClassIfGivenPlainType(String type1dotName, String type2dotName) {
        RuntimeAssertionError.assertTrue(type1dotName != null);
        RuntimeAssertionError.assertTrue(type2dotName != null);

        boolean type1isPlain = isPlainType(type1dotName);
        boolean type2isPlain = isPlainType(type2dotName);

        if (type1isPlain && type2isPlain) {
            return findSuperOfTwoPlainTypes(type1dotName, type2dotName);
        } else if (type1isPlain || type2isPlain) {
            return findSuperOfOnePlainTypeOneNonPlainType(type1dotName, type2dotName);
        } else {
            return null;
        }
    }

    /**
     * Returns a tightest common super class of the two plain types.
     *
     * If the tightest common super class is ambiguous (ie. there are multiples) then IObject is
     * returned.
     */
    private String findSuperOfTwoPlainTypes(String plain1dotName, String plain2dotName) {
        boolean plain1isPreRename = this.classRenamer.isPreRename(plain1dotName);
        boolean plain2isPreRename = this.classRenamer.isPreRename(plain2dotName);

        if (plain1isPreRename && plain2isPreRename) {
            // The hierarchy is post-rename only so we have to do some renaming.
            String plain1renamed = this.classRenamer.toPostRename(plain1dotName, ArrayType.PRECISE_TYPE);
            String plain2renamed = this.classRenamer.toPostRename(plain2dotName, ArrayType.PRECISE_TYPE);

            String postRenameSuper = this.classHierarchy.getTightestCommonSuperClass(plain1renamed, plain2renamed);

            // If the super class is ambiguous return java.lang.Object, otherwise convert the super class to pre-rename and return it.
            if (postRenameSuper == null) {
                return CommonType.JAVA_LANG_OBJECT.dotName;
            } else {
                return this.classRenamer.toPreRename(postRenameSuper);
            }

        } else if (!plain1isPreRename && !plain2isPreRename) {
            String commonSuper = this.classHierarchy.getTightestCommonSuperClass(plain1dotName, plain2dotName);

            // If the super class is ambiguous return IObject, otherwise return the super class.
            if (commonSuper == null) {
                return CommonType.I_OBJECT.dotName;
            } else {
                return commonSuper;
            }

        } else {
            // Then we have a pre- and post- rename plain type, they can only unify to java.lang.Object
            return CommonType.JAVA_LANG_OBJECT.dotName;
        }
    }

    /**
     * Returns a tightest common super class of the plain type and non-plain type.
     *
     * Returns java.lang.Throwable when:
     * 1. The plain type is pre-rename and the non-plain type is java.lang.Throwable or an exception wrapper.
     *
     * Returns java.lang.Object when:
     * 1. The plain type is pre-rename and the non-plain type is an array.
     * 2. The plain type is post-rename and the non-plain type is pre-rename.
     * 3. The plain type is post-rename and the non-plain type is an exception wrapper.
     *
     * Returns IObject when:
     * 1. The plain type is a post-rename interface type and the non-plain type is a post-rename array.
     * 2. The plain type is a post-rename class type and the non-plain type is a unifying array type.
     *
     * Returns shadow Object otherwise.
     */
    private String findSuperOfOnePlainTypeOneNonPlainType(String plain1dotName, String plain2dotName) {
        String plainDotName = isPlainType(plain1dotName) ? plain1dotName : plain2dotName;
        String nonPlainDotName = plainDotName.equals(plain1dotName) ? plain2dotName : plain1dotName;

        boolean plainIsPreRename = this.classRenamer.isPreRename(plainDotName);
        boolean nonPlainIsPreRename = this.classRenamer.isPreRename(nonPlainDotName);

        if (plainIsPreRename) {

            // We convert to post-rename to query the hierarchy because it accepts only post-rename names.
            String plainPostRename = this.classRenamer.toPostRename(plainDotName, ArrayType.PRECISE_TYPE);
            boolean plainIsException = this.classHierarchy.isDescendantOfClass(plainPostRename, CommonType.SHADOW_THROWABLE.dotName);
            boolean nonPlainIsException = nonPlainDotName.equals(CommonType.JAVA_LANG_THROWABLE.dotName) || ExceptionWrapperNameMapper.isExceptionWrapperDotName(nonPlainDotName);

            if (plainIsException && nonPlainIsException) {
                return CommonType.JAVA_LANG_THROWABLE.dotName;
            } else {
                return CommonType.JAVA_LANG_OBJECT.dotName;
            }

        } else {

            if (nonPlainIsPreRename) {
                return CommonType.JAVA_LANG_OBJECT.dotName;
            } else {

                boolean plainIsInterface = this.classHierarchy.postRenameTypeIsInterface(plainDotName);

                if (ExceptionWrapperNameMapper.isExceptionWrapperDotName(nonPlainDotName)) {
                    return CommonType.JAVA_LANG_OBJECT.dotName;
                } else if (ArrayUtil.isPostRenameUnifyingTypeObjectArray(NameStyle.DOT_NAME, nonPlainDotName)) {
                    return CommonType.I_OBJECT.dotName;
                } else if (ArrayUtil.isPostRenameConcreteTypeObjectArray(NameStyle.DOT_NAME, nonPlainDotName)) {
                    return (plainIsInterface) ? CommonType.I_OBJECT.dotName : CommonType.SHADOW_OBJECT.dotName;
                } else if (ArrayUtil.isPostRenamePrimitiveArray(NameStyle.DOT_NAME, nonPlainDotName)) {
                    return (plainIsInterface) ? CommonType.I_OBJECT.dotName : CommonType.SHADOW_OBJECT.dotName;
                } else {
                    throw RuntimeAssertionError.unreachable("Expected an exception wrapper or post-rename array: " + nonPlainDotName);
                }
            }
        }
    }

    private static boolean isPlainType(String dotName) {
        return !ArrayUtil.isArray(NameStyle.DOT_NAME, dotName) && !ExceptionWrapperNameMapper.isExceptionWrapperDotName(dotName);
    }
}
