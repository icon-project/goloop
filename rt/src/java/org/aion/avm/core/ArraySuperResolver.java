package org.aion.avm.core;

import org.aion.avm.ArrayRenamer;
import org.aion.avm.ArrayUtil;
import org.aion.avm.NameStyle;
import org.aion.avm.core.exceptionwrapping.ExceptionWrapperNameMapper;
import org.aion.avm.core.types.ClassHierarchy;
import org.aion.avm.core.types.CommonType;
import i.RuntimeAssertionError;

/**
 * A class that is used to determine a tightest common super class of two types, where at least one
 * of the two types must be an array type (either pre- or post- rename, it does not matter).
 *
 * If this resolver as well as {@link PlainTypeSuperResolver} and {@link PlainTypeSuperResolver} are
 * called on any two types then at least one of the resolvers will return a non-null and therefore
 * valid answer.
 */
public final class ArraySuperResolver {
    private final ClassHierarchy classHierarchy;
    private final ClassRenamer classRenamer;
    private final PlainTypeSuperResolver plainTypeSuperResolver;

    public ArraySuperResolver(ClassHierarchy classHierarchy, ClassRenamer classRenamer) {
        if (classHierarchy == null) {
            throw new NullPointerException("Cannot construct ArraySuperResolver with null class hierarchy.");
        }
        if (classRenamer == null) {
            throw new NullPointerException("Cannot construct ArraySuperResolver with null class renamer.");
        }
        this.classHierarchy = classHierarchy;
        this.classRenamer = classRenamer;
        this.plainTypeSuperResolver = new PlainTypeSuperResolver(this.classHierarchy, this.classRenamer);
    }

    /**
     * Returns a tightest common super class of the two given types if at least one of them is an
     * array type.
     *
     * Returns null if neither of the two given types are arrays.
     *
     * @param type1dotName The first type.
     * @param type2dotName The second type.
     * @return a tightest common super class or null.
     */
    public String getTightestSuperClassIfGivenArray(String type1dotName, String type2dotName) {
        RuntimeAssertionError.assertTrue(type1dotName != null);
        RuntimeAssertionError.assertTrue(type2dotName != null);

        boolean type1isArray = ArrayUtil.isArray(NameStyle.DOT_NAME, type1dotName);
        boolean type2isArray = ArrayUtil.isArray(NameStyle.DOT_NAME, type2dotName);
        boolean type1isSpecialArray = type1isArray && ArrayUtil.isSpecialPostRenameArray(NameStyle.DOT_NAME, type1dotName);
        boolean type2isSpecialArray = type2isArray && ArrayUtil.isSpecialPostRenameArray(NameStyle.DOT_NAME, type2dotName);
        boolean atLeastOneSpecialArray = type1isSpecialArray || type2isSpecialArray;

        if (type1isArray && type2isArray) {
            return atLeastOneSpecialArray ? findSuperOfAtLeastOneSpecialArray(type1dotName, type2dotName) : findSuperOfTwoArrays(type1dotName, type2dotName);
        } else if (type1isArray || type2isArray) {
            return atLeastOneSpecialArray ? findSuperOfAtLeastOneSpecialArray(type1dotName, type2dotName) : findSuperOfArrayAndNonArray(type1dotName, type2dotName);
        } else {
            return null;
        }
    }

    /**
     * A special array is one of our internal types: Array, IArray, ObjectArray, IObjectArray
     *
     * Assumption: at least one of the two arrays is a special array.
     */
    private String findSuperOfAtLeastOneSpecialArray(String array1, String array2) {
        boolean array1isSpecialArray = ArrayUtil.isSpecialPostRenameArray(NameStyle.DOT_NAME, array1);
        boolean array2isSpecialArray = ArrayUtil.isSpecialPostRenameArray(NameStyle.DOT_NAME, array2);

        if (array1isSpecialArray ^ array2isSpecialArray) {
            return findSuperOfSpecialArrayAndOther(array1, array2);
        } else {
            // Special arrays are in the hierarchy, we can query it directly.
            String commonSuper = this.classHierarchy.getTightestCommonSuperClass(array1, array2);

            // If the super class is ambiguous return IObject, otherwise return the super class.
            if (commonSuper == null) {
                return CommonType.I_OBJECT.dotName;
            } else {
                return commonSuper;
            }
        }
    }

