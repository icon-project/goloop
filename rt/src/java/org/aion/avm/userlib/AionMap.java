package org.aion.avm.userlib;

import java.util.*;

/**
 * B+ Tree based implementation of the {@code Map} interface.
 *
 * <p>This implementation provides all of the optional map operations, and permits {@code null} values.
 * {@code null} key will be rejected for this map.
 *
 * <p>This implementation provides log-time performance for the basic operations ({@code get} and {@code put}),
 * assuming the hashCode() function disperses the elements properly.
 * Iteration over collection views requires linear time.
 * If AionMap has more than {@code order} number of entries, it is at least half full.
 *
 * <p>Instead of using key as router in internal node, AionMap use the hash code of the key as router to make its
 * energy cost cheap on Aion blockchain.
 *
 * @param <K> The type of keys within the map.
 * @param <V> The type of values within the map.
 */

public class AionMap<K, V> implements Map<K, V> {

    /**
     * The default order of Btree map
     */
    static final int DEFAULT_ORDER = 4;

    /**
     * The current size of the map
     */
    private int size;

    /**
     * The order of the Btree map
     * For a BTree, on each node
     * There will be at least (order) number of entries.
     * There will be at most (2 * order) number of entries.
     * Order can not be changed after map is created
     */
    private final int order;

    /**
     * The root of the BTree map
     * It will be a leaf node when size of the map is less than (2 * order)
     * It will be a internal node when size of the map is more than (2 * order)
     */
    private BNode root;

    /**
     * Constructs an empty {@code AionMap} with the default order (4).
     */
    public AionMap(){
        this.order = DEFAULT_ORDER;
        this.size = 0;
        this.root = new BLeafNode();
    }

    /**
     * Constructs an empty {@code AionMap} with initial order.
     *
     * @param order The order of the {@code AionMap}
     */
    public AionMap(int order){
        this.order = order;
        this.size = 0;
        this.root = new BLeafNode();
    }

    /**
     * Returns the number of key-value mappings in this map.
     *
     * @return the number of key-value mappings in this map
     */
    @Override
    public int size() {
        return size;
    }


    /**
     * Returns {@code true} if this map contains no key-value mappings.
     *
     * @return {@code true} if this map contains no key-value mappings
     */
    @Override
    public boolean isEmpty() {
        return this.size == 0;
    }

    /**
     * Returns {@code true} if this map contains a mapping for the
     * specified key.
     *
     * @param   key   The key whose presence in this map is to be tested
     * @return {@code true} if this map contains a mapping for the specified
     * key.
     */
    @SuppressWarnings("unchecked")
    @Override
    public boolean containsKey(Object key) {
        keyNullCheck((K) key);
        AionMapEntry entry = this.searchForLeaf((K)key).searchForEntry((K)key);
        return (null != entry);
    }

    /**
     * Returns {@code true} if this map maps one or more keys to the
     * specified value.
     *
     * @param value value whose presence in this map is to be tested
     * @return {@code true} if this map maps one or more keys to the
     *         specified value
     */
    @Override
    public boolean containsValue(Object value) {
        for (V v : this.values()){
            if (v.equals(value)){
                return true;
            }
        }
        return false;
    }

    /**
     * Returns the value to which the specified key is mapped,
     * or {@code null} if this map contains no mapping for the key.
     *
     * <p>A return value of {@code null} does not <i>necessarily</i>
     * indicate that the map contains no mapping for the key; it's also
     * possible that the map explicitly maps the key to {@code null}.
     * The {@link #containsKey containsKey} operation may be used to
     * distinguish these two cases.
     *
     */
    @SuppressWarnings("unchecked")
    @Override
    public V get(Object key) {
        keyNullCheck((K) key);
        AionMapEntry entry = this.searchForLeaf((K)key).searchForEntry((K)key);
        return (null == entry) ? null : (V) entry.value;
    }


