package org.aion.avm.core.arraywrapping;

import org.aion.avm.RuntimeMethodFeeSchedule;

import java.util.*;
import a.ArrayElement;
import org.aion.avm.core.util.Helpers;
import i.PackageConstants;
import i.RuntimeAssertionError;
import org.objectweb.asm.ClassWriter;
import org.objectweb.asm.Label;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;



public class ArrayWrappingClassGenerator implements Opcodes {
    static private boolean DEBUG = false;

    static private String SHADOW_ARRAY = PackageConstants.kArrayWrapperSlashPrefix + "Array";

    public static byte[] arrayWrappingFactory(String request, ClassLoader loader){

        if (request.startsWith(PackageConstants.kArrayWrapperUnifyingDotPrefix + "_")){
            return genWrapperInterface(request, loader);
        }

        // we only handle class generation request prefixed with a.$
        if (request.startsWith(PackageConstants.kArrayWrapperDotPrefix + "$")){
            return genWrapperClass(request, loader);
        }

        return null;
    }

    private static byte[] genWrapperInterface(String requestInterface, ClassLoader loader) {
        // A wrapper interface backs all classes and interfaces made into arrays (as well as the interface implemented by those).
        // We do this to generalize the solution to a type unification problem (issue-82).
        // The responsibility of this wrapper interface is to represent all the type relationship of the class/interface element type, within the array.
        // This means that all the interfaces of that class/interface and any superclass of that class must be realized here, as an interface wrapper relationship.
        

        if (DEBUG) {
            System.out.println("*********************************");
            System.out.println("requestInterface : " + requestInterface);
        }

        String wrapperInterfaceSlashName = Helpers.fulllyQualifiedNameToInternalName(requestInterface);
        // Get element class and array dim
        String elementInterfaceSlashName = wrapperInterfaceSlashName.substring((PackageConstants.kArrayWrapperUnifyingSlashPrefix).length());
        int dim = ArrayNameMapper.getPrefixSize(elementInterfaceSlashName, '_');
        String elementInterfaceDotName = ArrayNameMapper.getElementInterfaceName(requestInterface);

        Class<?> elementClass = null;
        try {
            elementClass = loader.loadClass(elementInterfaceDotName);
        } catch (ClassNotFoundException e) {
            throw RuntimeAssertionError.unreachable("No valid component : " + elementInterfaceDotName);
        }

        // Handle the element interfaces.
        Class<?>[] superInterfaceClasses =  elementClass.getInterfaces();
        List<String> elementInterfaceWrapperNames = new ArrayList<>();
        for (Class<?> curI : superInterfaceClasses){
            String superInterfaceDotName = ArrayNameMapper.buildArrayDescriptor(dim, typeDescriptorForClass(curI));
            String superInterfaceSlashName = Helpers.fulllyQualifiedNameToInternalName(superInterfaceDotName);
            String superInterfaceWrapperSlashName = ArrayNameMapper.getInterfaceWrapper(superInterfaceSlashName);
            elementInterfaceWrapperNames.add(superInterfaceWrapperSlashName);
        }

        // Handle the element superclass (if not an interface).
        if (!elementClass.isInterface() && !elementClass.getName().equals("java.lang.Object")) {
            Class<?> elementSuperClass = elementClass.getSuperclass();
            String superClassDotName = ArrayNameMapper.buildArrayDescriptor(dim, typeDescriptorForClass(elementSuperClass));
            String slashName = Helpers.fulllyQualifiedNameToInternalName(superClassDotName);
            elementInterfaceWrapperNames.add(ArrayNameMapper.getInterfaceWrapper(slashName));
        }

        // Handle if we have a multi-dimensional IObject interface wrapper to point to its lower dimensional self.
        if (ArrayNameMapper.isIObjectInterfaceFormat(elementInterfaceSlashName)) {
            String slashName = elementInterfaceSlashName.substring(1);  // remove a _ from name
            String fullSlashName = PackageConstants.kArrayWrapperUnifyingSlashPrefix + slashName;
            elementInterfaceWrapperNames.add(ArrayNameMapper.getInterfaceWrapper(fullSlashName));
        }

        // Handle if we have a multi-dimensional Object interface wrapper to point to its lower dimensional self.
        if (ArrayNameMapper.isObjectInterfaceFormat(elementInterfaceSlashName)) {
            String slashName = elementInterfaceSlashName.substring(1);
            String fullSlashName = PackageConstants.kArrayWrapperUnifyingSlashPrefix + slashName;
            String interfaceName = ArrayNameMapper.getInterfaceWrapper(fullSlashName);
            elementInterfaceWrapperNames.add(interfaceName);
        }

        // Handle _IObject unifying type so that it unifies under IObjectArray.
        String IObject1D = PackageConstants.kArrayWrapperUnifyingSlashPrefix + "_L" + PackageConstants.kInternalSlashPrefix + "IObject";
        if (wrapperInterfaceSlashName.equals(IObject1D)) {
            elementInterfaceWrapperNames.add(PackageConstants.kInternalSlashPrefix + "IObjectArray");
        }

        if (DEBUG) {
            System.out.println("Generating interface : " + wrapperInterfaceSlashName);
            for (String s : elementInterfaceWrapperNames) {
                System.out.println("Interfaces : " + s);
            }
            System.out.println("Wrapper Dimension : " + dim);
            System.out.println("*********************************");
        }

        return generateInterfaceBytecode(wrapperInterfaceSlashName, elementInterfaceWrapperNames.toArray(new String[elementInterfaceWrapperNames.size()]));

    }

