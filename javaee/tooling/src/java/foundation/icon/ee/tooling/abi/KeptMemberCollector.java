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

package foundation.icon.ee.tooling.abi;

import foundation.icon.ee.struct.Member;
import org.objectweb.asm.AnnotationVisitor;
import org.objectweb.asm.ClassVisitor;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;
import score.annotation.Keep;

import java.util.ArrayList;
import java.util.List;

public class KeptMemberCollector extends ClassVisitor {
    private final List<Member> keptMethods = new ArrayList<>();
    private final List<Member> keptFields = new ArrayList<>();

    public KeptMemberCollector(ClassVisitor cv) {
        super(Opcodes.ASM7, cv);
    }

    @Override
    public MethodVisitor visitMethod(int access, String name,
            String descriptor, String signature, String[] exceptions) {
        var mv = super.visitMethod(access, name, descriptor,
                signature, exceptions);
        return new MethodVisitor(Opcodes.ASM7, mv) {
            @Override
            public AnnotationVisitor visitAnnotation(String aDesc,
                    boolean visible) {
                if (aDesc.equals(Type.getDescriptor(Keep.class))) {
                    keptMethods.add(new Member(name, descriptor));
                    return null;
                }
                return super.visitAnnotation(aDesc, visible);
            }
        };
    }

    @Override
    public FieldVisitor visitField(int access, String name, String descriptor,
            String signature, Object value) {
        var fv = super.visitField(access, name, descriptor,
                signature, value);
        return new FieldVisitor(Opcodes.ASM7, fv) {
            @Override
            public AnnotationVisitor visitAnnotation(String aDesc,
                    boolean visible) {
                if (aDesc.equals(Type.getDescriptor(Keep.class))) {
                    keptFields.add(new Member(name, descriptor));
                    return null;
                }
                return super.visitAnnotation(aDesc, visible);
            }
        };
    }

    public List<Member> getKeptMethods() {
        return keptMethods;
    }

    public List<Member> getKeptFields() {
        return keptFields;
    }
}