    /**
     * Associates the specified value with the specified key in this map.
     * If the map previously contained a mapping for the key, the old
     * value is replaced.
     *
     * @param key key with which the specified value is to be associated
     * @param value value to be associated with the specified key
     * @return the previous value associated with {@code key}, or
     *         {@code null} if there was no mapping for {@code key}.
     *         (A {@code null} return can also indicate that the map
     *         previously associated {@code null} with {@code key}.)
     */
    @Override
    public V put(K key, V value) {
        keyNullCheck(key);
        V ret = null;

        // Search for the leaf and slot
        BLeafNode leaf = this.searchForLeaf(key);
        int slotIdx = leaf.searchForEntrySlot(key);

        if (-1 == slotIdx){
            // If slot is not present, insert new entry into the leaf node
            bInsert(key, value);
            size++;
        }else {
            // If slot is present, search for entry
            AionMapEntry cur = leaf.searchForEntryInSlot(key, slotIdx);

            if (null != cur){
                // if entry is present, replace the value, return the old value
                ret = (V) cur.getValue();
                cur.setValue(value);
            }else{
                // If entry is not present, add new entry as new head of the slot
                AionMapEntry newEntry = new AionMapEntry(key, value);
                newEntry.next = leaf.entries[slotIdx];
                leaf.entries[slotIdx] = newEntry;
                size++;
            }
        }
        return ret;
    }

    /**
     * Removes the mapping for the specified key from this map if present.
     *
     * @param  key key whose mapping is to be removed from the map
     * @return the previous value associated with {@code key}, or
     *         {@code null} if there was no mapping for {@code key}.
     *         (A {@code null} return can also indicate that the map
     *         previously associated {@code null} with {@code key}.)
     */
    @SuppressWarnings("unchecked")
    @Override
    public V remove(Object key) {
        keyNullCheck((K) key);
        V ret = (V) root.delete((K) key);
        if (null != ret){size--;}
        return ret;
    }

    /**
     * Copies all of the mappings from the specified map to this map.
     * These mappings will replace any mappings that this map had for
     * any of the keys currently in the specified map.
     *
     * @param m mappings to be stored in this map
     * @throws NullPointerException if the specified map is null
     */
    @SuppressWarnings("unchecked")
    @Override
    public void putAll(Map<? extends K, ? extends V> m) {
        for (Entry<?, ?> e : m.entrySet()){
            this.put((K) e.getKey(), (V) e.getValue());
        }
    }


    /**
     * Removes all of the mappings from this map.
     * The map will be empty after this call returns.
     * The order of this map will remain the same.
     */
    @Override
    public void clear() {
        this.size = 0;
        this.root = new BLeafNode();
    }

    /**
     * Returns a {@link Set} view of the keys contained in this map.
     * The key set is based on snapshot of the map.
     * Modifications of the map after key set generation will not be reflected back to existing key set, and vice-versa.
     *
     * @return a set view of the keys contained in this map
     */
    @Override
    public Set<K> keySet() {
        Set<K> ret = new AionMapKeySet();
        return ret;
    }

    /**
     * Returns a {@link Collection} view of the values contained in this map.
     * The values is based on snapshot of the map.
     * Modifications of the map after values generation will not be reflected back to existing values, and vice-versa.
     *
     * @return a view of the values contained in this map
     */
    @Override
    public Collection<V> values() {
        return new AionMapValues();
    }

    /**
     * Returns a {@link Set} view of the mappings contained in this map.
     * The entry set is based on a snapshot of the map.
     * Modifications of the map after values generation will not be reflected back to existing entry set, and vice-versa.
     *
     * @return a set view of the mappings contained in this map
     */
    @Override
    public Set<Entry<K, V>> entrySet() {
        return new AionMapEntrySet();
    }

    public V getOrDefault(Object key, V defaultValue) {
        return containsKey(key) ? get(key) : defaultValue;
    }

    /**
     * Abstract representation of a node of the BTree
     * It can be a {@link BInternalNode} or {@link BLeafNode}
     */
    public abstract class BNode {
        BNode parent;

        BNode next;
        BNode pre;

        int nodeSize;

        abstract void insertNonFull(K key, V value);

        abstract V delete(K key);

