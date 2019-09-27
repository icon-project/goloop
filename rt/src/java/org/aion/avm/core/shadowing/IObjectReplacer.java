package org.aion.avm.core.shadowing;

import org.aion.avm.core.util.DescriptorParser;
import i.PackageConstants;

class IObjectReplacer {
    private static final String JAVA_LANG_OBJECT = "java/lang/Object";
    private static final String AVM_INTERNAL_IOBJECT = PackageConstants.kInternalSlashPrefix + "IObject";

    private final String shadowJavaLangObject;

    IObjectReplacer(String shadowPackage) {
        this.shadowJavaLangObject = shadowPackage + JAVA_LANG_OBJECT;
    }

    /**
     * Replace java.lang.Object with IObject if necessary.
     *
     * @param type
     * @param allowInterfaceReplacement If true, we will use IObject instead of our shadow Object when replacing java/lang/Object
     * @return
     */
    protected String replaceType(String type, boolean allowInterfaceReplacement) {
        boolean isTypeJavaLangObject = shadowJavaLangObject.equals(type);
        if (allowInterfaceReplacement && isTypeJavaLangObject) {
            return AVM_INTERNAL_IOBJECT;
        } else {
            return type;
        }
    }

    String replaceMethodDescriptor(String methodDescriptor) {
        StringBuilder sb = DescriptorParser.parse(methodDescriptor, new DescriptorParser.Callbacks<>() {
            @Override
            public StringBuilder readObject(int arrayDimensions, String type, StringBuilder userData) {
                populateArray(userData, arrayDimensions);
                userData.append(DescriptorParser.OBJECT_START);
                userData.append(replaceType(type, true));
                userData.append(DescriptorParser.OBJECT_END);
                return userData;
            }

            @Override
            public StringBuilder readBoolean(int arrayDimensions, StringBuilder userData) {
                populateArray(userData, arrayDimensions);
                userData.append(DescriptorParser.BOOLEAN);
                return userData;
            }

            @Override
            public StringBuilder readShort(int arrayDimensions, StringBuilder userData) {
                populateArray(userData, arrayDimensions);
                userData.append(DescriptorParser.SHORT);
                return userData;
            }

            @Override
            public StringBuilder readLong(int arrayDimensions, StringBuilder userData) {
                populateArray(userData, arrayDimensions);
                userData.append(DescriptorParser.LONG);
                return userData;
            }

            @Override
            public StringBuilder readInteger(int arrayDimensions, StringBuilder userData) {
                populateArray(userData, arrayDimensions);
                userData.append(DescriptorParser.INTEGER);
                return userData;
            }

            @Override
            public StringBuilder readFloat(int arrayDimensions, StringBuilder userData) {
                populateArray(userData, arrayDimensions);
                userData.append(DescriptorParser.FLOAT);
                return userData;
            }

            @Override
            public StringBuilder readDouble(int arrayDimensions, StringBuilder userData) {
                populateArray(userData, arrayDimensions);
                userData.append(DescriptorParser.DOUBLE);
                return userData;
            }

            @Override
            public StringBuilder readChar(int arrayDimensions, StringBuilder userData) {
                populateArray(userData, arrayDimensions);
                userData.append(DescriptorParser.CHAR);
                return userData;
            }

            @Override
            public StringBuilder readByte(int arrayDimensions, StringBuilder userData) {
                populateArray(userData, arrayDimensions);
                userData.append(DescriptorParser.BYTE);
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

            private void populateArray(StringBuilder builder, int dimensions) {
                for (int i = 0; i < dimensions; ++i) {
                    builder.append(DescriptorParser.ARRAY);
                }
            }
        }, new StringBuilder());

        return sb.toString();
    }
}