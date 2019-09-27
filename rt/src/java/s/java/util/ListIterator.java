package s.java.util;

import i.IObject;

public interface ListIterator<E> extends Iterator<E>{

    boolean avm_hasNext();

    IObject avm_next();

    boolean avm_hasPrevious();

    IObject avm_previous();

    int avm_nextIndex();

    int avm_previousIndex();

    void avm_remove();

    void avm_set(IObject e);

    void avm_add(IObject e);

}