        void rebalance(){
            if (this.isUnderflow() && root != this){
                // Check if we can borrow from left sibling
                if (null != this.pre){
                    // Check if we can borrow from left sibling
                    if (!this.pre.isMinimal()){
                        //Borrow
                        borrowFromLeft();
                    }else{
                        mergeToLeft();
                    }
                }else{
                    // Check if we can borrow from right sibling
                    if (!this.next.isMinimal()){
                        //Borrow
                        borrowFromRight();
                    }else{
                        mergeToRight();
                    }
                }
            }else if (root == this && needCollapse()){
                collapseRoot();
            }
        }

        boolean isUnderflow(){
            return nodeSize < order;
        }

        boolean isMinimal(){
            return nodeSize == order;
        }

        boolean needCollapse(){
            return nodeSize == 1;
        }

        BNode getAnchor(BNode target){
            // Find the common ancestor of this and target
            BNode thisAnchor = this.parent;
            BNode targetAnchor = target.parent;

            while (thisAnchor != targetAnchor){
                thisAnchor = thisAnchor.parent;
                targetAnchor = targetAnchor.parent;
            }

            return thisAnchor;
        }

        abstract void borrowFromLeft();

        abstract void mergeToLeft();

        abstract void borrowFromRight();

        abstract void mergeToRight();

        abstract void collapseRoot();

    }

    /**
     * Internal node of the BTree
     */
    public final class BInternalNode extends BNode {
        // Routers array for navigation
        private int[] routers;

        // The children of an internal node.
        // Children are either all internal nodes or all leaf nodes.
        BNode[] children;

        @SuppressWarnings("unchecked")
        BInternalNode(){
            this.routers = new int[2 * order - 1];
            this.children = new AionMap.BNode[2 * order];
        }

        @Override
        void insertNonFull(K key, V value) {
            int i = this.nodeSize - 1;
            while (i > 0 && key.hashCode() < this.routers[i - 1]){
                i--;
            }

            if (this.children[i].nodeSize == (2 * order)){
                bSplitChild(this, i);
                if (key.hashCode() >= this.routers[i]){
                    i++;
                }
            }
            this.children[i].insertNonFull(key, value);
        }

        @Override
        V delete(K key) {
            V ret = null;
            int i = this.nodeSize - 1;
            while (i > 0 && key.hashCode() < this.routers[i - 1]){
                i--;
            }

            ret = this.children[i].delete(key);

            // Delete succeed, check router and rebalance
            if (ret != null){
                if (this.recalibrate()){
                    this.rebalance();
                }
            }
            return ret;
        }

        private boolean recalibrate(){
            boolean ret = false;

            // If the first child is removed, shift both children and routers left by 1
            if (0 == this.children[0].nodeSize){
                System.arraycopy(this.children, 1, this.children, 0, this.nodeSize);
                if (this.nodeSize > 2) {
                    System.arraycopy(this.routers, 1, this.routers, 0, this.nodeSize - 1);
                }

                ret = true;
            }else {
                // Search for the removed children
                for (int i = 1; i < this.nodeSize; i++) {
                    if (0 == this.children[i].nodeSize) {
                        // Empty node
                        System.arraycopy(this.children, i + 1, this.children, i    , this.nodeSize - i);
                        if (this.nodeSize > 2) {
                            System.arraycopy(this.routers, i, this.routers, i - 1, this.nodeSize - i);
                        }
                        ret = true;
                        break;
                    }
                }
            }

            if (ret) this.nodeSize--;
            return ret;
        }

        @SuppressWarnings("unchecked")
        @Override
        void borrowFromLeft(){
            BInternalNode leftNode = (BInternalNode)this.pre;

            BInternalNode anchor = (BInternalNode) getAnchor(leftNode);
            int slot = findSlot(anchor, leftNode, this);

            // Shift current tree node to right by 1, insert the right most node from left sibling
            // Shift is always safe
            System.arraycopy(this.routers,  0, this.routers,  1, this.nodeSize - 1);
            System.arraycopy(this.children, 0, this.children, 1, this.nodeSize);

            // Set new head router
            this.routers[0] = anchor.routers[slot];
            anchor.routers[slot] = leftNode.routers[leftNode.nodeSize - 2];
            // Move last children from leftNode
            this.children[0] = leftNode.children[leftNode.nodeSize - 1];
            this.children[0].parent = this;
            leftNode.children[leftNode.nodeSize - 1] = null;
            this.nodeSize++;
            leftNode.nodeSize--;
        }

