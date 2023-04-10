package org.aion.avm.core.types;

import java.util.Collections;
import java.util.HashMap;
import java.util.HashSet;
import java.util.LinkedList;
import java.util.Map;
import java.util.Queue;
import java.util.Set;
import i.RuntimeAssertionError;
import org.aion.avm.core.ClassRenamer;
import org.aion.avm.core.ClassRenamer.ArrayType;

/**
 * A class hierarchy is a representation of a class type hierarchy.
 *
 * The hierarchy is a tree structure with java.lang.Object as its root. All child-parent
 * relationships between any nodes in the hierarchy reflect the actual super- and sub-class
 * relationships between these types.
 *
 * The hierarchy should always be constructed with the {@link ClassHierarchyBuilder} class, and
 * once it is constructed it should be verified as being a complete hierarchy using the
 * {@link ClassHierarchyVerifier} class. This second step is very important because it is possible
 * to construct all sorts of hierarchies that do not make any sense.
 *
 * Classes should always be added to the hierarchy using the {@code add()} method whenever the caller
 * knows this class should not already exist. This is to enforce correctness. If this kind of
 * knowledge cannot be obtained, then there is an {@code addIfAbsent()} method.
 *
 * The class hierarchy provides some basic query methods.
 *
 * The class hierarchy provides a method for getting the tightest super class of two classes in the
 * hierarchy: {@code getTightestCommonSuperClass()}.
 *
 * A means of producing a deep copy of the hierarchy is also provided.
 *
 * This hierarchy only accepts post-rename classes!
 */
public final class ClassHierarchy {
    private final DecoratedHierarchyNode root;
    private Map<String, DecoratedHierarchyNode> nameToNodeMapping;
    private Set<String> preRenameUserDefinedClasses;

    /**
     * Constructs a new class hierarchy with the following nodes already in place: java.lang.Object,
     * java.lang.Throwable, IObject, and shadow Object
     */
    public ClassHierarchy() {
        this.nameToNodeMapping = new HashMap<>();
        this.preRenameUserDefinedClasses = null;

        HierarchyNode javaLangObjectNode = HierarchyNode.from(ClassInformation.postRenameInfofrom(CommonType.JAVA_LANG_OBJECT));
        HierarchyNode IObjectNode = HierarchyNode.from(ClassInformation.postRenameInfofrom(CommonType.I_OBJECT));
        HierarchyNode shadowObjectNode = HierarchyNode.from(ClassInformation.postRenameInfofrom(CommonType.SHADOW_OBJECT));
        HierarchyNode javaLangThrowable = HierarchyNode.from(ClassInformation.postRenameInfofrom(CommonType.JAVA_LANG_THROWABLE));

        // Set up the parent-child pointers between the object types.
        connectChildAndParent(IObjectNode, javaLangObjectNode);
        connectChildAndParent(shadowObjectNode, IObjectNode);
        connectChildAndParent(javaLangThrowable, javaLangObjectNode);

        // Set the root and add the object types to the map.
        this.root = DecoratedHierarchyNode.decorate(javaLangObjectNode);
        this.nameToNodeMapping.put(this.root.getDotName(), this.root);
        this.nameToNodeMapping.put(IObjectNode.getDotName(), DecoratedHierarchyNode.decorate(IObjectNode));
        this.nameToNodeMapping.put(shadowObjectNode.getDotName(), DecoratedHierarchyNode.decorate(shadowObjectNode));
        this.nameToNodeMapping.put(javaLangThrowable.getDotName(), DecoratedHierarchyNode.decorate(javaLangThrowable));
    }

