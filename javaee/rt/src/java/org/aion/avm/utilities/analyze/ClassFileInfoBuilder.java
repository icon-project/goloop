package org.aion.avm.utilities.analyze;

import org.aion.avm.core.rejection.RejectedClassException;

import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;


public class ClassFileInfoBuilder {
    public static ClassFileInfo getClassFileInfo(byte[] classFile) {
        try {
            return internalClassFileInfo(classFile);
        } catch (Throwable t) {
            return null;
        }
    }

    public static ClassFileInfo getDirectClassFileInfo(byte[] classFile) {
        return internalClassFileInfo(classFile);
    }


    private static ClassFileInfo internalClassFileInfo(byte[] classFile) {
        // https://docs.oracle.com/javase/specs/jvms/se10/html/jvms-4.html

        Map<String, Integer> constantTypeCount = new HashMap<>();
        int totalUtf8Length = 0;
        int codeIndex = -1;

        ByteReader reader = new ByteReader(classFile);
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
                        byte[] raw = reader.readNBytes(length);
                        // We need to read the buffer to advance, but we are also looking for "Code".
                        if ("Code".equals(new String(raw))) {
                            codeIndex = i;
                        }
                        break;
                    case CONSTANT_METHOD_HANDLE:
                        reader.readU1();
                        reader.readU2();
                        break;
                }
            } catch (Exception e) {
                throw new RejectedClassException("Could not find constant pool tag " + tag, e);
            }
            constantTypeCount.put(constantType.name, constantTypeCount.getOrDefault(constantType.name, 0) + 1);
        }
        int constantPoolByteSize = reader.position();

        //access_flags
        reader.readU2();
        //this_class
        reader.readU2();
        //super_class
        reader.readU2();

        //interfaces_count
        int interfaceCount = reader.readU2();
        for (int i = 0; i < interfaceCount; ++i) {
            //interfaces[interfaces_count]
            reader.readU2();
        }

        //fields_count
        int fieldCount = reader.readU2();
        int instanceFieldCount = 0;
        for (int i = 0; i < fieldCount; ++i) {
            //fields[fields_count]
            boolean isInstance = readFieldInfo(reader);
            if (isInstance) {
                instanceFieldCount += 1;
            }
        }
        //methods_count
        int methodCount = reader.readU2();
        List<MethodCode> definedMethods = new ArrayList<>();
        for (int i = 0; i < methodCount; ++i) {
            //methods[methods_count]
            // Note that constant pool entries are 1-indexed, so add 1 to the code index as that is how it will be referenced.
            MethodCode method = readMethodInfo(reader, codeIndex + 1);
            // It is possible that we will get a method without any code section (declared but not defined).
            if (null != method) {
                definedMethods.add(method);
            }
        }
        //attributes_count
        int attributeCount = reader.readU2();
        for (int i = 0; i < attributeCount; ++i) {
            //attributes[attributes_count]
            readAttributeInfo(reader, -1);
        }
        return new ClassFileInfo(classFile.length, constantPoolCount, constantTypeCount, totalUtf8Length, constantPoolByteSize, instanceFieldCount, definedMethods);
    }


    private static boolean readFieldInfo(ByteReader reader) {
        // access_flags.
        int flags = reader.readU2();
        // name_index.
        reader.readU2();
        // descriptor_index.
        reader.readU2();
        // attributes_count.
        int attributeCount = reader.readU2();
        for (int i = 0; i < attributeCount; ++i) {
            readAttributeInfo(reader, -1);
        }
        // STATIC is 0x0008.
        boolean isStatic = (0x0008 == (flags | 0x0008));
        return !isStatic;
    }

    private static MethodCode readMethodInfo(ByteReader reader, int codeIndex) {
        MethodCode implementedMethod = null;;
        // access_flags.
        reader.readU2();
        // name_index.
        reader.readU2();
        // descriptor_index.
        reader.readU2();
        // attributes_count.
        int attributeCount = reader.readU2();
        for (int i = 0; i < attributeCount; ++i) {
            MethodCode code = readAttributeInfo(reader, codeIndex);
            if (null != code) {
                if (null != implementedMethod) {
                    throw new IllegalArgumentException("Multiple code attributes found for the same method");
                }
                implementedMethod = code;
            }
        }
        return implementedMethod;
    }

    private static MethodCode readAttributeInfo(ByteReader reader, int codeAttributeNameIndex) {
        // return null or MethodCode if this is a code attribute.
        MethodCode code = null;
        // attribute_name_index.
        int attributeNameIndex = reader.readU2();
        // attribute_length.
        int attributeLength = reader.readU4();
        int start = reader.position();
        if (codeAttributeNameIndex == attributeNameIndex) {
            // Section 4.7.3 "Code_attribute".
            int max_stack = reader.readU2();
            int max_locals = reader.readU2();
            int code_length = reader.readU4();
            //u1 code[code_length];
            reader.readNBytes(code_length);
            int exception_table_length = reader.readU2();
            for (int i = 0; i < exception_table_length; ++i) {
                //start_pc
                reader.readU2();
                //end_pc
                reader.readU2();
                //handler_pc
                reader.readU2();
                //catch_type
                reader.readU2();
            }
            // attributes_count.
            int attributeCount = reader.readU2();
            for (int i = 0; i < attributeCount; ++i) {
                readAttributeInfo(reader, -1);
            }
            code = new MethodCode(max_stack, max_locals, code_length, exception_table_length);
            /*
            Code_attribute {
                u2 attribute_name_index;
                u4 attribute_length;
                u2 max_stack;
                u2 max_locals;
                u4 code_length;
                u1 code[code_length];
                u2 exception_table_length;
                {   u2 start_pc;
                    u2 end_pc;
                    u2 handler_pc;
                    u2 catch_type;
                } exception_table[exception_table_length];
                u2 attributes_count;
                attribute_info attributes[attributes_count];
            }
            */
        } else {
            // Skip the info.
            reader.readNBytes(attributeLength);
        }
        int end = reader.position();
        if ((end - start) != attributeLength) {
            throw new IllegalArgumentException("Declared attribute length and walked attribute length differ");
        }
        return code;
    }


    public static class ClassFileInfo {
        public final int classFileLength;
        public final int constantPoolEntryCount;
        public final Map<String, Integer> constantTypeCount;
        public final int totalUtf8ByteLength;
        public final int totalConstantPoolByteSize;
        public final int instanceFieldCount;
        public final List<MethodCode> definedMethods;

        public ClassFileInfo(int classFileLength, int constantPoolEntryCount, Map<String, Integer> constantTypeCount, int totalUtf8ByteLength, int totalConstantPoolByteSize, int instanceFieldCount, List<MethodCode> definedMethods) {
            this.classFileLength = classFileLength;
            this.constantPoolEntryCount = constantPoolEntryCount;
            this.constantTypeCount = Collections.unmodifiableMap(constantTypeCount);
            this.totalUtf8ByteLength = totalUtf8ByteLength;
            this.totalConstantPoolByteSize = totalConstantPoolByteSize;
            this.instanceFieldCount = instanceFieldCount;
            this.definedMethods = Collections.unmodifiableList(definedMethods);
        }
    }


    public static class MethodCode {
        public final int maxStack;
        public final int maxLocals;
        public final int codeLength;
        public final int exceptionTableSize;

        public MethodCode(int maxStack, int maxLocals, int codeLength, int exceptionTableSize) {
            this.maxStack = maxStack;
            this.maxLocals = maxLocals;
            this.codeLength = codeLength;
            this.exceptionTableSize = exceptionTableSize;
        }
    }
}
