package s.java.util;

import i.IObject;
import i.IObjectArray;
import s.java.lang.Iterable;

public interface Collection<E extends IObject> extends Iterable<E>{

    // Query Operations

    int avm_size();

    boolean avm_isEmpty();

    boolean avm_contains(IObject o);

    IObjectArray avm_toArray();

    boolean avm_add(E e);

    boolean avm_remove(IObject o);

    boolean avm_containsAll(Collection<? extends IObject> c);

    boolean avm_addAll(Collection<? extends E> c);

    boolean avm_removeAll(Collection<? extends IObject> c);

    boolean avm_retainAll(Collection<? extends IObject> c);

    void avm_clear();

    boolean avm_equals(IObject o);

    int avm_hashCode();

    //Default


    //Exclude

    //<T> T[] toArray(T[] a);
}