    /**
     * Adds the set of pre-rename user-defined classes to the hierarchy. Note that this is the only
     * proper way of adding user-defined classes to the hierarchy so that they are handled correctly
     * in both debug and non-debug modes!
     *
     * This method can only ever be called once, this is because all user-defined classes should be
     * submitted all at once in a single batch, otherwise complications can arise in debug mode.
     *
     * @param classRenamer The class renamer utility.
     * @param preRenameUserDefinedClassInfos The pre-rename user-defined classes.
     */
    public void addPreRenameUserDefinedClasses(ClassRenamer classRenamer, Set<ClassInformation> preRenameUserDefinedClassInfos) {
        RuntimeAssertionError.assertTrue(this.preRenameUserDefinedClasses == null);
        this.preRenameUserDefinedClasses = new HashSet<>();

        // Construct the official set of pre-rename user-defined classes.
        for (ClassInformation classInfo : preRenameUserDefinedClassInfos) {
            this.preRenameUserDefinedClasses.add(classInfo.dotName);
        }

        // Some extra work is required if debuggability is preserved: the user class remains unnamed
        // but any non-user class parents that it subclasses DO get renamed.
        if (classRenamer.preserveDebuggability) {

            Set<ClassInformation> authoritativeUserDefinedClassSet = new HashSet<>();
            for (ClassInformation classInfo : preRenameUserDefinedClassInfos) {

                String superClass = classInfo.superClassDotName;
                String[] superInterfaces = classInfo.getInterfaces();

                // If the superClass is java.lang.Object we set it null (ClassInformation will reparent if necessary)
                boolean superIsObject = (superClass != null) && (superClass.equals(CommonType.JAVA_LANG_OBJECT.dotName));
                superClass = (superIsObject) ? null : superClass;

                if (superClass != null) {
                    superClass = !this.preRenameUserDefinedClasses.contains(superClass)
                        ? classRenamer.toPostRenameOrRejectClass(superClass, ArrayType.NOT_ARRAY)
                        : superClass;
                }

                for (int i = 0; i < superInterfaces.length; i++) {
                    superInterfaces[i] = !this.preRenameUserDefinedClasses.contains(superInterfaces[i])
                        ? classRenamer.toPostRenameOrRejectClass(superInterfaces[i], ArrayType.NOT_ARRAY)
                        : superInterfaces[i];
                }

                if (classInfo.isInterface && (superInterfaces.length == 0)) {
                    authoritativeUserDefinedClassSet.add(ClassInformation.postRenameInfoFor(classInfo.isInterface, classInfo.dotName, superClass, new String[]{ CommonType.I_OBJECT.dotName }));
                } else {
                    authoritativeUserDefinedClassSet.add(ClassInformation.postRenameInfoFor(classInfo.isInterface, classInfo.dotName, superClass, superInterfaces));
                }
            }

            // Finally, now we can add these classes to the hierarchy.
            for (ClassInformation authoritativeUserDefinedClass : authoritativeUserDefinedClassSet) {
                add(authoritativeUserDefinedClass);
            }

        } else {

            for (ClassInformation preRenameClassInfo : preRenameUserDefinedClassInfos) {
                add(ClassInformationRenamer.toPostRenameClassInfo(classRenamer, preRenameClassInfo));
            }

        }
    }

    /**
     * Returns {@code true} only if className is an interface.
     *
     * Assumption is that className is a post-renamed class name.
     */
    public boolean postRenameTypeIsInterface(String className) {
        if (className.equals(CommonType.JAVA_LANG_OBJECT.dotName) || (className.equals(CommonType.JAVA_LANG_THROWABLE.dotName))) {
            return false;
        }

        RuntimeAssertionError.assertTrue(this.nameToNodeMapping.containsKey(className));
        return this.nameToNodeMapping.get(className).getClassInfo().isInterface;
    }

    public String getConcreteSuperClassDotName(String className) {
        RuntimeAssertionError.assertTrue(this.nameToNodeMapping.containsKey(className));
        return this.nameToNodeMapping.get(className).getClassInfo().superClassDotName;
    }

    /**
     * Returns {@code true} only if className is a user-defined class.
     */
    public boolean isPreRenameUserDefinedClass(String className) {
        if (this.preRenameUserDefinedClasses == null) {
            return false;
        }
        return this.preRenameUserDefinedClasses.contains(className);
    }

    /**
     * Returns the set of all user defined classes and interfaces.
     *
     * Note that this does not include java/lang/Object WHEREAS THE OLD WAY DID. (enum, throwable)
     */
    public Set<String> getPreRenameUserDefinedClassesAndInterfaces() {
        return (this.preRenameUserDefinedClasses == null)
            ? Collections.emptySet()
            : new HashSet<>(this.preRenameUserDefinedClasses);
    }

    /**
     * Returns {@code true} only if {@code descendant} is a descendant of {@code superClass}.
     * Returns {@code false} otherwise.
     *
     * @param descendant The descendant class.
     * @param superClass The super class.
     * @return whether or not descendant is a descendant of superClass.
     */
    public boolean isDescendantOfClass(String descendant, String superClass) {
        RuntimeAssertionError.assertTrue(this.nameToNodeMapping.containsKey(descendant));
        RuntimeAssertionError.assertTrue(this.nameToNodeMapping.containsKey(superClass));

        Queue<String> nodesToVisit = new LinkedList<>();
        nodesToVisit.add(superClass);

        while (!nodesToVisit.isEmpty()) {
            DecoratedHierarchyNode nextNode = this.nameToNodeMapping.get(nodesToVisit.poll());

            for (IHierarchyNode child : nextNode.getChildren()) {
                nodesToVisit.add(child.getDotName());
            }

            if (nextNode.getDotName().equals(descendant)) {
                return true;
            }
        }

        return false;
    }

