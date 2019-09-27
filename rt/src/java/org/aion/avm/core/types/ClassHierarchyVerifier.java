package org.aion.avm.core.types;

import java.util.HashSet;
import java.util.Set;
import i.RuntimeAssertionError;

/**
 * A class that verifies that a given class hierarchy is complete and has no inconsistencies.
 *
 * Note that this verifier is looking for logical inconsistencies only. It does not have the classes
 * being referenced in its hands, and therefore is not ensuring that the nodes are faithful
 * representations of the actual types they correspond to!
 */
public final class ClassHierarchyVerifier {

    /**
     * Verifies that the specified hierarchy is a valid hierarchy.
     *
     * This verifier will return an unsuccessful verification result if any of the following faults
     * in the hierarchy are discovered:
     *
     * 1. There exists a ghost node in the hierarchy.
     * 2. There exists an interface that is a child of a non-interface.
     * 3. There exists a node with multiple non-interface parents.
     * 4. There exists a node that is not a descendant of the root node, java.lang.Object
     * 5. There exists a cycle in the graph.
     *
     * If none of these faults are discovered, then the verifier will return a successful result.
     *
     * @param hierarchy The hierarchy to be verified.
     */
    public HierarchyVerificationResult verifyHierarchy(ClassHierarchy hierarchy) {
        if (hierarchy == null) {
            throw new NullPointerException("Cannot verify a null hierarchy.");
        }

        // Note tracking both these sets is superfluous but ensures O(n) node verifications.
        Set<IHierarchyNode> nodesBeingExplored = new HashSet<>();
        Set<IHierarchyNode> nodesFullyExplored = new HashSet<>();

        HierarchyVerificationResult result = verifyNode(hierarchy.getRoot(), nodesBeingExplored, nodesFullyExplored);

        // If we encountered fewer nodes than there are in the hierarchy, then some nodes were not reachable from the root.
        int numUnvisitedNodes = hierarchy.size() - nodesFullyExplored.size();

        if (result.success && numUnvisitedNodes > 0) {
            return HierarchyVerificationResult.foundUnreachableNodes(numUnvisitedNodes);
        } else {
            return result;
        }
    }

    /**
     * Verifies the given node.
     *
     * This node will be added to the set nodesFullyExplored before this method returns.
     *
     * @param node The current node that is about to begin being explored.
     * @param nodesBeingExplored The nodes that are currently being explored and are not finished yet.
     * @param nodesFullyExplored The nodes that have been fully explored already.
     * @return the verification result pertaining to this node.
     */
    private HierarchyVerificationResult verifyNode(IHierarchyNode node, Set<IHierarchyNode> nodesBeingExplored, Set<IHierarchyNode> nodesFullyExplored) {
        RuntimeAssertionError.assertTrue(!nodesBeingExplored.contains(node));
        RuntimeAssertionError.assertTrue(!nodesFullyExplored.contains(node));

        // Add the node to the set of nodes currently being explored.
        nodesBeingExplored.add(node);

        // Verify that the node is not a ghost node. This should never happen.
        if (node.isGhostNode()) {
            return HierarchyVerificationResult.foundGhostNode(node.getDotName());
        }

        // Verify this node does not have multiple non-interface parents.
        int numberOfNonInterfaceParents = 0;
        for (IHierarchyNode parent : node.getParents()) {

            if (parent.isGhostNode()) {
                return HierarchyVerificationResult.foundGhostNode(parent.getDotName());
            }

            if (!parent.getClassInfo().isInterface) {
                numberOfNonInterfaceParents++;
            }
        }

        if (numberOfNonInterfaceParents > 1) {
            return HierarchyVerificationResult.foundMultipleNonInterfaceSuperClasses(node.getDotName());
        }

        // Verify each of its children using a depth-first traversal.
        for (IHierarchyNode child : node.getChildren()) {

            // Verify no interface is a child of a non-interface.
            if ((child.getClassInfo().isInterface) && (!node.getClassInfo().isInterface)) {

                // The only exception to this rule is when parent is java/lang/Object!
                if (!node.getClassInfo().dotName.equals(CommonType.JAVA_LANG_OBJECT.dotName)) {
                    return HierarchyVerificationResult.foundInterfaceWithConcreteSuperClass(child.getDotName());
                }
            }

            // If child is already being explored then it is a child of itself and we have hit a cycle.
            if (nodesBeingExplored.contains(child)) {
                return HierarchyVerificationResult.foundCycle(child.getDotName());
            }

            // Otherwise, if the child has not been encountered before, verify it.
            if (!nodesBeingExplored.contains(child) && !nodesFullyExplored.contains(child)) {

                HierarchyVerificationResult childResult = verifyNode(child, nodesBeingExplored, nodesFullyExplored);

                // If the child verification failed, we propagate this error.
                if (!childResult.success) {
                    return childResult;
                }
            }
        }

        // The node is now fully explored and no errors were encountered, return success.
        nodesBeingExplored.remove(node);
        nodesFullyExplored.add(node);
        return HierarchyVerificationResult.successful();
    }
}
