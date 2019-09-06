package org.aion.avm.tooling.abi;

import org.objectweb.asm.AnnotationVisitor;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;

public class ABICompilerFieldVisitor extends FieldVisitor {
    private int access;
    private String fieldName;
    private String fieldDescriptor;
    private boolean isInitializable = false;

    public boolean isInitializable() {
        return isInitializable;
    }


    public ABICompilerFieldVisitor(int access, String fieldName, String fieldDescriptor, FieldVisitor fv) {
        super(Opcodes.ASM6, fv);
        this.access = access;
        this.fieldName = fieldName;
        this.fieldDescriptor = fieldDescriptor;
    }

    @Override
    public AnnotationVisitor visitAnnotation(String descriptor, boolean visible) {
        if(Type.getType(descriptor).getClassName().equals(Initializable.class.getName())) {
            if ((this.access & Opcodes.ACC_STATIC) == 0) {
                throw new ABICompilerException("@Initializable fields must be static", fieldName);
            }
            if(!ABIUtils.isAllowedType(Type.getType(fieldDescriptor))) {
                throw new ABICompilerException(
                    Type.getType(fieldDescriptor).getClassName() + " is not an allowed @Initializable type", fieldName);
            }
            isInitializable = true;
            return null;
        } else if (Type.getType(descriptor).getClassName().equals(Fallback.class.getName())) {
            throw new ABICompilerException(
                "Fields cannot be annotated @Fallback", fieldName);
        } else if (Type.getType(descriptor).getClassName().equals(Callable.class.getName())) {
            throw new ABICompilerException(
                "Fields cannot be annotated @Callable", fieldName);
        } else {
            return super.visitAnnotation(descriptor, visible);
        }
    }

    public String getFieldName() {
        return fieldName;
    }

    public String getFieldDescriptor() {
        return fieldDescriptor;
    }
}
