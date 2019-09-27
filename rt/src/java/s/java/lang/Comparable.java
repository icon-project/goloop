package s.java.lang;

import i.IObject;

public interface Comparable<T extends IObject> extends IObject {

    public int avm_compareTo(T o);
}