    private static byte[] generateInterfaceBytecode(String wrapperInterfaceSlashName, String[] superInterfaces) {
        ClassWriter classWriter = new ClassWriter(ClassWriter.COMPUTE_MAXS);
        classWriter.visit(V10, ACC_PUBLIC | ACC_ABSTRACT | ACC_INTERFACE , wrapperInterfaceSlashName, null, "java/lang/Object", superInterfaces);
        classWriter.visitEnd();
        return classWriter.toByteArray();
    }

    private static byte[] genWrapperClass(String requestClass, ClassLoader loader) {
        if (DEBUG) {
            System.out.println("*********************************");
            System.out.println("requestClass : " + requestClass);
        }

        // Class name in bytecode
        String wrapperClassSlashName = Helpers.fulllyQualifiedNameToInternalName(requestClass);

        // Get element class and array dim
        String elementClassSlashName = wrapperClassSlashName.substring(PackageConstants.kArrayWrapperDotPrefix.length());
        int dim = ArrayNameMapper.getPrefixSize(elementClassSlashName, '$');
        String elementClassDotName = ArrayNameMapper.getClassWrapperElementName(requestClass);

        // Default super class is ObjectArray

        byte[] bytecode = null;
        // If element is not primitive type, we need to find its super class
        if (!ArrayNameMapper.isPrimitiveElement(elementClassDotName)) {
            // Element is NOT primitive.
            Class<?> elementClass = null;
            try {
                elementClass = loader.loadClass(elementClassDotName);
            } catch (ClassNotFoundException e) {
                throw RuntimeAssertionError.unreachable("No valid component : " + elementClassDotName);
            }

            // All of these ObjectArray classes are of the same shape:  subclass ObjectArray and implement their own single interface wrapper.
            String interfaceDotName = ArrayNameMapper.buildArrayDescriptor(dim, typeDescriptorForClass(elementClass));
            String interfaceSlashName = Helpers.fulllyQualifiedNameToInternalName(interfaceDotName);
            String interfaceWrapperSlashName = ArrayNameMapper.getInterfaceWrapper(interfaceSlashName);

            String superClassSlashName = PackageConstants.kArrayWrapperSlashPrefix + "ObjectArray";
            bytecode = generateClassBytecode(wrapperClassSlashName, superClassSlashName, dim, new String[] {interfaceWrapperSlashName});

            if (DEBUG) {
                System.out.println("Generating class : " + wrapperClassSlashName);
                System.out.println("Superclass class : " + superClassSlashName);
                System.out.println("Backing Interfaces : " + interfaceWrapperSlashName);
                System.out.println("Wrapper Dimension : " + dim);
                System.out.println("*********************************");
            }
        }else{
            // Element IS primitive
            bytecode = generateClassBytecode(wrapperClassSlashName, PackageConstants.kArrayWrapperSlashPrefix + "ObjectArray", dim, null);
            if (DEBUG) {
                System.out.println("Generating Prim Class : " + wrapperClassSlashName);
                System.out.println("Wrapper Dimension : " + dim);
                System.out.println("*********************************");
            }
        }
        
        // If this is null, an incomplete code path was added.
        RuntimeAssertionError.assertTrue(null != bytecode);
        return bytecode;
    }

