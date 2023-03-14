/*
 * Copyright 2020 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package foundation.icon.ee.struct;

import org.objectweb.asm.ClassVisitor;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;

import java.util.ArrayList;
import java.util.List;

public class ClassPropertyMemberInfoCollector extends ClassVisitor {
    private final List<PropertyMember> fields = new ArrayList<>();
    private final List<PropertyMember> getters = new ArrayList<>();
    private final List<PropertyMember> setters = new ArrayList<>();
    private boolean hasAConstructor = false;
    private boolean hasZeroArgPublicConstructor = false;
    private boolean hasCreatableModifier = true;
    private boolean collect;
    private final boolean onlyPublicClass;
    private Type type;
    private Type superType;

    public ClassPropertyMemberInfoCollector(boolean onlyPublicClass) {
        super(Opcodes.ASM7);
        this.onlyPublicClass = onlyPublicClass;
        this.collect = !onlyPublicClass;
    }

    public ClassPropertyMemberInfo getClassPropertyInfo() {
        return new ClassPropertyMemberInfo(type, superType, isCreatable(), fields,
                getters, setters);
    }

    private boolean isCreatable() {
        return (!hasAConstructor || hasZeroArgPublicConstructor)
                && hasCreatableModifier;
    }

    @Override
    public void visit(int version, int access, java.lang.String name,
            java.lang.String signature, java.lang.String superName,
            java.lang.String[] interfaces) {
        if (onlyPublicClass && (access&Opcodes.ACC_PUBLIC) != 0) {
            collect = true;
        }
        if ((access&Opcodes.ACC_ABSTRACT) != 0
                || (access&Opcodes.ACC_INTERFACE) != 0) {
            hasCreatableModifier = false;
        }
        type = Type.getType("L" + name + ";");
        superType = Type.getType("L" + superName + ";");
        super.visit(version, access, name, signature, superName, interfaces);
    }

    @Override
    public FieldVisitor visitField(int access, String name, String descriptor,
            String signature, Object value) {
        if ((access & Opcodes.ACC_PUBLIC) != 0
                && (access & Opcodes.ACC_STATIC) == 0
                && collect) {
            fields.add(new PropertyMember(PropertyMember.FIELD, type, name, descriptor));
        }
        return super.visitField(access, name, descriptor, signature, value);
    }

    private static boolean isSetter(int access, String name, Type type) {
        if ((access & Opcodes.ACC_PUBLIC) == 0
                || (access & Opcodes.ACC_STATIC) != 0) {
            return false;
        }
        if (name.length() < 4 || !name.startsWith("set")) {
            return false;
        }
        if (type.getArgumentTypes().length != 1) {
            return false;
        }
        return type.getReturnType()==Type.VOID_TYPE;
    }

    private static boolean isGetter(int access, String name, Type type) {
        if ((access & Opcodes.ACC_PUBLIC) == 0
                || (access & Opcodes.ACC_STATIC) != 0
                || type.getArgumentTypes().length != 0) {
            return false;
        }

        if ((type.getReturnType()==Type.BOOLEAN_TYPE
                || type.getReturnType()==Type.getType(Boolean.class))
                && name.startsWith("is")
                && name.length() > 2) {
            return true;
        }
        return type.getReturnType()!=Type.VOID_TYPE
                && name.startsWith("get")
                && name.length() > 3;
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String descriptor,
            String signature, String[] exceptions) {
        if (name.equals("<init>")) {
            hasAConstructor = true;
            if (descriptor.equals("()V")
                    && (access&Opcodes.ACC_STATIC) == 0
                    && (access&Opcodes.ACC_PUBLIC) != 0) {
                hasZeroArgPublicConstructor = true;
            }
        }
        var memberType = Type.getType(descriptor);
        if (isSetter(access, name, memberType) && collect) {
            setters.add(new PropertyMember(PropertyMember.SETTER, type, name, memberType));
        } else if (isGetter(access, name, memberType) && collect) {
            getters.add(new PropertyMember(PropertyMember.GETTER, type, name, memberType));
        }
        return super.visitMethod(access, name, descriptor, signature, exceptions);
    }
}