    /**
     * Returns a tightest common super class of the two types, under the assumption that both of them
     * are arrays.
     */
    private String findSuperOfTwoArrays(String array1dotName, String array2dotName) {
        boolean array1isPreRename = this.classRenamer.isPreRename(array1dotName);
        boolean array2isPreRename = this.classRenamer.isPreRename(array2dotName);

        if (array1isPreRename && array2isPreRename) {
            return findSuperOfTwoPreRenameArrays(array1dotName, array2dotName);
        } else if (!array1isPreRename && !array2isPreRename) {
            return findSuperOfTwoPostRenameArrays(array1dotName, array2dotName);
        } else {
            // We have a pre- and post-rename array, they can only unify to java.lang.Object
            return CommonType.JAVA_LANG_OBJECT.dotName;
        }
    }

    /**
     * Returns a tightest common super class of the two types, under the assumption that one of them
     * is an array and the other is not.
     *
     * Returns java.lang.Object when:
     * 1. array is pre-rename.
     * 2. one type is pre-rename, other is post-rename.
     * 3. array is post-rename and the non-array is an exception wrapper.
     *
     * Returns IObject when:
     * 1. both types are post-rename and either the array is a unifying object array or the non-array
     *    is an interface.
     *
     * Returns shadow Object otherwise.
     */
    private String findSuperOfArrayAndNonArray(String type1dotName, String type2dotName) {
        String arrayDotName = ArrayUtil.isArray(NameStyle.DOT_NAME, type1dotName) ? type1dotName : type2dotName;
        String nonArrayDotName = arrayDotName.equals(type1dotName) ? type2dotName : type1dotName;

        if (ArrayUtil.isPreRenameArray(arrayDotName)) {
            // If the array is pre-rename then it can only unify to java.lang.Object
            return CommonType.JAVA_LANG_OBJECT.dotName;
        } else {
            if (this.classRenamer.isPreRename(nonArrayDotName)) {
                // Then we have a pre- and post-rename unification, these unify to java.lang.Object
                return CommonType.JAVA_LANG_OBJECT.dotName;
            } else {

                // Exception wrappers descend from java.lang.Throwable so we unify to java.lang.Object
                if (ExceptionWrapperNameMapper.isExceptionWrapperDotName(nonArrayDotName)) {
                    return CommonType.JAVA_LANG_OBJECT.dotName;
                }

                boolean arrayIsUnifyingType = ArrayUtil.isPostRenameUnifyingTypeObjectArray(NameStyle.DOT_NAME, arrayDotName);
                boolean nonArrayIsInterface = this.classHierarchy.postRenameTypeIsInterface(nonArrayDotName);

                if (arrayIsUnifyingType || nonArrayIsInterface) {
                    return CommonType.I_OBJECT.dotName;
                } else {
                    return CommonType.SHADOW_OBJECT.dotName;
                }
            }
        }
    }

    private String findSuperOfTwoPreRenameArrays(String array1preRename, String array2preRename) {
        boolean array1isPrimitiveArray = ArrayUtil.isPreRenamePrimitiveArray(array1preRename);
        boolean array2isPrimitiveArray = ArrayUtil.isPreRenamePrimitiveArray(array2preRename);

        if (array1isPrimitiveArray && array2isPrimitiveArray) {
            // Primitive arrays unify to java.lang.Object only, unless they are equal.
            return array1preRename.equals(array2preRename) ? array1preRename : CommonType.JAVA_LANG_OBJECT.dotName;
        } else if (!array1isPrimitiveArray && !array2isPrimitiveArray) {
            return findSuperOfTwoPreRenameObjectArrays(array1preRename, array2preRename);
        } else {
            // We have a primitive and an object array, they can only unify to java.lang.Object
            return ArrayRenamer.prependPreRenameObjectArrayPrefix(CommonType.JAVA_LANG_OBJECT.dotName, 1);
        }
    }

    private String findSuperOfTwoPostRenameArrays(String array1postRename, String array2postRename) {
        boolean array1isPrimitiveArray = ArrayUtil.isPostRenamePrimitiveArray(NameStyle.DOT_NAME, array1postRename);
        boolean array2isPrimitiveArray = ArrayUtil.isPostRenamePrimitiveArray(NameStyle.DOT_NAME, array2postRename);

        if (array1isPrimitiveArray && array2isPrimitiveArray) {
            // Primitive arrays unify to shadow Object only, unless they are equal.
            return (array1postRename.equals(array2postRename)) ? array1postRename : CommonType.SHADOW_OBJECT.dotName;
        } else if (!array1isPrimitiveArray && !array2isPrimitiveArray) {
            return findSuperOfTwoPostRenameObjectArrays(array1postRename, array2postRename);
        } else {
            // We have a primitive and object array. They unify to IObject if the object array is a unifying type.
            // Otherwise, they unify to shadow Object.
            boolean objectIsUnifyingArray = (array1isPrimitiveArray)
                ? ArrayUtil.isPostRenameUnifyingTypeObjectArray(NameStyle.DOT_NAME, array2postRename)
                : ArrayUtil.isPostRenameUnifyingTypeObjectArray(NameStyle.DOT_NAME, array1postRename);

            return objectIsUnifyingArray ? CommonType.I_OBJECT.dotName : CommonType.SHADOW_OBJECT.dotName;
        }
    }

