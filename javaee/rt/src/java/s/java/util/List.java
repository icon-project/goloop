package s.java.util;

import i.IObject;

public interface List<E extends IObject> extends Collection<E> {

    // Positional Access Operations

    E avm_get(int index);

    IObject avm_set(int index, E element);

    void avm_add(int index, E element);

    E avm_remove(int index);

    int avm_indexOf(IObject o);

    int avm_lastIndexOf(IObject o);

    ListIterator<E> avm_listIterator();

    ListIterator<E> avm_listIterator(int index);

    // View

    List<E> avm_subList(int fromIndex, int toIndex);

}
