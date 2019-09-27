package org.aion.avm.core.types;

public class HierarchyVerificationResult {
    public final boolean success;
    public final boolean foundGhost;
    public final boolean foundInterfaceWithConcreteSuper;
    public final boolean foundMultipleNonInterfaceSupers;
    public final boolean foundUnreachableNodes;
    public final boolean foundCycle;

    public final String nodeName;
    public final int numberOfUnreachableNodes;

    private HierarchyVerificationResult(boolean success, boolean foundGhost,
        boolean foundInterfaceWithConcreteSuper, boolean foundMultipleNonInterfaceSupers,
        boolean foundUnreachableNodes, boolean foundCycle, String nodeName, int numberOfUnreachableNodes) {

        this.success = success;
        this.foundGhost = foundGhost;
        this.foundInterfaceWithConcreteSuper = foundInterfaceWithConcreteSuper;
        this.foundMultipleNonInterfaceSupers = foundMultipleNonInterfaceSupers;
        this.foundUnreachableNodes = foundUnreachableNodes;
        this.foundCycle = foundCycle;
        this.nodeName = nodeName;
        this.numberOfUnreachableNodes = numberOfUnreachableNodes;
    }

    public static HierarchyVerificationResult successful() {
        return new HierarchyVerificationResult(true, false, false, false, false, false, null, 0);
    }

    public static HierarchyVerificationResult foundGhostNode(String nodeName) {
        return new HierarchyVerificationResult(false,true, false, false, false, false, nodeName, 0);
    }

    public static HierarchyVerificationResult foundInterfaceWithConcreteSuperClass(String nodeName) {
        return new HierarchyVerificationResult(false, false, true, false, false, false, nodeName, 0);
    }

    public static HierarchyVerificationResult foundMultipleNonInterfaceSuperClasses(String nodeName) {
        return new HierarchyVerificationResult(false, false, false, true, false, false, nodeName, 0);
    }

    public static HierarchyVerificationResult foundUnreachableNodes(int numberOfUnreachableNodes) {
        return new HierarchyVerificationResult(false, false, false, false, true, false, null, numberOfUnreachableNodes);
    }

    public static HierarchyVerificationResult foundCycle(String nodeName) {
        return new HierarchyVerificationResult(false, false, false, false, false, true, nodeName, 0);
    }

    public String getError() {
        if (this.success) {
            return "";
        } else if (this.foundGhost) {
            return "found a ghost node '" + this.nodeName + "'";
        } else if (this.foundInterfaceWithConcreteSuper) {
            return "found an interface with a concrete super class '" + this.nodeName + "'";
        } else if (this.foundMultipleNonInterfaceSupers) {
            return "found a class with multiple non-interface super classes '" + this.nodeName + "'";
        } else if (this.foundCycle) {
            return "found a cycle in the graph: " + this.nodeName + " is a child of itself";
        } else {
            return "found " + this.numberOfUnreachableNodes + " nodes that do not descend from the root node";
        }
    }

    @Override
    public String toString() {
        if (this.success) {
            return "HierarchyVerificationResult { successful }";
        } else {
            return "HierarchyVerificationResult { unsuccessful: " + getError() + " }";
        }
    }
}
