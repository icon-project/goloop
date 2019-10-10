package org.aion.avm.tooling.abi;

import avm.Address;
import org.objectweb.asm.Type;

import java.math.BigInteger;

public class ABIUtils {
    public static boolean isAllowedType(Type type) {
        if(isPrimitiveType(type) || isAllowedObject(type)) {
            return true;
        }
        if (type.getSort() == Type.ARRAY) {
            switch(type.getDimensions()) {
                case 1:
                    return isAllowedType(type.getElementType());
                case 2:
                    // We do not allow 2-dimensional arrays of Strings and Addresses
                    return isPrimitiveType(type.getElementType());
                default:
                    return false;
            }
        }
        return false;
    }

    public static boolean isPrimitiveType(Type type) {
        switch (type.getSort()) {
            case Type.BYTE:
            case Type.BOOLEAN:
            case Type.CHAR:
            case Type.SHORT:
            case Type.INT:
            case Type.FLOAT:
            case Type.LONG:
            case Type.DOUBLE:
                return true;
            default:
                return false;
        }
    }

    public static boolean isAllowedObject(Type type) {
        return type.getClassName().equals(String.class.getName())
                || type.getClassName().equals(Address.class.getName())
                || type.getClassName().equals(BigInteger.class.getName());
    }

    public static String shortenClassName(String s) {
        if(s.contains(".")) {
            return s.substring(s.lastIndexOf('.') + 1);
        } else {
            return s;
        }
    }
}