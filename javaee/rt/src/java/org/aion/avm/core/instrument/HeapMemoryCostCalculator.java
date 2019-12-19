package org.aion.avm.core.instrument;

import org.aion.avm.core.NodeEnvironment;
import org.aion.avm.core.types.ClassInfo;
import org.aion.avm.core.types.Forest;
import org.aion.avm.core.types.Forest.Node;
import org.aion.avm.core.util.DescriptorParser;
import org.aion.avm.utilities.Utilities;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.tree.ClassNode;
import org.objectweb.asm.tree.FieldNode;

import java.util.*;

/**
 * Heap memory is allocated at the new object creation. This class provides a map of every class' instance size.
 * Every time an object is created by the "new" instruction, a piece of heap memory of this size is allocated.
 * The accordingly memory usage cost is then charged on the Energy meter.
 *
 * The hashmap stores one instance's heap allocation size of every class.
 *
 * Every instance has a copy of the class fields allocated in the heap.
 * The class fields include the ones declared in this class and its all superclasses.
 *
 * JVM implementation may distinguish between small and large objects and allocate the small ones in "thread local
 * areas (TLAs)" that is reserved from the heap and given to the Java thread (see JRockit JVM spec). Here we don't consider
 * this variance in JVM implementation, aka, the heap allocation size is counted linearly with tha actual object size.
 */
public class HeapMemoryCostCalculator {
    /**
     * Enum - class field size based on the descriptor / type.
     * Size in bits.
     */
    public enum FieldTypeSizeInBytes {
        BYTE        (Byte.BYTES),
        CHAR        (Character.BYTES),
        SHORT       (Short.BYTES),
        INT         (Integer.BYTES),
        LONG        (Long.BYTES),
        FLOAT       (Float.BYTES),
        DOUBLE      (Double.BYTES),
        BOOLEAN     (1),
        OBJECTREF   (8);

        private final int val;

        FieldTypeSizeInBytes(int val) {
            this.val = val;
        }

        public long getVal() {
            return val;
        }
    }

    /**
     * A map that stores the instance size of every class.
     * Key - class name
     * Value - the instance/heap size of the class
     */
    private Map<String, Integer> classHeapSizeMap;
    private Map<String, Integer> rootHeapSizeMap;

    /**
     * Constructor
     */
    public HeapMemoryCostCalculator() {
        classHeapSizeMap = new HashMap<>();
        rootHeapSizeMap = new HashMap<>();
    }

    /**
     * return the map of the class names to their instance sizes
     * @return the hash map that stores the calculated instance sizes of the classes
     */
    public Map<String, Integer> getClassHeapSizeMap() {
        return classHeapSizeMap;
    }

