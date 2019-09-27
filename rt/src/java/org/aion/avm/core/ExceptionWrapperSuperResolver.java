package org.aion.avm.core;

import org.aion.avm.core.ClassRenamer.ArrayType;
import org.aion.avm.core.exceptionwrapping.ExceptionWrapperNameMapper;
import org.aion.avm.core.types.ClassHierarchy;
import org.aion.avm.core.types.CommonType;
import i.RuntimeAssertionError;

/**
 * A class that is used to determine a tightest common super class of two types, where at least one
 * of the two types must be an exception wrapper.
 *
 * If this resolver as well as {@link ArraySuperResolver} and {@link PlainTypeSuperResolver} are
 * called on any two types then at least one of the resolvers will return a non-null and therefore
 * valid answer.
 */
public final class ExceptionWrapperSuperResolver {
    private final ClassHierarchy classHierarchy;
    private final ClassRenamer classRenamer;

    public ExceptionWrapperSuperResolver(ClassHierarchy classHierarchy, ClassRenamer classRenamer) {
        if (classHierarchy == null) {
            throw new NullPointerException("Cannot construct ExceptionWrapperSuperResolver with null class hierarchy.");
        }
        if (classRenamer == null) {
            throw new NullPointerException("Cannot construct ExceptionWrapperSuperResolver with null class renamer.");
        }
        this.classHierarchy = classHierarchy;
        this.classRenamer = classRenamer;
    }

    public String getTightestSuperClassIfGivenPlainType(String type1dotName, String type2dotName) {
        RuntimeAssertionError.assertTrue(type1dotName != null);
        RuntimeAssertionError.assertTrue(type2dotName != null);

        boolean type1isExceptionWrapper = ExceptionWrapperNameMapper.isExceptionWrapperDotName(type1dotName);
        boolean type2isExceptionWrapper = ExceptionWrapperNameMapper.isExceptionWrapperDotName(type2dotName);

        if (type1isExceptionWrapper && type2isExceptionWrapper) {
            return findSuperOfTwoExceptionWrappers(type1dotName, type2dotName);
        } else if (type1isExceptionWrapper || type2isExceptionWrapper) {
            return findSuperOfOneExceptionWrapperOneNonExceptionWrapper(type1dotName, type2dotName);
        } else {
            return null;
        }
    }

    private String findSuperOfTwoExceptionWrappers(String wrapper1, String wrapper2) {
        // We unwrap the exceptions and we are left with plain types that are in the hierarchy.
        String unwrapped1 = this.classRenamer.toPreRename(wrapper1);
        String unwrapped2 = this.classRenamer.toPreRename(wrapper2);

        String unwrappedSuper = this.classHierarchy.getTightestCommonSuperClass(unwrapped1, unwrapped2);

        // If the super class is ambiguous return java.lang.Throwable, otherwise wrap the super class and return it.
        if (unwrappedSuper == null) {
            return CommonType.JAVA_LANG_THROWABLE.dotName;
        } else {
            return this.classRenamer.toExceptionWrapper(unwrappedSuper);
        }
    }

    private String findSuperOfOneExceptionWrapperOneNonExceptionWrapper(String type1dotName, String type2dotName) {
        String exceptionWrapper = ExceptionWrapperNameMapper.isExceptionWrapperDotName(type1dotName) ? type1dotName : type2dotName;
        String otherType = exceptionWrapper.equals(type1dotName) ? type2dotName : type1dotName;

        if (this.classRenamer.isPreRename(otherType)) {
            // If otherType is an exception then java.lang.Throwable otherwise java.lang.Object

            // The hierarchy only deals with post-rename names so we check against shadow Throwable for exception types.
            String postRenameOther = this.classRenamer.toPostRename(otherType, ArrayType.PRECISE_TYPE);
            if (this.classHierarchy.isDescendantOfClass(postRenameOther, CommonType.SHADOW_THROWABLE.dotName)) {
                return CommonType.JAVA_LANG_THROWABLE.dotName;
            } else {
                return CommonType.JAVA_LANG_OBJECT.dotName;
            }

        } else {
            // exception wrappers descend from java.lang.Throwable, so they unify to java.lang.Object with a post-rename class.
            return CommonType.JAVA_LANG_OBJECT.dotName;
        }
    }
}
