package s.java.util;

import i.IObject;

public interface Iterator<E> extends IObject {
    boolean avm_hasNext();

    IObject avm_next();

    default void avm_remove() {
        throw new UnsupportedOperationException("remove");
    }
}
