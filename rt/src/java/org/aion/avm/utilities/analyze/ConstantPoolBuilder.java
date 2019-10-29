package org.aion.avm.utilities.analyze;

import java.util.HashMap;
import java.util.Map;

public class ConstantPoolBuilder {

    public static ClassConstantSizeInfo getConstantPoolInfo(byte[] byteCode) {

        Map<String, Integer> constantTypeCount = new HashMap<>();
        int totalUtf8Length = 0;

        ByteReader reader = new ByteReader(byteCode);
        // magic
        reader.readU4();
        //minorVersion
        reader.readU2();
        // majorVersion
        reader.readU2();

        int constantPoolCount = reader.readU2() - 1;
        for (int i = 0; i < constantPoolCount; i++) {
            int tag = reader.readU1();
            ConstantType constantType = ConstantType.forTag(tag);
            try {
                switch (constantType) {
                    case CONSTANT_CLASS:
                    case CONSTANT_METHOD_TYPE:
                    case CONSTANT_STRING:
                        reader.readU2();
                        break;
                    case CONSTANT_DOUBLE:
                    case CONSTANT_LONG:
                        reader.readU4();
                        reader.readU4();
                        // the next usable item in the pool is located at index n+2
                        i++;
                        break;
                    case CONSTANT_FIELDREF:
                    case CONSTANT_METHODREF:
                    case CONSTANT_NAME_AND_TYPE:
                    case CONSTANT_INVOKE_DYNAMIC:
                    case CONSTANT_INTERFACE_METHODREF:
                        reader.readU2();
                        reader.readU2();
                        break;
                    case CONSTANT_INTEGER:
                    case CONSTANT_FLOAT:
                        reader.readU4();
                        break;
                    case CONSTANT_UTF8:
                        int length = reader.readU2();
                        totalUtf8Length += length;
                        reader.readNBytes(length);
                        break;
                    case CONSTANT_METHOD_HANDLE:
                        reader.readU1();
                        reader.readU2();
                        break;
                }
            } catch (Exception e) {
                throw new AssertionError("Could not find constant pool tag " + tag);
            }
            constantTypeCount.put(constantType.name, constantTypeCount.getOrDefault(constantType.name, 0) + 1);
        }

        return new ClassConstantSizeInfo(byteCode.length, constantTypeCount, totalUtf8Length, reader.position());
    }


    public static class ClassConstantSizeInfo {
        public final int bytecodeLength;
        public final Map<String, Integer> constantTypeCount;
        public final int totalUtf8Length;
        public final int totalConstantPoolSize;

        ClassConstantSizeInfo(int bytecodeLength, Map<String, Integer> constantTypeCount, int totalUtf8Length, int totalConstantPoolSize) {
            this.bytecodeLength = bytecodeLength;
            this.constantTypeCount = constantTypeCount;
            this.totalUtf8Length = totalUtf8Length;
            this.totalConstantPoolSize = totalConstantPoolSize;
        }
    }
}