    private static byte[] generateClassBytecode(String wrapperClassSlashName, String superClassSlashName, int dimensions, String[] superInterfaceSlashNames){
        ClassWriter classWriter = new ClassWriter(ClassWriter.COMPUTE_MAXS);
        classWriter.visit(V10, ACC_PUBLIC | ACC_SUPER, wrapperClassSlashName, null, superClassSlashName, superInterfaceSlashNames);
        // Static factory for one dimensional array
        // We always generate one D factory for corner case like int[][][] a = new int[10][][];
        genSingleDimensionFactory(classWriter, wrapperClassSlashName, 1);

        if (dimensions > 1) {
            //Static factory for multidimensional array
            genMultiDimensionFactory(classWriter, wrapperClassSlashName, dimensions);
        }

        //Constructor
        genConstructor(classWriter, superClassSlashName);

        //Clone
        genClone(classWriter, wrapperClassSlashName);

        classWriter.visitEnd();

        return classWriter.toByteArray();
    }

    private static void genSingleDimensionFactory(ClassWriter cw, String wrapper, int d){
        String facDesc = ArrayNameMapper.getFactoryDescriptor(wrapper, d);
        MethodVisitor mv = cw.visitMethod(ACC_PUBLIC | ACC_STATIC, "initArray", facDesc, null, null);
        mv.visitCode();
        mv.visitTypeInsn(NEW, wrapper);
        mv.visitInsn(DUP);
        mv.visitVarInsn(ILOAD, 0);
        mv.visitMethodInsn(INVOKESPECIAL, wrapper, "<init>", "(I)V", false);

        // Charge energy
        mv.visitVarInsn(ILOAD, 0);
        mv.visitIntInsn(BIPUSH, (int) ArrayElement.REF.getEnergy());
        mv.visitInsn(IMUL);
        mv.visitMethodInsn(INVOKESTATIC, SHADOW_ARRAY, "chargeEnergy", "(I)V", false);

        mv.visitInsn(ARETURN);
        mv.visitMaxs(3, 1);
        mv.visitEnd();
    }