        @SuppressWarnings("unchecked")
        @Override
        void mergeToLeft(){
            BInternalNode leftNode = (BInternalNode)this.pre;

            BInternalNode anchor = (BInternalNode) getAnchor(leftNode);
            int slot = findSlot(anchor, leftNode, this);

            for (int i = 0; i < this.nodeSize; i++){
                this.children[i].parent = leftNode;
            }

            // Move this node to the tail of the leftNode
            System.arraycopy(this.routers,  0, leftNode.routers,  leftNode.nodeSize, this.nodeSize - 1);
            System.arraycopy(this.children, 0, leftNode.children, leftNode.nodeSize, this.nodeSize);

            leftNode.routers[leftNode.nodeSize - 1] = anchor.routers[slot];
            leftNode.nodeSize += this.nodeSize;
            this.nodeSize = 0;

            if (null != this.next){this.next.pre = leftNode;}
            leftNode.next = this.next;
        }

        @SuppressWarnings("unchecked")
        @Override
        void borrowFromRight(){
            BInternalNode rightNode = (BInternalNode)this.next;
            BNode childToMove = rightNode.children[0];

            BInternalNode anchor = (BInternalNode) getAnchor(rightNode);
            int slot = findSlot(anchor, this, rightNode);

            // Move the head of the right node to the tail of the current node
            this.children[this.nodeSize] = childToMove;
            childToMove.parent = this;
            this.routers[this.nodeSize - 1] = anchor.routers[slot];
            anchor.routers[slot] = rightNode.routers[0];

            System.arraycopy(rightNode.routers,  1, rightNode.routers,  0, rightNode.nodeSize - 2);
            System.arraycopy(rightNode.children, 1, rightNode.children, 0, rightNode.nodeSize - 1);
            this.nodeSize++;
            rightNode.nodeSize--;
        }

        @SuppressWarnings("unchecked")
        @Override
        void mergeToRight(){
            BInternalNode rightNode = (BInternalNode)this.next;

            BInternalNode anchor = (BInternalNode) getAnchor(rightNode);
            int slot = findSlot(anchor, this, rightNode);

            for (int i = 0; i < this.nodeSize; i++){
                this.children[i].parent = rightNode;
            }

            // Move this node to the head of the rightNode
            System.arraycopy(rightNode.routers, 0, rightNode.routers, this.nodeSize, rightNode.nodeSize - 1);
            System.arraycopy(this.routers,      0, rightNode.routers, 0,             this.nodeSize - 1);

            System.arraycopy(rightNode.children, 0, rightNode.children, this.nodeSize, rightNode.nodeSize);
            System.arraycopy(this.children,      0, rightNode.children, 0,             this.nodeSize);

            rightNode.routers[this.nodeSize - 1] = anchor.routers[slot];
            rightNode.nodeSize += this.nodeSize;
            this.nodeSize = 0;

            if (null != this.pre){this.pre.next = rightNode;}
            rightNode.pre = this.pre;
        }

        @Override
        void collapseRoot() {
            // Collapse
            root = this.children[0];
            root.parent = null;
        }
    }

    /**
     * Leaf node of the BTree
     */
    public final class BLeafNode extends BNode {
        // Entry array for data storage
        AionMapEntry[] entries;

        @SuppressWarnings("unchecked")
        BLeafNode(){
            this.entries = new AionMap.AionMapEntry[2 * order];
        }

        public int searchForEntrySlot(K key){
            int i = 0;
            while (i < nodeSize && key.hashCode() != entries[i].hashCode()){
                i = i + 1;
            }

            if (i < nodeSize && key.hashCode() == entries[i].hashCode()){
                return i;
            }

            return -1;
        }

