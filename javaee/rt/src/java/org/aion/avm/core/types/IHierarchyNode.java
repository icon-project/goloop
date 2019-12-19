package org.aion.avm.core.types;

import java.util.Set;

/**
 * A node in a class hierarchy, which represents a class in that hierarchy.
 */
public interface IHierarchyNode {

    /**
     * Returns true only if this node is a ghost node. Otherwise false.
     *
     * @return True if this node is a ghost node.
     */
    public boolean isGhostNode();

    /**
     * Returns the class information object pertaining to this node.
     *
     * @return This node's class information.
     */
    public ClassInformation getClassInfo();

    /**
     * Returns the .-style name of the class this node represents.
     *
     * @return This node's dot-style name.
     */
    public String getDotName();

    /**
     * Adds the specified node as a child of this node.
     *
     * @param node The child node.
     */
    public void addChild(IHierarchyNode node);

    /**
     * Adds the specified node as a parent of this node.
     *
     * @param node The parent node.
     */
    public void addParent(IHierarchyNode node);

    /**
     * Removes the specified parent node from this node's list of parents.
     *
     * @param node The parent node to remove.
     */
    public void removeParent(IHierarchyNode node);

    /**
     * Returns all of this node's parent nodes.
     *
     * @return This node's parent nodes.
     */
    public Set<IHierarchyNode> getParents();

    /**
     * Returns all of this node's child nodes.
     *
     * @return This node's child nodes.
     */
    public Set<IHierarchyNode> getChildren();

}
