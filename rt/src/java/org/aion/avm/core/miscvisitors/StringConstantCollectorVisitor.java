package org.aion.avm.core.miscvisitors;

import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.stream.Collectors;

import org.aion.avm.core.ClassToolchain;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;


/**
 * Collects string constants from any class it visits, uniquing them in a set.
 * These could be found via LDC loads of String or static String instance variables with constant values.
 */
public class StringConstantCollectorVisitor extends ClassToolchain.ToolChainClassVisitor {
    private final Set<String> stringConstants;

    public StringConstantCollectorVisitor() {
        super(Opcodes.ASM6);
        this.stringConstants = new HashSet<>();
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String descriptor, String signature, String[] exceptions) {
        MethodVisitor visitor = super.visitMethod(access, name, descriptor, signature, exceptions);
        return new MethodVisitor(Opcodes.ASM6, visitor) {
            @Override
            public void visitLdcInsn(Object value) {
                if (value instanceof String) {
                    StringConstantCollectorVisitor.this.stringConstants.add((String)value);
                }
                super.visitLdcInsn(value);
            }
        };
    }

    @Override
    public FieldVisitor visitField(int access, String name, String descriptor, String signature, Object value) {
        // If this is a string, we want to pull it out as a constant.
        if ((Opcodes.ACC_STATIC == (access & Opcodes.ACC_STATIC))
                && (value instanceof String)
        ) {
            this.stringConstants.add((String)value);
        }
        return super.visitField(access, name, descriptor, signature, value);
    }

    /**
     * @return All the string constants found so far, uniqued and sorted alphabetically.
     */
    public List<String> sortedStringConstants() {
        return this.stringConstants.stream().sorted().collect(Collectors.toList());
    }
}
