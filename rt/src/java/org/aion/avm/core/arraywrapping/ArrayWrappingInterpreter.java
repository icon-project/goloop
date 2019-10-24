package org.aion.avm.core.arraywrapping;

import org.aion.avm.core.types.ClassHierarchy;
import org.aion.avm.core.types.CommonType;
import org.aion.avm.core.util.Helpers;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;
import org.objectweb.asm.tree.AbstractInsnNode;
import org.objectweb.asm.tree.analysis.AnalyzerException;
import org.objectweb.asm.tree.analysis.BasicInterpreter;
import org.objectweb.asm.tree.analysis.BasicValue;

/**
 * A bytecode interpreter used for array type inference.
 *
 * See {@link org.aion.avm.core.arraywrapping.ArraysRequiringAnalysisClassVisitor} for its usage.
 */

public class ArrayWrappingInterpreter extends BasicInterpreter{
    private final ClassHierarchy hierarchy;

    ArrayWrappingInterpreter(ClassHierarchy hierarchy) {
      super(Opcodes.ASM6);
      this.hierarchy = hierarchy;
    }

    @Override
    // Override this method to get unmasked type from BasicInterpreter
    public BasicValue newValue(final Type type) {
        if (type == null) {
            return BasicValue.UNINITIALIZED_VALUE;
        }
        switch (type.getSort()) {
            case Type.VOID:
                return null;
            case Type.BOOLEAN:
            case Type.CHAR:
            case Type.BYTE:
            case Type.SHORT:
            case Type.INT:
            case Type.FLOAT:
            case Type.LONG:
            case Type.DOUBLE:
            case Type.ARRAY:
            case Type.OBJECT:
                return new BasicValue(type);
            default:
                throw new AssertionError();
        }
    }

    @Override
    public BasicValue binaryOperation(
            final AbstractInsnNode insn, final BasicValue value1, final BasicValue value2)
            throws AnalyzerException {
        switch (insn.getOpcode()) {
            case IALOAD:
            case BALOAD:
            case CALOAD:
            case SALOAD:
            case IADD:
            case ISUB:
            case IMUL:
            case IDIV:
            case IREM:
            case ISHL:
            case ISHR:
            case IUSHR:
            case IAND:
            case IOR:
            case IXOR:
                return BasicValue.INT_VALUE;
            case FALOAD:
            case FADD:
            case FSUB:
            case FMUL:
            case FDIV:
            case FREM:
                return BasicValue.FLOAT_VALUE;
            case LALOAD:
            case LADD:
            case LSUB:
            case LMUL:
            case LDIV:
            case LREM:
            case LSHL:
            case LSHR:
            case LUSHR:
            case LAND:
            case LOR:
            case LXOR:
                return BasicValue.LONG_VALUE;
            case DALOAD:
            case DADD:
            case DSUB:
            case DMUL:
            case DDIV:
            case DREM:
                return BasicValue.DOUBLE_VALUE;
            case AALOAD:
                return newValue(Type.getType(value1.toString().substring(1)));
            case LCMP:
            case FCMPL:
            case FCMPG:
            case DCMPL:
            case DCMPG:
                return BasicValue.INT_VALUE;
            case IF_ICMPEQ:
            case IF_ICMPNE:
            case IF_ICMPLT:
            case IF_ICMPGE:
            case IF_ICMPGT:
            case IF_ICMPLE:
            case IF_ACMPEQ:
            case IF_ACMPNE:
            case PUTFIELD:
                return null;
            default:
                throw new AssertionError();
        }
    }

    @Override
    public BasicValue merge(BasicValue value1, BasicValue value2) {
        BasicValue b = super.merge(value1, value2);

        if (b.equals(BasicValue.UNINITIALIZED_VALUE)) {

            if (value1.equals(BasicValue.UNINITIALIZED_VALUE) && value2.equals(BasicValue.UNINITIALIZED_VALUE)) {
                return b;
            }

            // Grab the value descriptors.
            String cleanDescriptor1 = value1.toString();
            String cleanDescriptor2 = value2.toString();

            int dimension1 = getArrayDimension(cleanDescriptor1);
            int dimension2 = getArrayDimension(cleanDescriptor2);

            if (dimension1 == 0 || dimension2 == 0) {
                return new BasicValue(Type.getType("[L" + Helpers.fulllyQualifiedNameToInternalName(CommonType.SHADOW_OBJECT.dotName) + ";"));
            }

            if (dimension1 != dimension2) {
                return new BasicValue(Type.getType("[L" + Helpers.fulllyQualifiedNameToInternalName(CommonType.I_OBJECT_ARRAY.dotName) + ";"));
            }

            // Strip the leading array signifiers (the '[' characters)
            cleanDescriptor1 = cleanDescriptor1.substring(dimension1);
            cleanDescriptor2 = cleanDescriptor2.substring(dimension2);

            boolean descriptor1isObject = cleanDescriptor1.startsWith("L");
            boolean descriptor2isObject = cleanDescriptor2.startsWith("L");

            // If we have object arrays, since we know they must differ, we return IObjectArray
            if (!descriptor1isObject && !descriptor2isObject) {
                return new BasicValue(Type.getType("[L" + Helpers.fulllyQualifiedNameToInternalName(CommonType.I_OBJECT_ARRAY.dotName) + ";"));
            }

            // If we have one object array and one non-object array then we return object.
            if ((descriptor1isObject && !descriptor2isObject) || (!descriptor1isObject && descriptor2isObject)) {
                return new BasicValue(Type.getType("[L" + Helpers.fulllyQualifiedNameToInternalName(CommonType.SHADOW_OBJECT.dotName) + ";"));
            }

            // Strip the 'L' character.
            cleanDescriptor1 = cleanDescriptor1.substring(1);
            cleanDescriptor2 = cleanDescriptor2.substring(1);

            // Next we strip the trailing ';' character.
            cleanDescriptor1 = cleanDescriptor1.substring(0,cleanDescriptor1.length() - 1);
            cleanDescriptor2 = cleanDescriptor2.substring(0, cleanDescriptor2.length() - 1);

            // Finally, convert them to dot-style names.
            cleanDescriptor1 = Helpers.internalNameToFulllyQualifiedName(cleanDescriptor1);
            cleanDescriptor2 = Helpers.internalNameToFulllyQualifiedName(cleanDescriptor2);

            // Find the common super class.
            String commonSuper = this.hierarchy.getTightestCommonSuperClass(cleanDescriptor1, cleanDescriptor2);

            // If the super class is ambiguous return IObject, otherwise return it.
            if (commonSuper == null) {
                commonSuper = Helpers.fulllyQualifiedNameToInternalName(CommonType.I_OBJECT.dotName);
            } else {
                // Convert back to slash-style and re-add the characters we stripped.
                commonSuper = Helpers.fulllyQualifiedNameToInternalName(commonSuper);
            }

            // Re-construct the descriptor that we took apart and return it.
            return new BasicValue(Type.getType(getArrayDimensionPrefix(dimension1) + "L" + commonSuper + ";"));
        }

        return b;
    }

    private int getArrayDimension(String descriptor) {
        int length = descriptor.length();

        int dimension = 0;
        for (int i = 0; i < length; i++) {
            if (descriptor.charAt(i) != '[') {
                return dimension;
            }

            dimension++;
        }

        return dimension;
    }

    private String getArrayDimensionPrefix(int dimension) {
        return new String(new char[dimension]).replaceAll("\0", "[");
    }

}
