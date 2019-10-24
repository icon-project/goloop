package org.aion.avm.core.miscvisitors;

import java.util.HashMap;
import java.util.Map;

import org.aion.avm.core.ClassToolchain;
import i.Helper;
import i.PackageConstants;
import i.RuntimeAssertionError;
import org.objectweb.asm.FieldVisitor;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;
import org.objectweb.asm.tree.MethodNode;


/**
 * A dedicated visitor to deal with String and Class constants.
 *
 * A class visitor which replaces the "ConstantValue" static final String constants with a "ldc+putstatic" pair in the &lt;clinit&gt;
 * (much like Class constants).
 * Assumes that all directly loaded String constants ("ldc", not an initialized static field) have already been found with an earlier
 * pass and that the map resolving those constants to field names in the constant class has been provided (along with the constant
 * class name).
 * This should be one of the earliest visitors in the chain since the transformation produces code with no special assumptions,
 * the re-write needs to happen before shadowing and metering, and it depends on nothing other than this being valid bytecode.
 * Note that ASM defines that visitField is called before visitMethod so we can collect the constants we need to re-write before
 * we will see the method we need to modify.
 * If the &lt;clinit&gt; exists, we will prepend this ldc+pustatic pairs.  If it doesn't, we will generate it last.
 * Also responsible for converting "ldc" of String constants to "getstatic" of the corresponding static in the constants class.
 */
public class ConstantVisitor extends ClassToolchain.ToolChainClassVisitor {
    private static final String kClinitName = "<clinit>";
    private static final int kClinitAccess = Opcodes.ACC_STATIC;
    private static final String kClinitDescriptor = "()V";

    private static final String postRenameStringDescriptor = "L" + PackageConstants.kShadowSlashPrefix + "java/lang/String;";

    private static final String wrapClassMethodName = "wrapAsClass";
    private static final String wrapClassMethodDescriptor = "(Ljava/lang/Class;)L" + PackageConstants.kShadowSlashPrefix + "java/lang/Class;";

    private final String constantClassName;
    private final Map<String, String> constantToFieldMap;
    private final Map<String, String> staticFieldNamesToConstantValues;
    private String thisClassName;
    private MethodNode cachedClinit;

    public ConstantVisitor(String constantClassName, Map<String, String> constantToFieldMap) {
        super(Opcodes.ASM6);
        this.constantClassName = constantClassName;
        this.constantToFieldMap = constantToFieldMap;
        
        this.staticFieldNamesToConstantValues = new HashMap<>();
    }

    @Override
    public void visit(int version, int access, String name, String signature, String superName, String[] interfaces) {
        // We just need this to capture the name for the "owner" in the PUTSTATIC calls.
        this.thisClassName = name;
        super.visit(version, access, name, signature, superName, interfaces);
    }

    @Override
    public FieldVisitor visitField(int access, String name, String descriptor, String signature, Object value) {
        // The special case we want to handle is a field with the following properties:
        // -static
        // -non-null value
        // -String
        Object filteredValue = value;
        if ((0 != (access & Opcodes.ACC_STATIC))
                && (null != value)
                && postRenameStringDescriptor.equals(descriptor)
        ) {
            // We need to do something special in this case.
            this.staticFieldNamesToConstantValues.put(name, (String)value);
            filteredValue = null;
        }
        return super.visitField(access, name, descriptor, signature, filteredValue);
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String descriptor, String signature, String[] exceptions) {
        // If this is a clinit, capture it into the MethodNode, for later use.  Otherwise, pass it on as normal.
        MethodVisitor visitor = null;
        if (kClinitName.equals(name)) {
            this.cachedClinit = new MethodNode(access, name, descriptor, signature, exceptions);
            visitor = this.cachedClinit;
        } else {
            visitor = super.visitMethod(access, name, descriptor, signature, exceptions);
        }

        return new MethodVisitor(Opcodes.ASM6, visitor) {
            @Override
            public void visitLdcInsn(Object value) {
                if (value instanceof Type && ((Type) value).getSort() == Type.OBJECT) {
                    // class constants
                    // This covers both Type.ARRAY and Type.OBJECT; since both cases were visited in UserClassMappingVisitor and renamed
                    super.visitLdcInsn(value);
                    super.visitMethodInsn(Opcodes.INVOKESTATIC, Helper.RUNTIME_HELPER_NAME, wrapClassMethodName, wrapClassMethodDescriptor, false);
                } else if (value instanceof String) {
                    // Note that we are moving all strings to the constantClassName, so look up the constant which has this value.
                    String staticFieldForConstant = ConstantVisitor.this.constantToFieldMap.get(value);
                    // (we just created this map in StringConstantCollectionVisitor so nothing can be missing).
                    RuntimeAssertionError.assertTrue(null != staticFieldForConstant);
                    super.visitFieldInsn(Opcodes.GETSTATIC, ConstantVisitor.this.constantClassName, staticFieldForConstant, postRenameStringDescriptor);
                } else {
                    // Type of METHOD and Handle are for classes with version 49 and 51 respectively, and should not happen
                    // https://asm.ow2.io/javadoc/org/objectweb/asm/MethodVisitor.html#visitLdcInsn-java.lang.Object-
                    RuntimeAssertionError.assertTrue(value instanceof Integer || value instanceof Float || value instanceof Long || value instanceof Double);
                    super.visitLdcInsn(value);
                }
            }
        };
    }

    @Override
    public void visitEnd() {
        // Note that visitEnd happens immediately after visitMethod, so we can synthesize the <clinit> here, if it is needed.
        // We want to write the <clinit> if either there was one (which we cached) or we have constant values load into our statics.
        if ((null != this.cachedClinit) || !this.staticFieldNamesToConstantValues.isEmpty()) {
            // Create the actual visitor for the clinit.
            MethodVisitor clinitVisitor = super.visitMethod(kClinitAccess, kClinitName, kClinitDescriptor, null, null);
            
            // Prepend the getstatic+putstatic pairs.
            for (Map.Entry<String, String> elt : this.staticFieldNamesToConstantValues.entrySet()) {
                // Note that we are moving all strings to the constantClassName, so look up the constant which has this value.
                String staticFieldForConstant = this.constantToFieldMap.get(elt.getValue());
                // (we just created this map in StringConstantCollectionVisitor so nothing can be missing).
                RuntimeAssertionError.assertTrue(null != staticFieldForConstant);
                
                // load constant
                clinitVisitor.visitFieldInsn(Opcodes.GETSTATIC, this.constantClassName, staticFieldForConstant, postRenameStringDescriptor);
                
                // set the field
                clinitVisitor.visitFieldInsn(Opcodes.PUTSTATIC, this.thisClassName, elt.getKey(), postRenameStringDescriptor);
            }
            this.staticFieldNamesToConstantValues.clear(); 
            
            // Dump the remaining <clinit> into the visitor or synthesize the end of it, if we don't have a cached one.
            if (null != this.cachedClinit) {
                this.cachedClinit.accept(clinitVisitor);
            } else {
                clinitVisitor.visitInsn(Opcodes.RETURN);
                clinitVisitor.visitMaxs(1, 0);
                clinitVisitor.visitEnd();
            }
        }
        super.visitEnd();
    }
}
