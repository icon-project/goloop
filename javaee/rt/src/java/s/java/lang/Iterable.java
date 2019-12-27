package s.java.lang;

import i.IObject;
import s.java.util.Iterator;

public interface Iterable<T extends IObject> extends IObject {
    Iterator<T> avm_iterator();

    //Default
}