        public AionMapEntry searchForEntryInSlot(K key, int slot){
            AionMapEntry cur = entries[slot];
            while(cur != null){
                if (cur.key.equals(key)){
                    return cur;
                }
                cur = cur.next;
            }
            return null;
        }

        public AionMapEntry searchForEntry(K key){
            int i = 0;
            while (i < nodeSize && key.hashCode() != entries[i].hashCode()){
                i = i + 1;
            }

            if (i < nodeSize && key.hashCode() == entries[i].hashCode()){
                //Find the leaf slot, search within linked list
                AionMapEntry cur = entries[i];
                while(cur != null){
                    if (cur.key.equals(key)){
                        return cur;
                    }
                    cur = cur.next;
                }
            }

            return null;
        }

        @Override
        void insertNonFull(K key, V value) {
            int i = this.nodeSize;
            while (i > 0 && key.hashCode() < this.entries[i - 1].hashCode()){
                //this.entries[i] = this.entries[i - 1];
                i--;
            }

            System.arraycopy(this.entries, i, this.entries, i + 1, this.nodeSize - i);
            this.entries[i] = new AionMapEntry(key, value);
            this.nodeSize++;
        }

        @Override
        V delete(K key) {
            V ret = null;

            int slotidx = this.searchForEntrySlot(key);

            if (-1 == slotidx) {return ret;}

            AionMapEntry pre = entries[slotidx];
            AionMapEntry cur = pre.next;

            if (pre.key.equals(key)){
                ret = pre.value;
                entries[slotidx] = pre.next;
            }else {
                while (null != cur && !cur.equals(key)) {
                    cur = cur.next;
                    pre = pre.next;
                }
                if (null != cur) {
                    pre.next = cur.next;
                    ret = cur.value;
                }
            }

            if (null == entries[slotidx]){
                System.arraycopy(this.entries, slotidx + 1, this.entries, slotidx, this.nodeSize - slotidx - 1);
                entries[nodeSize - 1] = null;
                this.nodeSize--;
                this.rebalance();
            }

            return ret;
        }

        @SuppressWarnings("unchecked")
        @Override
        void borrowFromLeft(){
            BLeafNode leftNode = (BLeafNode)this.pre;

            // Need to update router within anchor node
            BInternalNode anchor = (BInternalNode) getAnchor(leftNode);
            int slot = findSlot(anchor, leftNode, this);

            // Shift current leaf to right by 1, insert the right most node from left sibling
            // Shift is always safe
            System.arraycopy(this.entries, 0, this.entries, 1, this.nodeSize);
            this.entries[0] = leftNode.entries[leftNode.nodeSize - 1];
            leftNode.entries[leftNode.nodeSize - 1] = null;
            this.nodeSize++;
            leftNode.nodeSize--;

            anchor.routers[slot] = this.entries[0].hashCode();
        }

        @SuppressWarnings("unchecked")
        @Override
        void mergeToLeft(){
            BLeafNode leftNode = (BLeafNode)this.pre;
            // Move this node to the tail of the leftNode
            System.arraycopy(this.entries, 0, leftNode.entries, leftNode.nodeSize, this.nodeSize);
            leftNode.nodeSize += this.nodeSize;
            this.nodeSize = 0;

            if (null != this.next){this.next.pre = leftNode;}
            leftNode.next = this.next;
        }

        @SuppressWarnings("unchecked")
        @Override
        void borrowFromRight(){
            BLeafNode rightNode = (BLeafNode)this.next;

            // Need to update router within anchor node
            BInternalNode anchor = (BInternalNode) getAnchor(rightNode);
            int slot = findSlot(anchor, this, rightNode);

            // Move the head of the right node to the tail of the current node
            this.entries[this.nodeSize] = rightNode.entries[0];
            System.arraycopy(rightNode.entries, 1, rightNode.entries, 0, rightNode.nodeSize - 1);
            this.nodeSize++;
            rightNode.nodeSize--;

            anchor.routers[slot] = rightNode.entries[0].hashCode();
        }

