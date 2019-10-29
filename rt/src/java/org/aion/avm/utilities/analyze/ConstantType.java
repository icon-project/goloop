package org.aion.avm.utilities.analyze;

import java.util.HashMap;
import java.util.Map;

// Values are from the JVM specification
public enum ConstantType {
    CONSTANT_CLASS(7, "Class"),
    CONSTANT_FIELDREF(9, "Fieldref"),
    CONSTANT_METHODREF(10, "Methodref"),
    CONSTANT_INTERFACE_METHODREF(11, "InterfaceMethodref"),
    CONSTANT_STRING(8, "String"),
    CONSTANT_INTEGER(3, "Integer"),
    CONSTANT_FLOAT(4, "Float"),
    CONSTANT_LONG(5, "Long"),
    CONSTANT_DOUBLE(6, "Double"),
    CONSTANT_NAME_AND_TYPE(12, "NameAndType"),
    CONSTANT_UTF8(1, "Utf8"),
    CONSTANT_METHOD_HANDLE(15, "MethodHandle"),
    CONSTANT_METHOD_TYPE(16, "MethodType"),
    CONSTANT_INVOKE_DYNAMIC(18, "InvokeDynamic");

    public final int tag;
    public final String name;

    ConstantType(int tag, String name) {
        this.tag = tag;
        this.name = name;
    }

    private static final Map<Integer, ConstantType> TAG_ENUM_MAP = new HashMap<>();

    static {
        for (ConstantType constantType : ConstantType.values()) {
            TAG_ENUM_MAP.put(constantType.tag, constantType);
        }
    }

    public static ConstantType forTag(int tag) {
        return TAG_ENUM_MAP.get(tag);
    }
}