    /**
     * Returns {@code true} only if this hierarchy contains a class with the provided .-style name.
     * False otherwise.
     */
    public boolean contains(String dotName) {
        return this.nameToNodeMapping.containsKey(dotName);
    }

    /**
     * Returns the set of all user defined classes, excluding any user-defined interfaces.
     *
     * Note that this does not include java/lang/Object WHEREAS THE OLD WAY DID. (enum, throwable)
     */
    public Set<String> getPreRenameUserDefinedClassesOnly(ClassRenamer classRenamer) {
        Set<String> classes = new HashSet<>();

        if (this.preRenameUserDefinedClasses == null) {
            return classes;
        }

        for (String className : this.preRenameUserDefinedClasses) {

            // If we are not in debug mode then we have to rename this class since only the renamed version
            // is in the hierarchy.
            String classNameForQuery = classRenamer.toPostRename(className, ArrayType.NOT_ARRAY);

            if (!this.nameToNodeMapping.get(classNameForQuery).getClassInfo().isInterface) {
                classes.add(className);
            }

        }

        return classes;
    }

    /**
     * Returns the class that is the tightest common super class of the two specified classes.
     *
     * A tightest common super class is a class that is a super class of both the specified classes
     * and does not have any child class that is a common super class of the two specified classes.
     *
     * It is possible that multiple classes fulfill this definition. In this case, we consider the
     * tightest common super class ambiguous and return {@code null}.
     *
     * Otherwise, if there is exactly one such class, it is returned.
     *
     * @param class1 The first of the two classes to query.
     * @param class2 The second of the two classes to query.
     * @return The tightest common super class if one exists or else null if ambiguous.
     */
    public String getTightestCommonSuperClass(String class1, String class2) {
        if ((class1 == null) || (class2 == null)) {
            throw new NullPointerException("Cannot get the tightest super class of a null class: " + class1 + ", " + class2);
        }

        if (!this.nameToNodeMapping.containsKey(class1)) {
            throw new IllegalArgumentException("The hierarchy does not contain: " + class1);
        }
        if (!this.nameToNodeMapping.containsKey(class2)) {
            throw new IllegalArgumentException("The hierarchy does not contain: " + class2);
        }

        // Visit the ancestors of the two starting nodes and mark them differently.
        visitAncestorsAndMarkGreen(class1);
        visitAncestorsAndMarkRed(class2);

        // Now, starting at the root, discover all doubly marked leaf nodes.
        Set<ClassInformation> leafNodes = discoverAllDoublyMarkedLeafNodesFromRoot();

        // Clean up the mess we made.
        clearAllMarkings();

        // If these nodes have no super class in common something is very wrong.
        RuntimeAssertionError.assertTrue(!leafNodes.isEmpty());

        if (leafNodes.size() > 1) {
            return null;
        }

        RuntimeAssertionError.assertTrue(leafNodes.size() == 1);
        return leafNodes.iterator().next().dotName;
    }

    HierarchyNode getRoot() {
        return this.root.unwrapRealNode();
    }

    /**
     * Returns the number of classes in this hierarchy.
     *
     * @return The size of the hierarchy.
     */
    public int size() {
        return this.nameToNodeMapping.size();
    }

    /**
     * Note that a deep copy of a hierarchy containing ghost nodes will cause an exception to be
     * thrown.
     *
     * Deep copies should only be made on valid hierarchies that have finished being constructed
     * (and ideally have been verified by {@link ClassHierarchyVerifier}).
     */
    public ClassHierarchy deepCopy() {
        ClassHierarchy deepCopy = new ClassHierarchy();

        // Since ClassInformation is immutable and 'add' creates a tree out of these, we can just
        // re-add each class info to get the deeply copied hierarchy.

        Set<ClassInformation> classInfos = getClassInfosOfAllNodes();

        for (ClassInformation classInfo : classInfos) {

            // Don't ever re-add java/lang/Object to the hierarchy, nor the following for post-renaming: shadow Object, IObject, java/lang/Throwable.
            if (!classInfo.dotName.equals(CommonType.JAVA_LANG_OBJECT.dotName)) {

                if (!classInfo.dotName.equals(CommonType.I_OBJECT.dotName) && !classInfo.dotName.equals(CommonType.SHADOW_OBJECT.dotName) && !classInfo.dotName.equals(CommonType.JAVA_LANG_THROWABLE.dotName)) {
                    deepCopy.add(classInfo);
                }

            }

        }

        // Finally, also copy over the user-defined classes.
        deepCopy.preRenameUserDefinedClasses = (this.preRenameUserDefinedClasses == null)
            ? null
            : new HashSet<>(this.preRenameUserDefinedClasses);

        return deepCopy;
    }

