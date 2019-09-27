package org.aion.avm.core.types;

import java.util.*;

import static java.lang.String.format;

/**
 * Note! Nodes are double-linked parent has link to child and child has a link to the parent
 *
 * @author Roman Katerinenko
 */
public class Forest<I, C> {
    private final Collection<Node<I, C>> roots = new ArrayList<>();
    private final Map<I, Node<I, C>> nodesIndex = new HashMap<>();

    private Visitor<I, C> currentVisitor;
    private Node<I, C> currentVisitingRoot;

    public Collection<Node<I, C>> getRoots() {
        return Collections.unmodifiableCollection(roots);
    }

    public int getNodesCount() {
        return nodesIndex.size();
    }

    public Node<I, C> getNodeById(I id) {
        Objects.requireNonNull(id);
        return nodesIndex.get(id);
    }

    public Node<I, C> lookupNode(Node<I, C> target) {
        Objects.requireNonNull(target);
        return nodesIndex.get(target.getId());
    }

    public void add(Node<I, C> parent, Node<I, C> child) {
        Objects.requireNonNull(child);
        Objects.requireNonNull(parent);
        if (parent.getId().equals(child.getId())) {
            throw new IllegalArgumentException(format("parent(%s) id must not be equal to child id (%s)", parent.getId(), child.getId()));
        }
        Node<I, C> parentCandidate = lookupExistingFor(parent);
        if (parentCandidate == null) {
            parentCandidate = parent;
            roots.add(parentCandidate);
            nodesIndex.put(parentCandidate.getId(), parentCandidate);
        }
        Node<I, C> childCandidate = lookupExistingFor(child);
        boolean childExisted = true;
        if (childCandidate == null) {
            childExisted = false;
            childCandidate = child;
            nodesIndex.put(childCandidate.getId(), childCandidate);
        }
        if (childExisted && roots.contains(childCandidate)) {
            roots.remove(childCandidate);
        }
        parentCandidate.addChild(childCandidate);
        childCandidate.setParent(parentCandidate);
    }

    // Prune the Forest and only keep the trees of the 'newRoots' roots.
    public void prune(Collection<Node<I, C>> newRoots) {
        Objects.requireNonNull(newRoots);
        final var pruneVisitor = new Visitor<I, C>() {
            @Override
            public void onVisitRoot(Node<I, C> root) {
                nodesIndex.remove(root.getId());
            }

            @Override
            public void onVisitNotRootNode(Node<I, C> node) {
                nodesIndex.remove(node.getId());
            }

            @Override
            public void afterAllNodesVisited() {
            }
        };
        Iterator<Node<I, C>> iterator = roots.iterator();
        while (iterator.hasNext()) {
            Node<I, C> root = iterator.next();
            if (!newRoots.contains(root)) {
                walkOneTreePreOrder(pruneVisitor, root);
                iterator.remove();
            }
        }
    }

    public void walkPreOrder(Visitor<I, C> visitor) {
        Objects.requireNonNull(visitor);
        currentVisitor = visitor;
        for (Node<I, C> root : roots) {
            currentVisitingRoot = root;
            walkPreOrderInternal(root);
        }
        visitor.afterAllNodesVisited();
    }

    private void walkOneTreePreOrder(Visitor<I, C> visitor, Node<I, C> root) {
        Objects.requireNonNull(visitor);
        currentVisitor = visitor;
        currentVisitingRoot = root;
        walkPreOrderInternal(root);
        visitor.afterAllNodesVisited();
    }

    private void walkPreOrderInternal(Node<I, C> node) {
        if (node == currentVisitingRoot) {
            currentVisitor.onVisitRoot(node);
        } else {
            currentVisitor.onVisitNotRootNode(node);
        }
        for (Node<I, C> child : node.getChildren()) {
            walkPreOrderInternal(child);
        }
    }

    private Node<I, C> lookupExistingFor(Node<I, C> node) {
        return nodesIndex.get(node.getId());
    }

    public static class Node<I, C> {
        private final Collection<Node<I, C>> childs = new LinkedHashSet<>();

        private I id;
        private C content;
        private Node<I, C> parent;

        public Node(I id, C content) {
            Objects.requireNonNull(id);
            this.id = id;
            this.content = content;
        }

        public Node<I, C> getParent() {
            return parent;
        }

        public void setParent(Node<I, C> parent) {
            this.parent = parent;
        }

        public void addChild(Node<I, C> child) {
            Objects.requireNonNull(child);
            childs.add(child);
        }

        public Collection<Node<I, C>> getChildren() {
            return Collections.unmodifiableCollection(childs);
        }

        public I getId() {
            return id;
        }

        public C getContent() {
            return content;
        }

        public void setContent(C c) {
            this.content = c;
        }

        @Override
        public int hashCode() {
            return id.hashCode();
        }

        @Override
        public boolean equals(Object that) {
            return (that instanceof Node) && id.equals(((Node) that).id);
        }
    }

    public interface Visitor<I, C> {
        void onVisitRoot(Node<I, C> root);

        void onVisitNotRootNode(Node<I, C> node);

        void afterAllNodesVisited();
    }

    public static class VisitorAdapter<I, C> implements Visitor<I, C> {
        @Override
        public void onVisitRoot(Node<I, C> root) {
        }

        @Override
        public void onVisitNotRootNode(Node<I, C> node) {
        }

        @Override
        public void afterAllNodesVisited() {
        }
    }
}