        @SuppressWarnings("unchecked")
        @Override
        void mergeToRight(){
            BLeafNode rightNode = (BLeafNode)this.next;
            // Move this node to the head of the rightNode
            System.arraycopy(rightNode.entries, 0, rightNode.entries, this.nodeSize, rightNode.nodeSize);
            System.arraycopy(this.entries, 0, rightNode.entries, 0, this.nodeSize);
            rightNode.nodeSize += this.nodeSize;
            this.nodeSize = 0;

            if (null != this.pre){this.pre.next = rightNode;}
            rightNode.pre = this.pre;
        }

        @Override
        void collapseRoot() {
            // Do nothing is root is leaf node
        }
    }

    /**
     * Entry (K V pair) of AionMap
     */
    public class AionMapEntry implements Map.Entry<K, V>{

        private K key;

        private V value;

        public AionMapEntry next;

        public AionMapEntry(K key, V value){
            this.key = key;
            this.value = value;
            this.next = null;
        }

        @Override
        public K getKey() {
            return key;
        }

        @Override
        public V getValue() {
            return value;
        }

        @Override
        public V setValue(V value) {
            V ret = this.value;
            this.value = value;
            return ret;
        }

        // Since our map is hashcode based, the hashcode of an entry is defined as hashcode of the key.
        @Override
        public int hashCode() {
            return key.hashCode();
        }
    }

    public abstract class AionAbstractCollection<E> implements Collection<E> {
        protected AionAbstractCollection(){
        }

        public abstract Iterator<E> iterator();

        public abstract int size();

        public boolean isEmpty() {
            return size() == 0;
        }

        public boolean contains(Object o) {
            Iterator<E> it = iterator();
            if (o==null) {
                while (it.hasNext())
                    if (it.next()==null)
                        return true;
            } else {
                while (it.hasNext())
                    if (o.equals(it.next()))
                        return true;
            }
            return false;
        }

        public Object[] toArray() {
            throw new UnsupportedOperationException();
        }

        public <T> T[] toArray(T[] a) {
            throw new UnsupportedOperationException();
        }

        public boolean add(E e) {
            throw new UnsupportedOperationException();
        }

        public boolean remove(Object o) {
            Iterator<E> it = iterator();
            if (o==null) {
                while (it.hasNext()) {
                    if (it.next()==null) {
                        it.remove();
                        return true;
                    }
                }
            } else {
                while (it.hasNext()) {
                    if (o.equals(it.next())) {
                        it.remove();
                        return true;
                    }
                }
            }
            return false;
        }

        public boolean containsAll(Collection<?> c) {
            for (Object e : c)
                if (!contains(e))
                    return false;
            return true;
        }

        public boolean addAll(Collection<? extends E> c) {
            boolean modified = false;
            for (E e : c)
                if (add(e))
                    modified = true;
            return modified;
        }

        public boolean removeAll(Collection<?> c) {
            boolean modified = false;
            Iterator<?> it = iterator();
            while (it.hasNext()) {
                if (c.contains(it.next())) {
                    it.remove();
                    modified = true;
                }
            }
            return modified;
        }

        public boolean retainAll(Collection<?> c) {
            boolean modified = false;
            Iterator<E> it = iterator();
            while (it.hasNext()) {
                if (!c.contains(it.next())) {
                    it.remove();
                    modified = true;
                }
            }
            return modified;
        }

        public void clear() {
            Iterator<E> it = iterator();
            while (it.hasNext()) {
                it.next();
                it.remove();
            }
        }
    }

    public final class AionMapKeySet extends AionAbstractCollection<K> implements Set<K> {
        @Override
        public final int size() {
            return AionMap.this.size;
        }

        @Override
        public final Iterator<K> iterator() {
            return new AionMapKeyIterator();
        }

        @Override
        public final void clear() {
            AionMap.this.clear();
        }

        @Override
        public final boolean contains(Object o) {
            return AionMap.this.containsKey(o);
        }

