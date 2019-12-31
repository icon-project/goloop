package s.java.util;

import a.ObjectArray;
import i.IObject;
import i.IObjectArray;
import s.java.lang.Object;

public class UnmodifiableArrayCollection<E extends IObject>
        extends UnmodifiableArrayContainer
        implements Collection<E> {
    UnmodifiableArrayCollection(IObject[] elems) {
        super(elems);
    }

    public int avm_size() {
        return data.length;
    }

    public boolean avm_contains(IObject o) {
        return indexOf(o) >= 0;
    }

    public IObjectArray avm_toArray() {
        return new ObjectArray(data.clone());
    }

    public boolean avm_add(E e) {
        throw new UnsupportedOperationException();
    }

    public boolean avm_remove(IObject o) {
        throw new UnsupportedOperationException();
    }

    public boolean avm_containsAll(Collection<? extends IObject> c) {
        var iter = c.avm_iterator();
        while (iter.avm_hasNext()) {
            if (!avm_contains(iter.avm_next())) {
                return false;
            }
        }
        return true;
    }

    public boolean avm_addAll(Collection<? extends E> c) {
        throw new UnsupportedOperationException();
    }

    public boolean avm_removeAll(Collection<? extends IObject> c) {
        throw new UnsupportedOperationException();
    }

    public boolean avm_retainAll(Collection<? extends IObject> c) {
        throw new UnsupportedOperationException();
    }

    class Iter extends Object implements Iterator<E> {
        int index;

        Iter() {
        }

        Iter(int index) {
            this.index = index;
        }

        public boolean avm_hasNext() {
            return index < data.length;
        }

        public E avm_next() {
            return (E) data[index++];
        }

        public void avm_remove() {
            throw new UnsupportedOperationException();
        }
    }

    public Iterator<E> avm_iterator() {
        return new Iter();
    }

    public UnmodifiableArrayCollection(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }
}
