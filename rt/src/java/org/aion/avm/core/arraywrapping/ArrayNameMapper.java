package org.aion.avm.core.arraywrapping;

import org.aion.avm.ArrayClassNameMapper;
import org.aion.avm.NameStyle;
import org.aion.avm.core.rejection.RejectedClassException;
import org.aion.avm.ArrayUtil;
import org.aion.avm.core.util.DescriptorParser;
import i.PackageConstants;
import i.RuntimeAssertionError;

import java.util.Set;
import java.util.regex.Pattern;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public class ArrayNameMapper {

    static private Pattern IOBJECT_INTERFACE_FORMAT = Pattern.compile("[_]{2,}Li/IObject");

    static private Set<String> PRIMITIVES = Stream.of("I", "J", "Z", "B", "S", "D", "F", "C").collect(Collectors.toSet());
    static private Pattern OBJECT_INTERFACE_FORMAT = Pattern.compile("[_\\[]{2,}Ls/java/lang/Object");


    static java.lang.String updateMethodDesc(java.lang.String desc) {
        return mapDescriptor(desc);
    }

    public static String getOriginalNameOf(String array) {
        if (ArrayUtil.isPostRenamePrimitiveArray(NameStyle.SLASH_NAME, array)) {
            return getOriginalNameOfPrimitiveArray(array);
        } else if (ArrayUtil.isPostRenameObjectArray(NameStyle.SLASH_NAME, array)) {
            return getOriginalNameOfObjectArray(array);
        } else {
            throw RuntimeAssertionError.unreachable("Expected post-rename slash-style array: " + array);
        }
    }

    private static String getOriginalNameOfPrimitiveArray(String primitiveArray) {
        if (ArrayUtil.isPostRenameSingleDimensionPrimitiveArray(NameStyle.SLASH_NAME, primitiveArray)) {
            return getOriginalNameOfPrimitiveArray1D(primitiveArray);
        } else if (ArrayUtil.isPostRenameMultiDimensionPrimitiveArray(NameStyle.SLASH_NAME, primitiveArray)) {
            return getOriginalNameOfPrimitiveArrayMD(primitiveArray);
        } else {
            throw RuntimeAssertionError.unreachable("Expected post-rename slash-style primitive array: " + primitiveArray);
        }
    }

    private static String getOriginalNameOfObjectArray(String objectArray) {
        if (ArrayUtil.isPostRenameConcreteTypeObjectArray(NameStyle.SLASH_NAME, objectArray)) {
            return getOriginalNameOfPreciseTypeObjectArray(objectArray);
        } else if (ArrayUtil.isPostRenameUnifyingTypeObjectArray(NameStyle.SLASH_NAME, objectArray)) {
            return getOriginalNameOfUnifyingTypeObjectArray(objectArray);
        } else {
            throw RuntimeAssertionError.unreachable("Expected post-rename slash-style object array: " + objectArray);
        }
    }

    private static String getOriginalNameOfPrimitiveArray1D(String primitiveArray1D) {
        String name = ArrayClassNameMapper.getOriginalNameFromWrapper(primitiveArray1D);
        if(name == null){
            throw RuntimeAssertionError.unreachable("Expected post-rename slash-style 1-dimension primitive array: " + primitiveArray1D);
        }
        return name;
    }

    private static String getOriginalNameOfPrimitiveArrayMD(String primitiveArrayMD) {
        if (primitiveArrayMD.startsWith(PackageConstants.kArrayWrapperSlashPrefix)) {
            String unwrappedArray = primitiveArrayMD.substring(PackageConstants.kArrayWrapperSlashPrefix.length());
            return unwrappedArray.replaceAll("\\$", "\\[");
        } else {
            throw RuntimeAssertionError.unreachable("Expected post-rename slash-style multi-dimension primitive array: " + primitiveArrayMD);
        }
    }

    private static String getOriginalNameOfPreciseTypeObjectArray(String preciseObjectArray) {
        if (preciseObjectArray.startsWith(PackageConstants.kArrayWrapperSlashPrefix)) {
            String unwrappedArray = preciseObjectArray.substring(PackageConstants.kArrayWrapperSlashPrefix.length());
            return unwrappedArray.replaceAll("\\$", "\\[");
        } else {
            throw RuntimeAssertionError.unreachable("Expected post-rename slash-style 'precise type' object array: " + preciseObjectArray);
        }
    }

    private static String getOriginalNameOfUnifyingTypeObjectArray(String unifyingObjectArray) {
        if (unifyingObjectArray.startsWith(PackageConstants.kArrayWrapperUnifyingSlashPrefix)) {
            String unwrappedArray = unifyingObjectArray.substring(PackageConstants.kArrayWrapperUnifyingSlashPrefix.length());
            return unwrappedArray.replaceAll("_", "\\[");
        } else {
            throw RuntimeAssertionError.unreachable("Expected post-rename slash-style 'unifying type' object array: " + unifyingObjectArray);
        }
    }

    // Return the wrapper descriptor of an array
    public static String getPreciseArrayWrapperDescriptor(String desc){
        return getClassWrapperDescriptor(desc);
    }

    // Return the wrapper descriptor of an array
    private static String getClassWrapperDescriptor(String desc){
        if (desc.endsWith(";")){
            desc = desc.substring(0, desc.length() - 1);
        }

        java.lang.String ret;
        if (desc.charAt(0) != '['){
            ret = desc;
        } else {
            validateArrayDimension(desc);
            ret = ArrayClassNameMapper.getClassWrapper(desc);
            if (ret == null) {
                ret = newClassWrapper(desc);
            }
        }
        return ret;
    }

    // Return the wrapper descriptor of an array
    static java.lang.String getInterfaceWrapper(java.lang.String desc){
        if (desc.endsWith(";")){
            desc = desc.substring(0, desc.length() - 1);
        }

        java.lang.String ret;

        if (desc.charAt(0) != '[') {
            ret = desc;
        } else {
            validateArrayDimension(desc);
            ret = ArrayClassNameMapper.getInterfaceWrapper(desc);
            if (ret == null) {
                ret = newInterfaceWrapper(desc);
            }
        }
        return ret;
    }

    private static java.lang.String newClassWrapper(java.lang.String desc){
        StringBuilder sb = new StringBuilder();
        sb.append(PackageConstants.kArrayWrapperSlashPrefix);

        //Check if the desc is a ref array
        if((desc.charAt(1) == 'L') || (desc.charAt(1) == '[')){
            sb.append(desc.replace('[', '$'));
        }else{
            throw RuntimeAssertionError.unreachable("newClassWrapper: " + desc);
        }

        return sb.toString();
    }

    private static java.lang.String newInterfaceWrapper(java.lang.String desc){
        if (ArrayUtil.isPreRenamePrimitiveArray(desc)) {
            return wrapperForPrimitiveArrays(desc);
        } else if (OBJECT_INTERFACE_FORMAT.matcher(desc).matches()) {
            return getMultiDimensionalObjectArrayDescriptor(desc);
        }

        StringBuilder sb = new StringBuilder();
        sb.append(PackageConstants.kArrayWrapperUnifyingSlashPrefix);

        //Check if the desc is a ref array
        if((desc.charAt(1) == 'L') || (desc.charAt(1) == '[')){
            sb.append(desc.replace('[', '_'));
        }else{
            throw RuntimeAssertionError.unreachable("newInterfaceWrapper :" + desc);
        }

        return sb.toString();
    }

    /**
     * Wraps primitive array descriptors of any dimensionality and returns their wrapped results.
     *
     * @param desc A primitive array descriptor.
     * @return The wrapped primitive array descriptor.
     */
    private static java.lang.String wrapperForPrimitiveArrays(java.lang.String desc) {
        int index = desc.lastIndexOf('[');
        index = (index == -1) ? desc.lastIndexOf('$') : index;
        int dimension = index + 1;

        if (dimension > 2) {
            return newClassWrapper(desc);
        } else {
            switch (desc.substring(dimension)) {
                case "B": return getByteArrayWrapper(dimension);
                case "Z": return getBooleanArrayWrapper(dimension);
                case "J": return getLongArrayWrapper(dimension);
                case "I": return getIntArrayWrapper(dimension);
                case "S": return getShortArrayWrapper(dimension);
                case "F": return getFloatArrayWrapper(dimension);
                case "D": return getDoubleArrayWrapper(dimension);
                case "C": return getCharArrayWrapper(dimension);
                default: return null;
            }
        }
    }

    /**
     * Converts a multi-dimensional java/lang/Object array descriptor into a same-dimensional array
     * interface descriptor of IObject instead of Object, which is now the proper unifying type for
     * Object arrays.
     *
     * @param descriptor A multi-dimensional Object array descriptor.
     * @return The unified array descriptor.
     */
    private static String getMultiDimensionalObjectArrayDescriptor(String descriptor) {
        int dim = descriptor.lastIndexOf('[') + 1;
        String dimPrefix = new String(new char[dim]).replace('\0', '_');
        return PackageConstants.kArrayWrapperUnifyingSlashPrefix + dimPrefix + "L"
                + PackageConstants.kInternalSlashPrefix + "IObject";
    }

    private static java.lang.String getObjectArrayWrapper(java.lang.String type, int dim){
        return getUnifyingArrayWrapperDescriptor(buildArrayDescriptor(dim, 'L' + type + ';'));
    }

    private static java.lang.String getByteArrayWrapper(int dim){
        return getClassWrapperDescriptor(buildArrayDescriptor(dim, "B"));
    }

    private static java.lang.String getCharArrayWrapper(int dim){
        return getClassWrapperDescriptor(buildArrayDescriptor(dim, "C"));
    }

    private static java.lang.String getIntArrayWrapper(int dim){
        return getClassWrapperDescriptor(buildArrayDescriptor(dim, "I"));
    }

    private static java.lang.String getDoubleArrayWrapper(int dim){
        return getClassWrapperDescriptor(buildArrayDescriptor(dim, "D"));
    }

    private static java.lang.String getFloatArrayWrapper(int dim){
        return getClassWrapperDescriptor(buildArrayDescriptor(dim, "F"));
    }

    private static java.lang.String getLongArrayWrapper(int dim){
        return getClassWrapperDescriptor(buildArrayDescriptor(dim, "J"));
    }

    private static java.lang.String getShortArrayWrapper(int dim){
        return getClassWrapperDescriptor(buildArrayDescriptor(dim, "S"));
    }

    private static java.lang.String getBooleanArrayWrapper(int dim){
        return getClassWrapperDescriptor(buildArrayDescriptor(dim, "Z"));
    }

    public static String buildArrayDescriptor(int length, String elementDescriptor) {
        return buildFullString(length, '[') + elementDescriptor;
    }

    private static String buildFullString(int length, char element) {
        return new String(new char[length]).replace('\0', element);
    }

    // Return the wrapper descriptor of an array
    public static String getUnifyingArrayWrapperDescriptor(String desc){
        // Note that we can't do any special unifying operation for primitive arrays so just handle them as "precise" types.
        boolean isPrimitiveArray = (2 == desc.length())
                && ('[' == desc.charAt(0))
                && (isPrimitiveElement(desc.substring(1)));
        return isPrimitiveArray
                ? getClassWrapperDescriptor(desc)
                : getInterfaceWrapper(desc);
    }

    public static String getElementInterfaceName(String interfaceClassName){
        // Get element class and array dim
        String elementName = interfaceClassName.substring((PackageConstants.kArrayWrapperUnifyingDotPrefix).length());
        int dim = getPrefixSize(elementName, '_');
        elementName = elementName.substring(dim);
        if (elementName.startsWith("L")){
            elementName = elementName.substring(1);
        }
        return  elementName;
    }

    public static String getClassWrapperElementName(String wrapperClassName){
        // Get element class and array dim
        String elementName = wrapperClassName.substring(PackageConstants.kArrayWrapperDotPrefix.length());
        int dim = getPrefixSize(elementName, '$');
        elementName = elementName.substring(dim);
        if (elementName.startsWith("L")){
            elementName = elementName.substring(1);
        }
        return elementName;
    }

    // Return the element type of an array
    // 1D Primitive array will not be called with this method since there will be no aaload
    static java.lang.String getElementType(java.lang.String desc){

        RuntimeAssertionError.assertTrue(desc.startsWith("["));
        String ret = desc.substring(1);

        if (ret.startsWith("L")){
            ret = ret.substring(1, ret.length() - 1);
        }

        return ret;
    }

    public static int getPrefixSize(String input, char prefixChar) {
        int d = 0;
        while (input.charAt(d) == prefixChar) {
            d++;
        }
        return d;
    }

    // Return the wrapper descriptor of an array
    static java.lang.String getFactoryDescriptor(java.lang.String wrapper, int d){
        String facDesc = buildFullString(d, 'I');
        facDesc = "(" + facDesc + ")L" + wrapper + ";";
        return facDesc;
    }

    private static String mapDescriptor(String descriptor) {
        StringBuilder builder = DescriptorParser.parse(descriptor, new DescriptorParser.Callbacks<>() {
            @Override
            public StringBuilder readObject(int arrayDimensions, String type, StringBuilder userData) {
                if (arrayDimensions > 0) {
                    userData.append(DescriptorParser.OBJECT_START);
                }
                userData.append(getObjectArrayWrapper(type, arrayDimensions));
                userData.append(DescriptorParser.OBJECT_END);
                return userData;
            }

            @Override
            public StringBuilder readBoolean(int arrayDimensions, StringBuilder userData) {
                if (arrayDimensions > 0) {
                    userData.append(DescriptorParser.OBJECT_START);
                    userData.append(getBooleanArrayWrapper(arrayDimensions));
                    userData.append(DescriptorParser.OBJECT_END);
                }else {
                    userData.append(DescriptorParser.BOOLEAN);
                }
                return userData;
            }

            @Override
            public StringBuilder readShort(int arrayDimensions, StringBuilder userData) {
                if (arrayDimensions > 0) {
                    userData.append(DescriptorParser.OBJECT_START);
                    userData.append(getShortArrayWrapper(arrayDimensions));
                    userData.append(DescriptorParser.OBJECT_END);
                }else{
                    userData.append(DescriptorParser.SHORT);
                }
                return userData;
            }

            @Override
            public StringBuilder readLong(int arrayDimensions, StringBuilder userData) {
                if (arrayDimensions > 0) {
                    userData.append(DescriptorParser.OBJECT_START);
                    userData.append(getLongArrayWrapper(arrayDimensions));
                    userData.append(DescriptorParser.OBJECT_END);
                }else{
                    userData.append(DescriptorParser.LONG);
                }
                return userData;
            }

            @Override
            public StringBuilder readInteger(int arrayDimensions, StringBuilder userData) {
                if (arrayDimensions > 0) {
                    userData.append(DescriptorParser.OBJECT_START);
                    userData.append(getIntArrayWrapper(arrayDimensions));
                    userData.append(DescriptorParser.OBJECT_END);
                }else{
                    userData.append(DescriptorParser.INTEGER);
                }
                return userData;
            }

            @Override
            public StringBuilder readFloat(int arrayDimensions, StringBuilder userData) {
                if (arrayDimensions > 0) {
                    userData.append(DescriptorParser.OBJECT_START);
                    userData.append(getFloatArrayWrapper(arrayDimensions));
                    userData.append(DescriptorParser.OBJECT_END);
                }else{
                    userData.append(DescriptorParser.FLOAT);
                }

                return userData;
            }

            @Override
            public StringBuilder readDouble(int arrayDimensions, StringBuilder userData) {
                if (arrayDimensions > 0) {
                    userData.append(DescriptorParser.OBJECT_START);
                    userData.append(getDoubleArrayWrapper(arrayDimensions));
                    userData.append(DescriptorParser.OBJECT_END);
                }else{
                    userData.append(DescriptorParser.DOUBLE);
                }
                return userData;
            }

            @Override
            public StringBuilder readChar(int arrayDimensions, StringBuilder userData) {
                if (arrayDimensions > 0) {
                    userData.append(DescriptorParser.OBJECT_START);
                    userData.append(getCharArrayWrapper(arrayDimensions));
                    userData.append(DescriptorParser.OBJECT_END);
                }else{
                    userData.append(DescriptorParser.CHAR);
                }
                return userData;
            }

            @Override
            public StringBuilder readByte(int arrayDimensions, StringBuilder userData) {
                if (arrayDimensions > 0) {
                    userData.append(DescriptorParser.OBJECT_START);
                    userData.append(getByteArrayWrapper(arrayDimensions));
                    userData.append(DescriptorParser.OBJECT_END);
                }else{
                    userData.append(DescriptorParser.BYTE);
                }
                return userData;
            }

            @Override
            public StringBuilder argumentStart(StringBuilder userData) {
                userData.append(DescriptorParser.ARGS_START);
                return userData;
            }
            @Override
            public StringBuilder argumentEnd(StringBuilder userData) {
                userData.append(DescriptorParser.ARGS_END);
                return userData;
            }
            @Override
            public StringBuilder readVoid(StringBuilder userData) {
                userData.append(DescriptorParser.VOID);
                return userData;
            }

        }, new StringBuilder());

        return unifyArraysInMethodDescriptor(builder.toString());
    }

    /**
     * Converts any array types in the method signature given by descriptor to their respective
     * unifying types.
     *
     * @param descriptor The method signature descriptor.
     * @return The descriptor with array types converted to their unifying types.
     */
    private static String unifyArraysInMethodDescriptor(String descriptor) {
        String[] splitDesc = descriptor.substring(1).split("\\)");
        return arrayParametersToUnifyingTypes(splitDesc[0]) + arrayReturnTypeToUnifyingType(splitDesc[1]);
    }

    /**
     * Returns the method parameter descriptor parameters but with each object array type promoted
     * to its unifying type.
     *
     * @param parameters A method parameter descriptor.
     * @return The method parameter descriptor with all object array types promoted to their unifying
     * types.
     */
    private static String arrayParametersToUnifyingTypes(String parameters) {
        StringBuilder builder = new StringBuilder("(");
        String token;

        int index = 0;
        while ((token = parameterAtIndex(parameters, index)) != null) {
            if (token.length() == 1) {
                builder.append(token);
            } else {
                builder.append(unifyArrayDescriptor(token));
            }
            index += token.length();
        }
        return builder.append(")").toString();
    }

    /**
     * Returns a method return type descriptor that is equivalent to methodType unless methodType
     * is an object array, in which case the array type is promoted to its unifying type.
     *
     * @param methodType A method return type descriptor.
     * @return The method return type descriptor with array object type unifying promotion.
     */
    private static String arrayReturnTypeToUnifyingType(String methodType) {
        return (methodType.length() == 1) ? methodType : unifyArrayDescriptor(methodType);
    }

    /**
     * Returns the next parameter descriptor in parameters, which must be a method parameter
     * descriptor with the leading ( and trailing ) removed, beginning at the index startIndex.
     *
     * Returns null if startIndex is larger than the largest index in parameters.
     *
     * @param parameters A method parameter descriptor.
     * @param startIndex The start of the next parameter.
     * @return The next parameter.
     */
    private static String parameterAtIndex(String parameters, int startIndex) {
        if (startIndex >= parameters.length()) { return null; }

        startIndex = (parameters.charAt(startIndex) == ';') ? startIndex + 1 : startIndex;
        if (PRIMITIVES.contains(String.valueOf(parameters.charAt(startIndex)))) {
            return String.valueOf(parameters.charAt(startIndex));
        }
        return parameters.substring(startIndex, parameters.indexOf(';', startIndex) + 1);
    }

    /**
     * Returns a unified array descriptor of the one given by descriptor, ensuring that, in the case
     * of object arrays, a leading L and a trailing semi-colon (;) are present in the returned
     * descriptor.
     *
     * Note that in the case of primitive arrays no unification takes place.
     *
     * @param descriptor An array descriptor.
     * @return A unified array descriptor.
     */
    private static String unifyArrayDescriptor(String descriptor) {
        String objectArrayPrefix = "L" + PackageConstants.kArrayWrapperSlashPrefix + "$";

        if (descriptor.startsWith(objectArrayPrefix)) {
            // remove trailing semi-colon if there is one.
            int len = descriptor.length() - 1;
            String desc = descriptor.endsWith(";") ? descriptor.substring(0, len) : descriptor;
            String array = desc.substring(objectArrayPrefix.length());
            if (ArrayUtil.isPreRenamePrimitiveArray(array)) {
                return objectArrayPrefix + array + ";";
            }
            String preparedArray = "[" + prepareObjectArrayForUnification(array);
            String unifiedArray = getUnifyingArrayWrapperDescriptor(preparedArray);
            return "L" + unifiedArray + ";";
        } else {
            String starting = (descriptor.startsWith("L")) ? "" : "L";
            String ending = (descriptor.endsWith(";")) ? "" : ";";
            return starting + descriptor + ending;
        }
    }

    /**
     * Converts a multi-dimensional object array descriptor, given by descriptor, whose dimensionality
     * is represented by $ characters into the same descriptor but one in which the dimensionality
     * is represented by [ characters.
     *
     * The purpose of this method is to return a descriptor whose format is of the format expected
     * by the {@link #getUnifyingArrayWrapperDescriptor} method.
     *
     * @param descriptor The multi-dimensional object array descriptor.
     * @return The unifying multi-dimensional object array descriptor.
     */
    private static String prepareObjectArrayForUnification(String descriptor) {
        int numLeadingTokens = 0;
        for (char c : descriptor.toCharArray()) {
            if (c == '$') {
                numLeadingTokens++;
            } else {
                break;
            }
        }
        String transformedTokens = new String(new char[numLeadingTokens]).replace("\0", "[");
        String remainder = descriptor.substring(numLeadingTokens);
        return transformedTokens + remainder;
    }

    public static boolean isIObjectInterfaceFormat(String elementInterfaceSlashName){
        return IOBJECT_INTERFACE_FORMAT.matcher(elementInterfaceSlashName).matches();
    }
    public static boolean isObjectInterfaceFormat(String elementInterfaceSlashName) {
        return OBJECT_INTERFACE_FORMAT.matcher(elementInterfaceSlashName).matches();
    }
    public static boolean isPrimitiveElement(String desc){
        return PRIMITIVES.contains(desc);
    }

    private static void validateArrayDimension(String desc){
        int dim = getPrefixSize(desc, '[');
        if(dim > 3) {
            RejectedClassException.arrayDimensionTooBig(desc);
        }
    }
}
