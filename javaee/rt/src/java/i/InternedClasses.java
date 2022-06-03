package i;

import java.util.IdentityHashMap;


/**
 * Really just a high-level wrapper over an IdentityHashMap to contain real classes to shadow classes.
 * This exists because some of the logic was duplicated in a few places.
 */
public class InternedClasses {
    private final IdentityHashMap<Class<?>, s.java.lang.Class<?>> internedClassWrappers;

    public InternedClasses() {
        this.internedClassWrappers = new IdentityHashMap<>();
    }

    public <T> s.java.lang.Class<T> get(Class<T> underlyingClass) {
        s.java.lang.Class<?> internedClass = this.internedClassWrappers.get(underlyingClass);
        if (null == internedClass) {
            internedClass = new s.java.lang.Class<>(underlyingClass);
            this.internedClassWrappers.put(underlyingClass, internedClass);
        }
        return (s.java.lang.Class<T>)internedClass;
    }

    public void purgeEnumCaches() {
        for (var cls : internedClassWrappers.values()) {
            cls.purgeEnumCache();
        }
    }
}
