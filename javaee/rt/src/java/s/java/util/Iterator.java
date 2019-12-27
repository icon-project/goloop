package s.java.util;

import i.IObject;

public interface Iterator<E extends IObject> extends IObject {
    boolean avm_hasNext();

    E avm_next();

    default void avm_remove() {
        throw new UnsupportedOperationException("remove");
    }
}
