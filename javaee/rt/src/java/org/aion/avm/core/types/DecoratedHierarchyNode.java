package org.aion.avm.core.types;

import java.util.Set;
import i.RuntimeAssertionError;

/**
 * A decorated node is just a {@link IHierarchyNode} wrapper that allows for a node to be marked different
 * colours.
 *
 * This is used by the {@link ClassHierarchy#getTightestCommonSuperClass(String, String)}
 * algorithm.
 *
 * A decorated node cannot 'decorate' (wrap) another decorated node. You can always assume the
 * wrapped node is not decorated.
 *
 * A decorated node directly exposes the node it wraps and so the immutability of this underlying
 * node is subject to the immutability guarantees of the wrapped node (typically not immutable), and
 * the markings on the decorated node are not immutable either.
 */
public class DecoratedHierarchyNode implements IHierarchyNode {
    private IHierarchyNode node;
    private boolean isGreen;
    private boolean isRed;

    private DecoratedHierarchyNode(IHierarchyNode node) {
        if (node == null) {
            throw new NullPointerException("Cannot decorate a null node.");
        }
        RuntimeAssertionError.assertTrue(!(node instanceof DecoratedHierarchyNode));

        this.node = node;
        this.isGreen = false;
        this.isRed = false;
    }

    public static DecoratedHierarchyNode decorate(IHierarchyNode node) {
        return new DecoratedHierarchyNode(node);
    }

    public IHierarchyNode unwrap() {
        return this.node;
    }

    public HierarchyNode unwrapRealNode() {
        return (HierarchyNode) this.node;
    }

    public HierarchyGhostNode unwrapGhostNode() {
        return (HierarchyGhostNode) this.node;
    }

    public void markGreen() {
        this.isGreen = true;
    }

    public void markRed() {
        this.isRed = true;
    }

    public boolean isMarkedGreen() {
        return this.isGreen;
    }

    public boolean isMarkedRed() {
        return this.isRed;
    }

    public void clearMarkings() {
        this.isGreen = false;
        this.isRed = false;
    }

    @Override
    public boolean isGhostNode() {
        return this.node.isGhostNode();
    }

    @Override
    public ClassInformation getClassInfo() {
        return this.node.getClassInfo();
    }

    @Override
    public String getDotName() {
        return this.node.getDotName();
    }

    @Override
    public void addChild(IHierarchyNode node) {
        RuntimeAssertionError.assertTrue(!(node instanceof DecoratedHierarchyNode));
        this.node.addChild(node);
    }

    @Override
    public void addParent(IHierarchyNode node) {
        RuntimeAssertionError.assertTrue(!(node instanceof DecoratedHierarchyNode));
        this.node.addParent(node);
    }

    @Override
    public void removeParent(IHierarchyNode node) {
        RuntimeAssertionError.assertTrue(!(node instanceof DecoratedHierarchyNode));
        this.node.removeParent(node);
    }

    @Override
    public Set<IHierarchyNode> getParents() {
        return this.node.getParents();
    }

    @Override
    public Set<IHierarchyNode> getChildren() {
        return this.node.getChildren();
    }

    @Override
    public String toString() {
        return "DecoratedHierarchyNode { decorating: " + this.node + " }";
    }
}
