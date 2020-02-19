package org.aion.avm.tooling.abi;

import score.Address;
import org.objectweb.asm.Type;

import java.math.BigInteger;
import java.util.List;
import java.util.Map;

public class ABIUtils {
    private static final List<String> paramTypes = List.of(
            "Z",
            "C",
            "B",
            "S",
            "I",
            "J",
            Type.getDescriptor(BigInteger.class),
            Type.getDescriptor(String.class),
            Type.getDescriptor(Address.class),
            "[B"
    );

    public static boolean isAllowedParamType(Type type) {
        return paramTypes.contains(type.getDescriptor());
    }

    private static final List<String> returnTypes = List.of(
            "Z",
            "C",
            "B",
            "S",
            "I",
            "J",
            Type.getDescriptor(BigInteger.class),
            Type.getDescriptor(String.class),
            Type.getDescriptor(Address.class),
            "[B",
            Type.getDescriptor(List.class),
            Type.getDescriptor(Map.class),
            "V"
    );

    public static boolean isAllowedReturnType(Type type) {
        return returnTypes.contains(type.getDescriptor());
    }

    public static String shortenClassName(String s) {
        if(s.contains(".")) {
            return s.substring(s.lastIndexOf('.') + 1);
        } else {
            return s;
        }
    }
}