package s.java.util;

import foundation.icon.ee.utils.IObjects;
import i.IObject;

// iteration order is deterministic
// may have null value
// confirms standard set hashCode and equals
public class UnmodifiableArraySet<E extends IObject>
        extends UnmodifiableArrayCollection<E>
        implements Set<E> {
    UnmodifiableArraySet(IObject[] data) {
        super(data);
    }

    public boolean avm_equals(IObject o) {
        if (o == this) {
            return true;
        }
        if (!(o instanceof Set)) {
            return false;
        }
        Set<?> s = (Set<?>) o;
        if (s.avm_size() != data.length) {
            return false;
        }
        try {
            return avm_containsAll(s);
        } catch (ClassCastException | NullPointerException ex) {
            return false;
        }
    }

    public int avm_hashCode() {
        int h = 0;
        for (var e : data) {
            if (e != null) {
                h += IObjects.hashCode(e);
            }
        }
        return h;
    }

    public UnmodifiableArraySet(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }
}
