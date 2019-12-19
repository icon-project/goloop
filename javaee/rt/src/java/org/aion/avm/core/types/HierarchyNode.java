package org.aion.avm.core.types;

import java.util.HashSet;
import java.util.Set;
import i.RuntimeAssertionError;

/**
 * A node in a class hierarchy, which represents a class in that hierarchy.
 *
 * Note that, since a class hierarchy can be constructed from classes in any order, the hierarchy
 * may have sufficient information to create a node but insufficient information to create its
 * relatives. In the case of the latter, a ghost node will be added to the hierarchy.
 *
 * Therefore, any children or parents of this node may be a {@link HierarchyNode} or a
 * {@link HierarchyGhostNode}. But they will never be a {@link DecoratedHierarchyNode}.
 *
 * The class info pertaining to this node is immutable. However, the lists of parents and children
 * is not.
 */
public class HierarchyNode implements IHierarchyNode {
    private final ClassInformation classInfo;
    private Set<IHierarchyNode> parents;
    private Set<IHierarchyNode> children;

    private HierarchyNode(ClassInformation classInfo) {
        if (classInfo == null) {
            throw new NullPointerException("Cannot construct node from null class info.");
        }

        this.classInfo = classInfo;
        this.parents = new HashSet<>();
        this.children = new HashSet<>();
    }

    public static HierarchyNode from(ClassInformation classInfo) {
        return new HierarchyNode(classInfo);
    }

    /**
     * Adds node as a child of this node.
     *
     * Throws an exception if node is of type {@link DecoratedHierarchyNode} -- this is to avoid
     * all sorts of confusion that could arise from decorating nodes.
     *
     * @param node The child node.
     */
    @Override
    public void addChild(IHierarchyNode node) {
        if (node == null) {
            throw new NullPointerException("Cannot add pointer to null child node.");
        }

        RuntimeAssertionError.assertTrue(!(node instanceof DecoratedHierarchyNode));
        this.children.add(node);
    }

    /**
     * Adds node as a parent of this node.
     *
     * Throws an exception if node is of type {@link DecoratedHierarchyNode} -- this is to avoid
     * all sorts of confusion that could arise from decorating nodes.
     *
     * @param node The parent node.
     */
    @Override
    public void addParent(IHierarchyNode node) {
        if (node == null) {
            throw new NullPointerException("Cannot add pointer to null parent node.");
        }

        RuntimeAssertionError.assertTrue(!(node instanceof DecoratedHierarchyNode));
        this.parents.add(node);
    }

    @Override
    public void removeParent(IHierarchyNode node) {
        if (node == null) {
            throw new NullPointerException("Cannot remove pointer to null parent node.");
        }

        RuntimeAssertionError.assertTrue(!(node instanceof DecoratedHierarchyNode));
        this.parents.remove(node);
    }

    @Override
    public Set<IHierarchyNode> getParents() {
        return new HashSet<>(this.parents);
    }

    @Override
    public Set<IHierarchyNode> getChildren() {
        return new HashSet<>(this.children);
    }

    @Override
    public String getDotName() {
        return this.classInfo.dotName;
    }

    @Override
    public ClassInformation getClassInfo() {
        return this.classInfo;
    }

    @Override
    public boolean isGhostNode() {
        return false;
    }

    @Override
    public String toString() {
        return "HierarchyNode { " + this.classInfo.rawString() + " }";
    }

    /**
     * Returns true only if other is a {@link HierarchyNode} and its class info is equivalent
     * to this node's class info.
     *
     * Note that this means this node and the other node may have different sets of child and parent
     * pointers and still be equal. This is because pointers get populated when a node joins a
     * hierarchy and equals is decoupled from this dependency since what we typically want to know
     * is whether or not these nodes are talking about the same class as determined by its class
     * info.
     */
    @Override
    public boolean equals(Object other) {
        if (!(other instanceof HierarchyNode)) {
            return false;
        }

        return this.classInfo.equals(((HierarchyNode) other).classInfo);
    }

    @Override
    public int hashCode() {
        return this.classInfo.hashCode();
    }

}