    private String findSuperOfTwoPostRenameObjectArrays(String array1postRename, String array2postRename) {
        int dimension1 = ArrayUtil.dimensionOfPostRenameObjectArray(NameStyle.DOT_NAME, array1postRename);
        int dimension2 = ArrayUtil.dimensionOfPostRenameObjectArray(NameStyle.DOT_NAME, array2postRename);

        if (dimension1 == dimension2) {
            // They have the same dimension so we get an array of the super class of their base types.
            String baseType1 = ArrayRenamer.getPostRenameObjectArrayWrapperUnderlyingTypeName(NameStyle.DOT_NAME, array1postRename);
            String baseType2 = ArrayRenamer.getPostRenameObjectArrayWrapperUnderlyingTypeName(NameStyle.DOT_NAME, array2postRename);

            // We cannot have an array of array or array of exception wrapper so we have two plain types.
            String superOfBases = this.plainTypeSuperResolver.getTightestSuperClassIfGivenPlainType(baseType1, baseType2);
            RuntimeAssertionError.assertTrue(superOfBases != null);

            // IObject is given when super is ambiguous, in this case we have to wrap as a unifying type.
            boolean superIsIObject = superOfBases.equals(CommonType.I_OBJECT.dotName);
            boolean array1isConcreteType = ArrayUtil.isPostRenameConcreteTypeObjectArray(NameStyle.DOT_NAME, array1postRename);
            boolean array2isConcreteType = ArrayUtil.isPostRenameConcreteTypeObjectArray(NameStyle.DOT_NAME, array2postRename);

            if (!superIsIObject && array1isConcreteType && array2isConcreteType) {
                // We have two concrete types array, they must unify to a concrete type array.
                return superOfBases.equals(CommonType.SHADOW_OBJECT.dotName)
                    ? CommonType.SHADOW_OBJECT.dotName
                    : ArrayRenamer.wrapAsConcreteObjectArray(NameStyle.DOT_NAME, superOfBases, dimension1);

            } else {
                // A unifying array is present so the two must unify to a unifying type.
                return ArrayRenamer.wrapAsUnifyingObjectArray(NameStyle.DOT_NAME, superOfBases, dimension1);
            }
        } else {
            // They differ in dimension, so they can only unify to IObject or shadow Object.
            boolean array1isUnifyingType = ArrayUtil.isPostRenameUnifyingTypeObjectArray(NameStyle.DOT_NAME, array1postRename);
            boolean array2isUnifyingType = ArrayUtil.isPostRenameUnifyingTypeObjectArray(NameStyle.DOT_NAME, array2postRename);

            if (array1isUnifyingType || array2isUnifyingType) {
                return CommonType.I_OBJECT.dotName;
            } else {
                return CommonType.SHADOW_OBJECT.dotName;
            }
        }
    }

    private String findSuperOfTwoPreRenameObjectArrays(String array1, String array2) {
        int dimension1 = ArrayUtil.dimensionOfPreRenameObjectArray(array1);
        int dimension2 = ArrayUtil.dimensionOfPreRenameObjectArray(array2);

        if (dimension1 == dimension2) {
            // They are the same dimension so these will have a complex unification.
            // We need the base types of each of the two arrays so that we can unify them.

            String baseType1 = ArrayRenamer.getPreRenameObjectArrayWrapperUnderlyingTypeName(array1);
            String baseType2 = ArrayRenamer.getPreRenameObjectArrayWrapperUnderlyingTypeName(array2);

            // We are left with two plain types as our base types.
            String superOfBases = this.plainTypeSuperResolver.getTightestSuperClassIfGivenPlainType(baseType1, baseType2);
            RuntimeAssertionError.assertTrue(superOfBases != null);

            return ArrayRenamer.prependPreRenameObjectArrayPrefix(superOfBases, dimension1);
        } else {
            // They differ in dimensions and can only unify to java.lang.Object
            return ArrayRenamer.prependPreRenameObjectArrayPrefix(CommonType.JAVA_LANG_OBJECT.dotName, 1);
        }
    }

