package org.aion.avm.core;

import org.aion.avm.core.dappreading.LoadedJar;
import org.aion.avm.core.types.ClassInfo;
import org.aion.avm.core.types.Forest;
import org.aion.avm.core.types.Pair;
import org.objectweb.asm.Attribute;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.ClassVisitor;
import org.objectweb.asm.Opcodes;

import java.io.IOException;
import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

/**
 * A helper which maintain the class inheritance relations.
 *
 * There is one hierarchy forest struct per each DApp; and the forest may include multiple trees.
 * The hierarchy forest is to record all the inheritance relationships of the DApp's classes, but not the ones of the runtime
 * or java.lang.* ones. However, some DApp classes can have a parent class that is one of runtime or java.lang.*. For these
 * classes, it is still needed to record their parents in this hierarchy.
 * Because of that, after the hierarchy of a DApp is built, it should contain one or several trees; each tree has a root
 * node representing a class of the runtime or java.lang.*; and besides the root node, all other node in the tree should
 * represent a DApp class.
 */
public final class ClassHierarchyForest extends Forest<String, ClassInfo> {

    private final LoadedJar loadedJar;

    public Map<String, ClassInfo> toFlatMapWithoutRoots() {
        final var collector = new FlatMapCollector(getNodesCount());
        walkPreOrder(collector);
        return collector.getMap();
    }

    public static ClassHierarchyForest createForestFrom(LoadedJar loadedJar) throws IOException {
        final var forest = new ClassHierarchyForest(loadedJar);
        forest.createForestInternal();
        return forest;
    }

    private ClassHierarchyForest(LoadedJar loadedJar) {
        this.loadedJar = loadedJar;
    }

    private void createForestInternal() throws IOException {
        Map<String, byte[]> classNameToBytes = this.loadedJar.classBytesByQualifiedNames;
        for (Map.Entry<String, byte[]> entry : classNameToBytes.entrySet()) {
            Pair<String, ClassInfo> pair = analyzeClass(entry.getValue());

            if (!pair.value.isInterface()) {
                String parentName = pair.key;
                byte[] parentBytes = classNameToBytes.get(parentName);
                ClassInfo parentInfo = (parentBytes != null) ? analyzeClass(parentBytes).value : new ClassInfo(false, null);

                final var parentNode = new Node<>(parentName, parentInfo);
                final var childNode = new Node<>(entry.getKey(), pair.value);
                add(parentNode, childNode);
            }else{
                // Interface will be added into forest as child of Object
                final var parentNode = new Node<>(Object.class.getName(), new ClassInfo(false, null));
                final var childNode = new Node<>(entry.getKey(), pair.value);
                add(parentNode, childNode);
            }
        }
    }

    /**
     * Analyze the basic info of the class.
     *
     * @param klass the class bytecode
     * @return the declared parent name and the class meta info
     */
    private Pair<String, ClassInfo> analyzeClass(byte[] klass) {
        ClassReader reader = new ClassReader(klass);
        final var codeVisitor = new CodeVisitor();
        reader.accept(codeVisitor, ClassReader.SKIP_FRAMES);
        String parent = codeVisitor.getParentQualifiedName();
        boolean isInterface = codeVisitor.isInterface();
        return Pair.of(parent, new ClassInfo(isInterface, klass));
    }

    private static final class CodeVisitor extends ClassVisitor {
        private String parentQualifiedName;
        private boolean isInterface;

        private CodeVisitor() {
            super(Opcodes.ASM6);
        }

        @Override
        public void visit(int version, int access, String name, String signature, String superName, String[] interfaces) {
            // if the parent is null, DApp deployment will fail due to corrupted JAR data
            parentQualifiedName = toQualifiedName(superName);
            isInterface = Opcodes.ACC_INTERFACE == (access & Opcodes.ACC_INTERFACE);
        }

        @Override
        public void visitSource(String source, String debug) {
            super.visitSource(source, debug);
        }

        @Override
        public void visitAttribute(Attribute attribute) {
            super.visitAttribute(attribute);
        }

        private boolean isInterface() {
            return isInterface;
        }

        private String getParentQualifiedName() {
            return parentQualifiedName;
        }

        private static String toQualifiedName(String internalClassName) {
            return internalClassName.replaceAll("/", ".");
        }
    }

    private static final class FlatMapCollector extends VisitorAdapter<String, ClassInfo> {
        private final Map<String, ClassInfo> map;

        private FlatMapCollector(int size) {
            map = new HashMap<>(size);
        }

        @Override
        public void onVisitNotRootNode(Node<String, ClassInfo> node) {
            map.put(node.getId(), node.getContent());
        }

        private Map<String, ClassInfo> getMap() {
            return Collections.unmodifiableMap(map);
        }
    }
}