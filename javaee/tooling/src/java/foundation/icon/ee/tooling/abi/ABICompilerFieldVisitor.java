/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package foundation.icon.ee.tooling.abi;

import org.objectweb.asm.AnnotationVisitor;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;
import score.annotation.EventLog;
import score.annotation.External;
import score.annotation.Optional;
import score.annotation.Payable;

public class ABICompilerFieldVisitor extends FieldVisitor {
    private final int access;
    private final String fieldName;
    private final String fieldDescriptor;

    public ABICompilerFieldVisitor(int access, String fieldName, String fieldDescriptor, FieldVisitor fv) {
        super(Opcodes.ASM7, fv);
        this.access = access;
        this.fieldName = fieldName;
        this.fieldDescriptor = fieldDescriptor;
    }

    @Override
    public AnnotationVisitor visitAnnotation(String descriptor, boolean visible) {
        String[] annotations = new String[] {
                EventLog.class.getName(),
                External.class.getName(),
                Optional.class.getName(),
                Payable.class.getName(),
        };
        for (String annotation : annotations) {
            if (Type.getType(descriptor).getClassName().equals(annotation)) {
                throw new ABICompilerException(
                        "Fields cannot be annotated " + annotation, fieldName);
            }
        }
        return null;
    }

    public String getFieldName() {
        return fieldName;
    }

    public String getFieldDescriptor() {
        return fieldDescriptor;
    }
}
