package org.aion.avm.core.types;

import java.util.Collections;
import java.util.HashSet;
import java.util.Set;
import i.RuntimeAssertionError;

/**
 * A ghost node in a class hierarchy, which represents some class in that hierarchy.
 *
 * A ghost node is used in the event that a hierarchy knows a class does exist (ie. some other class
 * is a subclass of it, and so the hierarchy knows about it) but it has inadequate information about
 * that class at the time it is created.
 *
 * A ghost node can therefore be thought of as a placeholder in the hierarchy.
 *
 * A ghost node never has parent nodes but it may have child nodes, because a ghost node is only
 * ever learnt about because one of its children has already been processed. Its children can only
 * ever be of type {@link HierarchyNode}.
 *
 * The class info for this node is immutable. However, the children list is not.
 */
public class HierarchyGhostNode implements IHierarchyNode {
    private final String name;
    private Set<IHierarchyNode> children;

    public HierarchyGhostNode(String name) {
        if (name == null) {
            throw new NullPointerException("Cannot construct ghost node with null name.");
        }

        this.name = name;
        this.children = new HashSet<>();
    }

    /**
     * Adds the node as a child only if it is a {@link HierarchyNode}. Otherwise throws an
     * exception.
     *
     * @param node The child node.
     */
    @Override
    public void addChild(IHierarchyNode node) {
        if (node == null) {
            throw new NullPointerException("Cannot add null child to ghost node: " + this.name);
        }

        RuntimeAssertionError.assertTrue(node instanceof HierarchyNode);
        this.children.add(node);
    }

    /**
     * Unimplemented: if we know the parents of this ghost node then we have enough information to
     * construct a legitimate node.
     *
     * @param node The parent node.
     */
    @Override
    public void addParent(IHierarchyNode node) {
        throw RuntimeAssertionError.unimplemented("[" + this.name + "] A ghost node cannot have parent nodes.");
    }

    /**
     * Unimplemented: see {@code addParent()}.
     *
     * @param node The parent node to remove.
     */
    @Override
    public void removeParent(IHierarchyNode node) {
        throw RuntimeAssertionError.unimplemented("[" + this.name + "] A ghost node cannot have parent nodes.");
    }

    @Override
    public String getDotName() {
        return this.name;
    }

    /**
     * Unimplemented: if we have the class information for the node then we have enough information
     * to construct a legitimate node.
     */
    @Override
    public ClassInformation getClassInfo() {
        throw RuntimeAssertionError.unimplemented("[" + this.name + "] A ghost node has no class information.");
    }

    @Override
    public boolean isGhostNode() {
        return true;
    }

    /**
     * Returns an empty set. See {@code addParent()}.
     */
    @Override
    public Set<IHierarchyNode> getParents() {
        return Collections.emptySet();
    }

    @Override
    public Set<IHierarchyNode> getChildren() {
        return new HashSet<>(this.children);
    }

    @Override
    public String toString() {
        return "HierarchyGhostNode { " + this.name + " }";
    }

}
