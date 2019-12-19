package org.aion.avm.core.miscvisitors;

import org.aion.avm.core.ClassToolchain;
import org.aion.avm.core.util.DescriptorParser;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;


/**
 * A visitor which is only used for transforming the various "Helper" class names to the abstract name we inject into the generated
 * code.  This allows the Helper implementation to be decoupled from the instrumentation activity.
 * 
 * WARNING:  Since we don't want to manually update stack frame records, this must be read with ClassReader.SKIP_FRAMES and written
 * with ClassWriter.COMPUTE_FRAMES.
 */
public class ClassRenameVisitor extends ClassToolchain.ToolChainClassVisitor {
    private final String targetSlashName;
    private String originalSlashName;

    public ClassRenameVisitor(String targetSlashName) {
        super(Opcodes.ASM6);
        this.targetSlashName = targetSlashName;
    }

    @Override
    public void visit(int version, int access, String name, String signature, String superName, String[] interfaces) {
        this.originalSlashName = name;
        super.visit(version, access, this.targetSlashName, signature, superName, interfaces);
    }

    @Override
    public FieldVisitor visitField(int access, String name, String descriptor, String signature, Object value) {
        // Note that this field might be an instance of this type or an array of that type.
        return super.visitField(access, name, mapDescriptor(descriptor), null, value);
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String descriptor, String signature, String[] exceptions) {
        MethodVisitor visitor = super.visitMethod(access, name, mapDescriptor(descriptor), null, exceptions);
        
        return new MethodVisitor(Opcodes.ASM6, visitor) {
            @Override
            public void visitTypeInsn(int opcode, String type) {
                super.visitTypeInsn(opcode, mapName(type));
            }
            @Override
            public void visitFieldInsn(int opcode, String owner, String name, String descriptor) {
                super.visitFieldInsn(opcode, mapName(owner), name, mapDescriptor(descriptor));
            }
            @Override
            public void visitMethodInsn(int opcode, String owner, String name, String descriptor, boolean isInterface) {
                super.visitMethodInsn(opcode, mapName(owner), name, mapDescriptor(descriptor), isInterface);
            }
        };
    }


    private String mapDescriptor(String descriptor) {
        StringBuilder builder = DescriptorParser.parse(descriptor, new DescriptorParser.Callbacks<>() {
            @Override
            public StringBuilder readObject(int arrayDimensions, String type, StringBuilder userData) {
                writeArray(arrayDimensions, userData);
                userData.append(DescriptorParser.OBJECT_START);
                String typeName = mapName(type);
                userData.append(typeName);
                userData.append(DescriptorParser.OBJECT_END);
                return userData;
            }
            @Override
            public StringBuilder readBoolean(int arrayDimensions, StringBuilder userData) {
                writeArray(arrayDimensions, userData);
                userData.append(DescriptorParser.BOOLEAN);
                return userData;
            }
            @Override
            public StringBuilder readShort(int arrayDimensions, StringBuilder userData) {
                writeArray(arrayDimensions, userData);
                userData.append(DescriptorParser.SHORT);
                return userData;
            }
            @Override
            public StringBuilder readLong(int arrayDimensions, StringBuilder userData) {
                writeArray(arrayDimensions, userData);
                userData.append(DescriptorParser.LONG);
                return userData;
            }
            @Override
            public StringBuilder readInteger(int arrayDimensions, StringBuilder userData) {
                writeArray(arrayDimensions, userData);
                userData.append(DescriptorParser.INTEGER);
                return userData;
            }
            @Override
            public StringBuilder readFloat(int arrayDimensions, StringBuilder userData) {
                writeArray(arrayDimensions, userData);
                userData.append(DescriptorParser.FLOAT);
                return userData;
            }
            @Override
            public StringBuilder readDouble(int arrayDimensions, StringBuilder userData) {
                writeArray(arrayDimensions, userData);
                userData.append(DescriptorParser.DOUBLE);
                return userData;
            }
            @Override
            public StringBuilder readChar(int arrayDimensions, StringBuilder userData) {
                writeArray(arrayDimensions, userData);
                userData.append(DescriptorParser.CHAR);
                return userData;
            }
            @Override
            public StringBuilder readByte(int arrayDimensions, StringBuilder userData) {
                writeArray(arrayDimensions, userData);
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
            private void writeArray(int arrayDimensions, StringBuilder userData) {
                for (int i = 0; i < arrayDimensions; ++i) {
                    userData.append(DescriptorParser.ARRAY);
                }
            }
        }, new StringBuilder());
        return builder.toString();
    }

    private String mapName(String name) {
        return ClassRenameVisitor.this.originalSlashName.equals(name)
                ? ClassRenameVisitor.this.targetSlashName
                : name;
    }
}