    private String findSuperOfSpecialArrayAndOther(String type1dotName, String type2dotName) {
        String specialArray = (ArrayUtil.isSpecialPostRenameArray(NameStyle.DOT_NAME, type1dotName)) ? type1dotName : type2dotName;
        String otherType = specialArray.equals(type1dotName) ? type2dotName : type1dotName;

        if (this.classRenamer.isPreRename(otherType)) {
            return CommonType.JAVA_LANG_OBJECT.dotName;
        } else {

            // Both types are post-rename. Other may still be an array or an exception wrapper.
            if (ExceptionWrapperNameMapper.isExceptionWrapperDotName(otherType)) {
                // Exception wrappers descend from java.lang.Throwable so they can only unify to java.lang.Object here
                return CommonType.JAVA_LANG_OBJECT.dotName;
            } else if (ArrayUtil.isPostRenameArray(NameStyle.DOT_NAME, otherType)) {

                if (ArrayUtil.isPostRenameSingleDimensionPrimitiveArray(NameStyle.DOT_NAME, otherType)) {

                    // The primitive array unifies to the special array if it is its parent, otherwise it
                    // unifies to shadow Object or IObject depending on which special array we have.
                    if (specialArray.equals(CommonType.ARRAY.dotName) || specialArray.equals(CommonType.I_ARRAY.dotName)) {
                        return specialArray;
                    } else if (specialArray.equals(CommonType.OBJECT_ARRAY.dotName)) {
                        return CommonType.SHADOW_OBJECT.dotName;
                    } else if (specialArray.equals(CommonType.I_OBJECT_ARRAY.dotName)) {
                        return CommonType.I_OBJECT.dotName;
                    } else {
                        throw RuntimeAssertionError.unreachable("Expected a special array type: " + specialArray);
                    }

                } else if (ArrayUtil.isPostRenameMultiDimensionPrimitiveArray(NameStyle.DOT_NAME, otherType)) {

                    // Multi-dimensional primitive arrays unify to IObject if the special array is an interface, otherwise shadow Object
                    if (specialArray.equals(CommonType.I_OBJECT_ARRAY.dotName) || specialArray.equals(CommonType.I_ARRAY.dotName)) {
                        return CommonType.I_OBJECT.dotName;
                    } else if (specialArray.equals(CommonType.ARRAY.dotName) || specialArray.equals(CommonType.OBJECT_ARRAY.dotName)) {
                        return CommonType.SHADOW_OBJECT.dotName;
                    } else {
                        throw RuntimeAssertionError.unreachable("Expected a special array type: " + specialArray);
                    }

                } else if (ArrayUtil.isPostRenameConcreteTypeObjectArray(NameStyle.DOT_NAME, otherType)) {

                    // The concrete array unifies to the special array if it is its parent, otherwise it
                    // unifies to shadow Object or IObject depending on which special array we have.
                    if (specialArray.equals(CommonType.OBJECT_ARRAY.dotName) || specialArray.equals(CommonType.I_OBJECT_ARRAY.dotName)) {
                        return specialArray;
                    } else if (specialArray.equals(CommonType.ARRAY.dotName)) {
                        return CommonType.SHADOW_OBJECT.dotName;
                    } else if (specialArray.equals(CommonType.I_ARRAY.dotName)) {
                        return CommonType.I_OBJECT.dotName;
                    } else {
                        throw RuntimeAssertionError.unreachable("Expected a special array type: " + specialArray);
                    }

                } else if (ArrayUtil.isPostRenameUnifyingTypeObjectArray(NameStyle.DOT_NAME, otherType)) {

                    // A unifying array type unifies to its parent IObjectArray or else IObject
                    if (specialArray.equals(CommonType.I_OBJECT_ARRAY.dotName)) {
                        return specialArray;
                    } else if (specialArray.equals(CommonType.OBJECT_ARRAY.dotName) || specialArray.equals(CommonType.I_ARRAY.dotName) || specialArray.equals(CommonType.ARRAY.dotName)) {
                        return CommonType.I_OBJECT.dotName;
                    } else {
                        throw RuntimeAssertionError.unreachable("Expected a special array type: " + specialArray);
                    }

                } else {
                    throw RuntimeAssertionError.unreachable("Expected a post-rename array: " + otherType);
                }
            } else {

                // Then other is not an exception wrapper or an array, just a plain type. The two can
                // only unify to IObject and shadow Object
                boolean specialTypeIsInterface = specialArray.equals(CommonType.I_OBJECT_ARRAY.dotName) || specialArray.equals(CommonType.I_ARRAY.dotName);
                boolean otherTypeIsInterface = this.classHierarchy.postRenameTypeIsInterface(otherType);

                if (specialTypeIsInterface || otherTypeIsInterface) {
                    return CommonType.I_OBJECT.dotName;
                } else {
                    return CommonType.SHADOW_OBJECT.dotName;
                }
            }
        }
    }
}