        @Override
        public final boolean remove(Object key) {
            return null != AionMap.this.remove(key);
        }
    }

    public final class AionMapValues extends AionAbstractCollection<V> implements Collection<V> {
        @Override
        public final int size() {
            return AionMap.this.size;
        }

        @Override
        public final Iterator<V> iterator() {
            return new AionMapValueIterator();
        }

        @Override
        public final void clear() {
            AionMap.this.clear();
        }

        @Override
        public final boolean contains(Object o) {
            return AionMap.this.containsValue(o);
        }
    }

    public final class AionMapEntrySet extends AionAbstractCollection<Entry<K, V>> implements Set<Entry<K, V>> {
        @Override
        public final int size() {
            return AionMap.this.size;
        }

        @Override
        public final Iterator<Entry<K, V>> iterator() {
            return new AionMapEntryIterator();
        }

        @Override
        public final void clear() {
            AionMap.this.clear();
        }

        @SuppressWarnings("unchecked")
        @Override
        public final boolean contains(Object o) {
            K key   = (K) ((Entry<K, V>) o).getKey();
            V value = (V) ((Entry<K, V>) o).getValue();

            return AionMap.this.containsKey(o) && AionMap.this.get(key).equals(value);
        }

        @SuppressWarnings("unchecked")
        @Override
        public final boolean remove(Object ent) {
            Entry<K, V> entry = (Entry<K, V> ) ent;
            return null != AionMap.this.remove(entry.getKey());
        }
    }

    public abstract class AionMapIterator{
        BLeafNode curLeaf;

        AionMapEntry preEntry;

        AionMapEntry curEntry;

        int curSlot;

        AionMapIterator(){
            curLeaf = AionMap.this.getLeftMostLeaf();
            curSlot = 0;
            curEntry = curLeaf.entries[curSlot];
            preEntry = null;
        }

        public boolean hasNext() {
            return (null != curEntry);
        }

        @SuppressWarnings("unchecked")
        public AionMapEntry nextEntry() {
            AionMapEntry elt = null;

            if (null != curEntry){
                elt = curEntry;

                // Advance cursor
                if (null != curEntry.next){
                    curEntry = curEntry.next;
                }else if (curSlot + 1 < curLeaf.nodeSize){
                    curSlot++;
                    curEntry = curLeaf.entries[curSlot];
                }else if (null != curLeaf.next){
                    curLeaf = (BLeafNode) curLeaf.next;
                    curSlot = 0;
                    curEntry = curLeaf.entries[curSlot];
                }else{
                    curEntry = null;
                }
            } else {
                throw new NoSuchElementException();
            }
            preEntry = elt;
            return elt;
        }

        public void remove() {
            AionMap.this.remove(preEntry.key);
            if (null != curEntry) {
                curLeaf = AionMap.this.searchForLeaf((K) curEntry.key);
                curSlot = curLeaf.searchForEntrySlot((K) curEntry.key);
            }
        }
    }

    public final class AionMapEntryIterator extends AionMapIterator implements Iterator<Entry<K, V>> {
        @Override
        public AionMapEntry next() {
            return nextEntry();
        }
    }

    public final class AionMapKeyIterator extends AionMapIterator implements Iterator<K> {
        @Override
        public K next() {
            return nextEntry().key;
        }
    }

    public final class AionMapValueIterator extends AionMapIterator implements Iterator<V> {
        @Override
        public V next() {
            return nextEntry().value;
        }
    }


    /**
     * Returns the left most {@link BLeafNode} of the BTree.
     * This node serves as the entry point of data traversal.
     *
     * @return the left most {@link BLeafNode} of the BTree.
     */
    @SuppressWarnings("unchecked")
    BLeafNode getLeftMostLeaf(){
        BNode cur = this.root;
        while (!(cur instanceof AionMap.BLeafNode)) {
            cur = ((BInternalNode) cur).children[0];
        }
        return (BLeafNode)cur;
    }

