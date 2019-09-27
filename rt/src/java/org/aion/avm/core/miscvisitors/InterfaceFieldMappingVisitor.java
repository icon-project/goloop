package org.aion.avm.core.miscvisitors;

import org.aion.avm.core.ClassToolchain;
import org.aion.avm.core.types.GeneratedClassConsumer;
import org.objectweb.asm.ClassWriter;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.tree.FieldNode;
import org.objectweb.asm.tree.MethodNode;

import java.util.ArrayList;
import java.util.List;
import java.util.Set;

import static org.objectweb.asm.Opcodes.ACC_INTERFACE;
import static org.objectweb.asm.Opcodes.ACC_PRIVATE;
import static org.objectweb.asm.Opcodes.ALOAD;
import static org.objectweb.asm.Opcodes.INVOKESPECIAL;
import static org.objectweb.asm.Opcodes.RETURN;
import static org.objectweb.asm.Opcodes.V1_6;

/**
 * A visitor which maps the fields of interface into a separate class to solve issue-208.
 * <p>
 * 1) The fields and clinit method of an interface are moved to a generated class;
 * 2) All PUT_FIELD and GET_FIELD are updated accordingly.
 */
public class InterfaceFieldMappingVisitor extends ClassToolchain.ToolChainClassVisitor {

    private static final String SUFFIX = "$FIELDS";

    private GeneratedClassConsumer consumer;
    private Set<String> userInterfaceSlashNames;
    private String javaLangObject;

    private boolean isInterface = false;
    private String name = null;
    private int access = 0;
    private List<FieldNode> fields = new ArrayList<>();
    private MethodNode clinit = null;

    /**
     * Create an InterfaceFieldMappingVisitor instance.
     *
     * @param consumer                A container to collect all the generated classes
     * @param userInterfaceSlashNames The set of user defined classes
     * @param javaLangObjectSlashName The java/lang/Object class name, either pre-rename or post-rename
     */
    public InterfaceFieldMappingVisitor(GeneratedClassConsumer consumer, Set<String> userInterfaceSlashNames, String javaLangObjectSlashName) {
        super(Opcodes.ASM6);
        this.consumer = consumer;
        this.userInterfaceSlashNames = userInterfaceSlashNames;
        this.javaLangObject = javaLangObjectSlashName;
    }

    @Override
    public void visit(int version, int access, String name, String signature, String superName, String[] interfaces) {
        if ((access & ACC_INTERFACE) != 0) {
            this.isInterface = true;
            this.name = name;
            this.access = access;
        }
        super.visit(version, access, name, signature, superName, interfaces);
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String descriptor, String signature, String[] exceptions) {
        MethodVisitor mv;
        if (isInterface && "<clinit>".equals(name)) {
            clinit = new MethodNode(access, name, descriptor, signature, exceptions);
            mv = clinit;
        } else {
            mv = super.visitMethod(access, name, descriptor, signature, exceptions);
        }

        return new MethodVisitor(Opcodes.ASM6, mv) {
            public void visitFieldInsn(int opcode, String owner, String name, String descriptor) {
                if (userInterfaceSlashNames.contains(owner)) {
                    owner += SUFFIX;
                }
                super.visitFieldInsn(opcode, owner, name, descriptor);
            }
        };
    }

    @Override
    public FieldVisitor visitField(int access, String name, String descriptor, String signature, Object value) {
        if (isInterface) {
            FieldNode field = new FieldNode(access, name, descriptor, signature, value);
            fields.add(field);
            return field;
        } else {
            return super.visitField(access, name, descriptor, signature, value);
        }
    }

    @Override
    public void visitEnd() {
        if (isInterface) {
            // NOTE: if the user define such a class, it will get overwritten
            String genName = name + SUFFIX;
            String genSuperName = javaLangObject;
            int genAccess = access & ~ACC_INTERFACE;

            ClassWriter cw = new ClassWriter(0);

            // class declaration
            cw.visit(V1_6, genAccess, genName, null, genSuperName, null);

            // default constructor
            {
                MethodVisitor mv = cw.visitMethod(ACC_PRIVATE, "<init>", "()V", null, null);
                mv.visitCode();
                mv.visitVarInsn(ALOAD, 0); //load the first local variable: this
                mv.visitMethodInsn(INVOKESPECIAL, javaLangObject, "<init>", "()V");
                mv.visitInsn(RETURN);
                mv.visitMaxs(1, 1);
                mv.visitEnd();
            }

            // fields
            for (FieldNode field : fields) {
                field.accept(cw);
            }

            // clinit
            if (clinit != null) {
                clinit.accept(cw);
            }

            consumer.accept(genSuperName, genName, cw.toByteArray());
        }
    }
}
