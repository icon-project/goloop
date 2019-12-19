package s.java.util;

import i.IObject;
import i.IObjectArray;

public interface List<E> extends Collection<E> {

    // Query Operations

    int avm_size();

    boolean avm_isEmpty();

    boolean avm_contains(IObject o);

    IObjectArray avm_toArray();

    boolean avm_add(IObject e);

    boolean avm_remove(IObject o);

    boolean avm_containsAll(Collection<?> c);

    boolean avm_addAll(Collection<? extends E> c);

    boolean avm_removeAll(Collection<?> c);

    boolean avm_retainAll(Collection<?> c);

    void avm_clear();

    boolean avm_equals(IObject o);

    int avm_hashCode();

    // Positional Access Operations

    IObject avm_get(int index);

    IObject avm_set(int index, IObject element);

    void avm_add(int index, IObject element);

    IObject avm_remove(int index);

    int avm_indexOf(IObject o);

    int avm_lastIndexOf(IObject o);

    ListIterator<IObject> avm_listIterator();

    ListIterator<IObject> avm_listIterator(int index);

    // View

    List<IObject> avm_subList(int fromIndex, int toIndex);

}