    /**
     * A helper method that calculates the instance size of one class and record it in the "classHeapSizeMap".
     * @param classBytes input class bytecode stream.
     *
     * Note, this method is called from the top to bottom of the class inheritance hierarchy. Such that, it can
     * be assumed that the parent classes' heap size is already in the map.
     */
    private void calcInstanceSizeOfOneClass(byte[] classBytes) {
        if (classHeapSizeMap == null) {
            throw new IllegalStateException("HeapMemoryCostCalculator does not have the classHeapSizeMap.");
        }

        // read in, build the classNode
        ClassNode classNode = new ClassNode();
        ClassReader cr = new ClassReader(classBytes);
        cr.accept(classNode, ClassReader.SKIP_DEBUG);

        // read the class name; check if it is already in the classHeapInfoMap
        if (classHeapSizeMap.containsKey(classNode.name)) {
            return;
        }

        // calculate it if not in the classHeapInfoMap
        int heapSize = 0;
        int superClassSize;

        // get the parent classes, copy the fieldsMap
        if (rootHeapSizeMap.containsKey(classNode.superName)) {
            superClassSize = rootHeapSizeMap.get(classNode.superName);
        } else if (classHeapSizeMap.containsKey(classNode.superName)) {
            superClassSize = classHeapSizeMap.get(classNode.superName);
        } else {
            throw new IllegalStateException("A parent class is not processed by HeapMemoryCostCalculator: " + classNode.superName);
        }

        heapSize += superClassSize;

        // read the declared fields in the current class, add the size of each according to the FieldType
        List<FieldNode> fieldNodes = classNode.fields;
        for (FieldNode fieldNode : fieldNodes) {
            // only calculate heap size for non static fields
            if ((fieldNode.access & Opcodes.ACC_STATIC) == 0) {
                // ArrayType Note:  class object creation only allocates a ref in the heap;
                // and later the bytecode "NEWARRAY / ANEWARRAY" allocates the memory for each element.
                heapSize += DescriptorParser.parse(fieldNode.desc, new DescriptorParser.TypeOnlyCallbacks<Long>() {
                    @Override
                    public Long readObject(int arrayDimensions, String type, Long userData) {
                        return FieldTypeSizeInBytes.OBJECTREF.getVal();
                    }

                    @Override
                    public Long readBoolean(int arrayDimensions, Long userData) {
                        return (0 == arrayDimensions)
                                ? FieldTypeSizeInBytes.BOOLEAN.getVal()
                                : FieldTypeSizeInBytes.OBJECTREF.getVal();
                    }

                    @Override
                    public Long readShort(int arrayDimensions, Long userData) {
                        return (0 == arrayDimensions)
                                ? FieldTypeSizeInBytes.SHORT.getVal()
                                : FieldTypeSizeInBytes.OBJECTREF.getVal();
                    }

                    @Override
                    public Long readLong(int arrayDimensions, Long userData) {
                        return (0 == arrayDimensions)
                                ? FieldTypeSizeInBytes.LONG.getVal()
                                : FieldTypeSizeInBytes.OBJECTREF.getVal();
                    }

                    @Override
                    public Long readInteger(int arrayDimensions, Long userData) {
                        return (0 == arrayDimensions)
                                ? FieldTypeSizeInBytes.INT.getVal()
                                : FieldTypeSizeInBytes.OBJECTREF.getVal();
                    }

                    @Override
                    public Long readFloat(int arrayDimensions, Long userData) {
                        return (0 == arrayDimensions)
                                ? FieldTypeSizeInBytes.FLOAT.getVal()
                                : FieldTypeSizeInBytes.OBJECTREF.getVal();
                    }

                    @Override
                    public Long readDouble(int arrayDimensions, Long userData) {
                        return (0 == arrayDimensions)
                                ? FieldTypeSizeInBytes.DOUBLE.getVal()
                                : FieldTypeSizeInBytes.OBJECTREF.getVal();
                    }

                    @Override
                    public Long readChar(int arrayDimensions, Long userData) {
                        return (0 == arrayDimensions)
                                ? FieldTypeSizeInBytes.CHAR.getVal()
                                : FieldTypeSizeInBytes.OBJECTREF.getVal();
                    }

                    @Override
                    public Long readByte(int arrayDimensions, Long userData) {
                        return (0 == arrayDimensions)
                                ? FieldTypeSizeInBytes.BYTE.getVal()
                                : FieldTypeSizeInBytes.OBJECTREF.getVal();
                    }
                }, null);
            }
        }

        classHeapSizeMap.put(classNode.name, heapSize);
    }

    /**
     * Calculate the instance sizes of classes and record them in the "classHeapInfoMap".
     * This method is called to calculate the heap size of classes that belong to one Dapp, at the deployment time.
     * @param classHierarchy the pre-constructed class hierarchy forest
     */
    public void calcClassesInstanceSize(Forest<String, ClassInfo> classHierarchy) {
        //the pre-constructed map of the runtime and java.lang.* classes to their instance size
        Map<String, Integer> rootClassObjectSizes = NodeEnvironment.singleton.preRenameRuntimeObjectSizeMap;
        // get the root nodes list of the class hierarchy
        Collection<Node<String, ClassInfo>> rootClasses = classHierarchy.getRoots();

        // calculate for each tree in the class hierarchy
        for (Node<String, ClassInfo> rootClass : rootClasses) {
            // 'rootClassObjectSizes' map already has the root class object size.
            // copy rootClass size to classHeapSizeMap
            final String slashName = Utilities.fulllyQualifiedNameToInternalName(rootClass.getId());
            rootHeapSizeMap.put(slashName, rootClassObjectSizes.get(slashName));
        }
        final var visitor = new Forest.Visitor<String, ClassInfo>() {
            @Override
            public void onVisitRoot(Node<String, ClassInfo> root) {
            }

            @Override
            public void onVisitNotRootNode(Node<String, ClassInfo> node) {
                calcInstanceSizeOfOneClass(node.getContent().getBytes());
            }

            @Override
            public void afterAllNodesVisited() {
            }
        };
        classHierarchy.walkPreOrder(visitor);
    }
}