    /**
     * Adds the specified class as a node to the hierarchy, unless it already is present in the
     * hierarchy, then the class is not added.
     *
     * This method will fail if {@code classToAdd == null}.
     *
     * The preferred method to use is {@code add} because it catches us accidentally attempting to
     * add a duplicate class. However, there are certain parts of the system where we can't make
     * assumptions about what is already in the hierarchy yet we want to ensure a list of classes
     * is always in there. In these cases, this method should be used.
     *
     * Note that this method does allow you to construct all sorts of corrupt hierarchies. The
     * {@link ClassHierarchyVerifier} must be run on the hierarchy once it is considered finished in
     * order to verify that the hierarchy is in fact valid.
     *
     * @param classToAdd The class to add to the hierarchy.
     */
    public void addIfAbsent(ClassInformation classToAdd) {
        if (classToAdd == null) {
            throw new NullPointerException("Cannot add a null node to the hierarchy.");
        }

        // Note that a node is considered absent if it is not a real node! A ghost node should be
        // considered absent since its purpose is to act as a placeholder until we encounter it.
        DecoratedHierarchyNode node = this.nameToNodeMapping.get(classToAdd.dotName);

        if ((node == null) || (node.isGhostNode())) {
            add(classToAdd);
        }

    }

    /**
     * Adds the specified class as a node to the hierarchy.
     *
     * This method will fail if:
     *   1. {@code classToAdd == null}
     *   2. The class being added has already been added to the hierarchy previously.
     *
     * But otherwise, this method does allow you to construct all sorts of corrupt hierarchies.
     * The {@link ClassHierarchyVerifier} must be run on the hierarchy once it is considered finished
     * in order to verify that the hierarchy we have constructed is in fact valid.
     *
     * @param classToAdd The class to add to the hierarchy.
     */
    public void add(ClassInformation classToAdd) {
        RuntimeAssertionError.assertTrue(classToAdd != null);
        RuntimeAssertionError.assertTrue(!classToAdd.isPreRenameClassInfo);
        RuntimeAssertionError.assertTrue(!classToAdd.dotName.contains("/"));

        // Add the new node to the hierarchy.
        HierarchyNode newNode = HierarchyNode.from(classToAdd);

        DecoratedHierarchyNode nodeToAddFoundInMap = this.nameToNodeMapping.get(classToAdd.dotName);

        if (nodeToAddFoundInMap == null) {
            // The node we want to add is not already present, so we create it.
            this.nameToNodeMapping.put(newNode.getDotName(), DecoratedHierarchyNode.decorate(newNode));
        } else {
            if (nodeToAddFoundInMap.isGhostNode()) {
                // The node we want to add is already present as a ghost node, so now we can make it
                // a real node since we have its information.
                replaceGhostNodeWithRealNode(nodeToAddFoundInMap.unwrapGhostNode(), newNode);
            } else {
                // The node we want to add already exists - something went wrong.
                throw new IllegalArgumentException("Attempted to re-add a node: " + classToAdd.dotName);
            }
        }

        // Create all the child-parent pointers in the hierarchy for this node.
        String[] superClasses = classToAdd.superClasses();
        for (String superClass : superClasses) {

            // Verify that we are not attempting to add a node directly under java/lang/Object.
            if (superClass.equals(CommonType.JAVA_LANG_OBJECT.dotName)) {
                this.nameToNodeMapping.remove(classToAdd.dotName);
                throw new IllegalArgumentException("Attempted to subclass " + CommonType.JAVA_LANG_OBJECT.dotName + " in a post-rename hierarchy: " + classToAdd.dotName);
            }

            DecoratedHierarchyNode parentNode = this.nameToNodeMapping.get(superClass);

            if (parentNode == null) {
                // The parent isn't in the hierarchy yet, so we create a 'ghost' node as a placeholder for now.
                DecoratedHierarchyNode ghost = DecoratedHierarchyNode.decorate(new HierarchyGhostNode(superClass));
                this.nameToNodeMapping.put(ghost.getDotName(), ghost);

                parentNode = ghost;
            }

            // Add the pointers.
            parentNode.addChild(newNode);
            newNode.addParent(parentNode.unwrap());
        }
    }