    private static void genMultiDimensionFactory(ClassWriter cw, String wrapper, int dim) {
        // Code template for $$$MyObject.initArray (3D array of MyObject)
        // Note that for D = n array, n dimension parameter will be passed into initArray
        //
        // public static $$$MyObj initArray(int d0, int d1, int d2){
        //    $$$MyObj ret = new $$$MyObj(d0);
        //    for (int i = 0; i < d0; i++) {
        //        ret.set(i, $$MyObj.initArray(d1, d2));
        //    }
        //    return ret;
        // }

        for (int d = 2; d <= dim; d++) {
            String facDesc = ArrayNameMapper.getFactoryDescriptor(wrapper, d);
            MethodVisitor mv = cw.visitMethod(ACC_PUBLIC | ACC_STATIC, "initArray", facDesc, null, null);
            mv.visitCode();

            // Create new wrapper object with d0 LVT[0]
            mv.visitTypeInsn(NEW, wrapper);
            mv.visitInsn(DUP);
            mv.visitVarInsn(ILOAD, 0);
            mv.visitMethodInsn(INVOKESPECIAL, wrapper, "<init>", "(I)V", false);

            // Charge energy
            mv.visitVarInsn(ILOAD, 0);
            mv.visitIntInsn(BIPUSH, (int) ArrayElement.REF.getEnergy());
            mv.visitInsn(IMUL);
            mv.visitMethodInsn(INVOKESTATIC, SHADOW_ARRAY, "chargeEnergy", "(I)V", false);

            // Wrapper OBJ to return
            // Now LVT[0] ~ LVT[d-1] hold all dimension data, LVT[d] hold wrapper object.
            mv.visitVarInsn(ASTORE, d);

            // Initialize counter to LVT[d+1]
            mv.visitInsn(ICONST_0);
            mv.visitVarInsn(ISTORE, d + 1);

            // For loop head label
            Label forLoopHead = new Label();
            mv.visitLabel(forLoopHead);

            // Stack map frame for for loop
            // Append [wrapper, int] to current frame
            mv.visitFrame(Opcodes.F_APPEND, 2, new Object[]{wrapper, Opcodes.INTEGER}, 0, null);

            // Load counter LVT[d + 1]
            // Load current dimension LVT[0]
            mv.visitVarInsn(ILOAD, d + 1);
            mv.visitVarInsn(ILOAD, 0);

            // compare counter to current dimension
            Label forLoopTail = new Label();
            mv.visitJumpInsn(IF_ICMPGE, forLoopTail);

            // Load wrapper object LVT[d]
            mv.visitVarInsn(ALOAD, d);
            // Load counter LVT[d+1]
            mv.visitVarInsn(ILOAD, d + 1);
            // Load rest of the dimension data LVT[1] ~ LVT[d-1]
            for (int j = 1; j < d; j++) {
                mv.visitVarInsn(ILOAD, j);
            }

            // Call child wrapper factory, child wrapper will pop last d - 1 stack slot as argument.
            // Child wrapper factory descriptor will be constructed here.
            String childWrapper;
            String childFacDesc;
            childWrapper = wrapper.substring((PackageConstants.kArrayWrapperSlashPrefix + "$").length());
            RuntimeAssertionError.assertTrue(childWrapper.startsWith("$"));
            char[] childArray = childWrapper.toCharArray();
            for (int i = 0; childArray[i] == '$'; i++) {
                childArray[i] = '[';
            }
            childWrapper = new String(childArray);
            childWrapper = ArrayNameMapper.getPreciseArrayWrapperDescriptor(childWrapper);
            childFacDesc = ArrayNameMapper.getFactoryDescriptor(childWrapper, d - 1);

            mv.visitMethodInsn(INVOKESTATIC, childWrapper, "initArray", childFacDesc, false);

            // Call set
            mv.visitMethodInsn(INVOKEVIRTUAL, wrapper, "set", "(ILjava/lang/Object;)V", false);

            // Increase counter LVT[d+1]
            mv.visitIincInsn(d + 1, 1);

            mv.visitJumpInsn(GOTO, forLoopHead);
            mv.visitLabel(forLoopTail);

            // Chop off the counter from stack map frame
            mv.visitFrame(Opcodes.F_CHOP, 1, null, 0, null);

            // Load wrapper object LVT[d]
            mv.visitVarInsn(ALOAD, d);

            mv.visitInsn(ARETURN);

            // maxStack is d + 1
            // maxLVT is d + 2
            // We can use class writer to calculate them anyway
            mv.visitMaxs(d + 1, d + 2);
            mv.visitEnd();
        }
    }