    @SuppressWarnings("unchecked")
    private BLeafNode searchForLeaf(K key){
        BNode cur = this.root;

        while (!(cur instanceof AionMap.BLeafNode)){
            BInternalNode tmp = (BInternalNode) cur;

            int i = tmp.nodeSize - 1;
            while (i > 0 && key.hashCode() < tmp.routers[i - 1]){
                i--;
            }
            cur = tmp.children[i];
        }

        return (BLeafNode)cur;
    }

    private void keyNullCheck(Object key){
        if (null == key){
            throw new NullPointerException("AionMap does not allow empty key.");
        }
    }

    private void bInsert(K key, V value){
        BNode r = this.root;
        if (r.nodeSize == ((2 * order))){
            BInternalNode s = new BInternalNode();
            this.root = s;
            s.nodeSize = 1;
            s.children[0] = r;
            r.parent = s;
            bSplitChild(s, 0);
            s.insertNonFull(key, value);
        }else{
            r.insertNonFull(key, value);
        }
    }

    private void bSplitChild(BInternalNode x, int i){
        if (x.children[i] instanceof AionMap.BInternalNode){
            bSplitTreeChild(x, i);
        }else{
            bSplitLeafChild(x, i);
        }
    }

    @SuppressWarnings("unchecked")
    private void bSplitTreeChild(BInternalNode x, int i){
        // Left node
        BInternalNode y = (BInternalNode) x.children[i];
        // Right node
        BInternalNode z = new BInternalNode();

        // Right node has t children
        z.nodeSize = order;

        System.arraycopy(y.routers , order, z.routers , 0, order - 1);
        System.arraycopy(y.children, order, z.children, 0, order);
        for (int j = 0; j < order; j++){
            z.children[j].parent = z;
        }

        // Left node has t children
        y.nodeSize = order;
        int newXRouter = y.routers[order - 1];
        y.routers [order - 1] = 0;
        y.children[2 * order - 1] = null;

        // Link tree node
        z.next = y.next;
        if (null != y.next) {
            y.next.pre = z;
        }
        z.pre = y;
        y.next = z;
        y.parent = x;
        z.parent = x;

        // Shift parent node
        if (x.nodeSize > 1) {
            System.arraycopy(x.routers, i, x.routers, i + 1, x.nodeSize - i - 1);
            System.arraycopy(x.children, i, x.children, i + 1, x.nodeSize - i);
        }

        x.children[i + 1] = z;
        x.routers [i] = newXRouter;
        x.nodeSize++;
    }

    @SuppressWarnings("unchecked")
    private void bSplitLeafChild(BInternalNode x, int i){
        BLeafNode y = (BLeafNode) x.children[i];
        BLeafNode z = new BLeafNode();

        // Right node has t children
        z.nodeSize = order;
        // Move t children to right node
        System.arraycopy(y.entries, order, z.entries, 0, order);

        y.nodeSize = order;

        // Link leaf nodes
        z.next = y.next;
        if (null != y.next) {
            y.next.pre = z;
        }
        z.pre = y;
        y.next = z;
        y.parent = x;
        z.parent = x;

        // Shift parent node
        if (x.nodeSize > 0) {
            System.arraycopy(x.routers, i, x.routers, i + 1, x.nodeSize - i - 1);
            System.arraycopy(x.children, i, x.children, i + 1, x.nodeSize - i);
        }

        x.children[i + 1] = z;
        x.routers [i] = z.entries[0].hashCode();
        x.nodeSize++;
    }

    private int findSlot(BInternalNode anchor, BLeafNode left, BLeafNode right){
        int lvalue = left.entries[left.nodeSize - 1].hashCode();
        int rvalue = right.entries[0].hashCode();

        int i = 0;
        while (!(lvalue < anchor.routers[i] && rvalue >= anchor.routers[i])){
            i++;
        }
        return i;
    }

    private int findSlot(BInternalNode anchor, BInternalNode left, BInternalNode right) {

        int lvalue = left.routers[0];
        int rvalue = right.routers[0];

        int i = 0;
        while (!(lvalue < anchor.routers[i] && rvalue >= anchor.routers[i])) {
            i++;
        }

        return i;
    }
}