    private Set<ClassInformation> getClassInfosOfAllNodes() {
        Set<ClassInformation> classInfos = new HashSet<>();

        for (DecoratedHierarchyNode node : this.nameToNodeMapping.values()) {
            // Ghost nodes don't have associated class info (or they would be a real node).
            RuntimeAssertionError.assertTrue(!node.isGhostNode());
            classInfos.add(node.getClassInfo());
        }

        return classInfos;
    }

    /**
     * Visits all ancestor nodes of the provided starting node and marks them green.
     *
     * ASSUMPTION: startingNode is non-null and exists in the hierarchy.
     */
    private void visitAncestorsAndMarkGreen(String startingNode) {
        visitAncestors(startingNode, true);
    }

    /**
     * Visits all ancestor nodes of the provided starting node and marks them red.
     *
     * ASSUMPTION: startingNode is non-null and exists in the hierarchy.
     */
    private void visitAncestorsAndMarkRed(String startingNode) {
        visitAncestors(startingNode, false);
    }

    /**
     * Visits all descendants of the root node in the hierarchy only if they are doubly marked
     * (that is, marked both green and red).
     *
     * Returns the list of all such doubly-marked nodes that are leaf nodes in this node subset.
     */
    private Set<ClassInformation> discoverAllDoublyMarkedLeafNodesFromRoot() {
        RuntimeAssertionError.assertTrue(this.root.isMarkedGreen() && this.root.isMarkedRed());

        Queue<String> nodesToVisit = new LinkedList<>();
        nodesToVisit.add(this.root.getDotName());

        Set<ClassInformation> leafNodes = new HashSet<>();
        while (!nodesToVisit.isEmpty()) {

            DecoratedHierarchyNode nextNode = this.nameToNodeMapping.get(nodesToVisit.poll());

            // A leaf node in our context is a node that has no doubly-marked children!
            boolean foundChild = false;

            for (IHierarchyNode child : nextNode.getChildren()) {

                // The child pointers are not decorated, so we need to graph the node from the map.
                DecoratedHierarchyNode decoratedChild = this.nameToNodeMapping.get(child.getDotName());

                // Only visit a doubly-marked node.
                if (decoratedChild.isMarkedGreen() && decoratedChild.isMarkedRed()) {
                    foundChild = true;
                    nodesToVisit.add(child.getDotName());
                }

            }

            // If we did not find any children then this is a leaf node.
            if (!foundChild) {
                leafNodes.add(nextNode.getClassInfo());
            }

        }

        return leafNodes;
    }

    /**
     * Clears all of the decorated nodes of any markings applied to them.
     */
    private void clearAllMarkings() {
        for (DecoratedHierarchyNode node : this.nameToNodeMapping.values()) {
            node.clearMarkings();
        }
    }

    /**
     * Replaces the ghost node with the real node.
     *
     * All child-parent pointers that the ghost node had will now be inherited by the real node.
     * Any nodes previously pointing to the ghost node will no longer do so.
     *
     * The ghost node will be entirely removed from the hierarchy.
     *
     * ASSUMPTIONS:
     *    ghostNode and realNode are both non-null
     *    ghostNode is currently in the hierarchy
     */
    private void replaceGhostNodeWithRealNode(HierarchyGhostNode ghostNode, HierarchyNode realNode) {
        RuntimeAssertionError.assertTrue(ghostNode.getDotName().equals(realNode.getDotName()));

        // First, add the real node to the hierarchy now. Since it has the same key as the ghost node,
        // this also removes the ghost node from the mapping.
        this.nameToNodeMapping.put(realNode.getDotName(), DecoratedHierarchyNode.decorate(realNode));

        // Second, inherit all of the child-parent pointer relationships from the ghost node.
        for (IHierarchyNode child : ghostNode.getChildren()) {
            realNode.addChild(child);
            child.addParent(realNode);

            child.removeParent(ghostNode);
        }
    }

    private void visitAncestors(String startingNode, boolean markGreen) {
        Queue<String> nodesToVisit = new LinkedList<>();
        nodesToVisit.add(startingNode);

        while (!nodesToVisit.isEmpty()) {

            String next = nodesToVisit.poll();

            DecoratedHierarchyNode nextNode = this.nameToNodeMapping.get(next);

            if (markGreen) {
                nextNode.markGreen();
            } else {
                nextNode.markRed();
            }

            for (IHierarchyNode child : nextNode.getParents()) {
                nodesToVisit.add(child.getDotName());
            }
        }
    }

    private void connectChildAndParent(IHierarchyNode child, IHierarchyNode parent) {
        child.addParent(parent);
        parent.addChild(child);
    }

    @Override
    public String toString() {
        return "ClassHierarchy { post-rename hierarchy of " + this.nameToNodeMapping.size() + " classes. }";
    }

}