    private static void genConstructor(ClassWriter cw, String superName){
        String initName = "<init>";
        
        MethodVisitor methodVisitor = cw.visitMethod(ACC_PUBLIC, initName, "(I)V", null, null);
        methodVisitor.visitCode();
        methodVisitor.visitVarInsn(ALOAD, 0);
        methodVisitor.visitVarInsn(ILOAD, 1);
        methodVisitor.visitMethodInsn(INVOKESPECIAL, superName, initName, "(I)V", false);
        methodVisitor.visitInsn(RETURN);
        methodVisitor.visitMaxs(2, 2);
        methodVisitor.visitEnd();

        methodVisitor = cw.visitMethod(ACC_PUBLIC, initName, "([Ljava/lang/Object;)V", null, null);
        methodVisitor.visitCode();
        methodVisitor.visitVarInsn(ALOAD, 0);
        methodVisitor.visitMethodInsn(INVOKESPECIAL, superName, initName, "()V", false);
        methodVisitor.visitVarInsn(ALOAD, 0);
        methodVisitor.visitVarInsn(ALOAD, 1);
        methodVisitor.visitFieldInsn(PUTFIELD, PackageConstants.kArrayWrapperSlashPrefix + "ObjectArray", "underlying", "[Ljava/lang/Object;");
        methodVisitor.visitInsn(RETURN);
        methodVisitor.visitMaxs(2, 2);
        methodVisitor.visitEnd();

        methodVisitor = cw.visitMethod(ACC_PUBLIC, initName, "()V", null, null);
        methodVisitor.visitCode();
        methodVisitor.visitVarInsn(ALOAD, 0);
        methodVisitor.visitMethodInsn(INVOKESPECIAL, superName, initName, "()V", false);
        methodVisitor.visitInsn(RETURN);
        methodVisitor.visitMaxs(1, 1);
        methodVisitor.visitEnd();

        // Create the deserialization constructor (as seen in AutomaticGraphVisitor).
        String deserializationConstructorDescriptor = "(Ljava/lang/Void;I)V";
        methodVisitor = cw.visitMethod(Opcodes.ACC_PUBLIC, initName, deserializationConstructorDescriptor, null, null);
        methodVisitor.visitCode();
        methodVisitor.visitVarInsn(Opcodes.ALOAD, 0);
        methodVisitor.visitVarInsn(Opcodes.ALOAD, 1);
        methodVisitor.visitVarInsn(Opcodes.ILOAD, 2);
        methodVisitor.visitMethodInsn(Opcodes.INVOKESPECIAL, superName, initName, deserializationConstructorDescriptor, false);
        methodVisitor.visitInsn(Opcodes.RETURN);
        methodVisitor.visitMaxs(4, 4);
        methodVisitor.visitEnd();
    }

    private static void genClone(ClassWriter cw, String wrapper) {
        String cloneMethodName = "avm_clone";
        String cloneMethodDesc = "()Li/IObject;";
        MethodVisitor methodVisitor = cw.visitMethod(ACC_PUBLIC, cloneMethodName, cloneMethodDesc, null, null);

        // energy charge
        methodVisitor.visitLdcInsn(RuntimeMethodFeeSchedule.ObjectArray_avm_clone);
        methodVisitor.visitLdcInsn(RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR);
        methodVisitor.visitVarInsn(ALOAD, 0);
        methodVisitor.visitMethodInsn(INVOKEVIRTUAL, wrapper, "length", "()I", false);
        methodVisitor.visitInsn(IMUL);
        methodVisitor.visitInsn(IADD);
        methodVisitor.visitMethodInsn(INVOKESTATIC, SHADOW_ARRAY, "chargeEnergy", "(I)V", false);

        // lazyLoad
        methodVisitor.visitVarInsn(ALOAD, 0);
        methodVisitor.visitMethodInsn(INVOKEVIRTUAL, wrapper, "lazyLoad", "()V", false);

        methodVisitor.visitCode();
        methodVisitor.visitTypeInsn(NEW, wrapper);
        methodVisitor.visitInsn(DUP);
        methodVisitor.visitVarInsn(ALOAD, 0);
        methodVisitor.visitFieldInsn(GETFIELD, wrapper, "underlying", "[Ljava/lang/Object;");
        methodVisitor.visitVarInsn(ALOAD, 0);
        methodVisitor.visitFieldInsn(GETFIELD, wrapper, "underlying", "[Ljava/lang/Object;");
        methodVisitor.visitInsn(ARRAYLENGTH);
        methodVisitor.visitMethodInsn(INVOKESTATIC, "java/util/Arrays", "copyOf", "([Ljava/lang/Object;I)[Ljava/lang/Object;", false);
        methodVisitor.visitMethodInsn(INVOKESPECIAL, wrapper, "<init>", "([Ljava/lang/Object;)V", false);
        methodVisitor.visitInsn(ARETURN);
        methodVisitor.visitMaxs(4, 1);
        methodVisitor.visitEnd();
    }

    private static String typeDescriptorForClass(Class<?> clazz) {
        return 'L' + clazz.getName() + ';';
    }

}
