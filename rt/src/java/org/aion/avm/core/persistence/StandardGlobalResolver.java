package org.aion.avm.core.persistence;

import org.aion.avm.core.NodeEnvironment;
import i.ConstantToken;
import i.InternedClasses;
import i.RuntimeAssertionError;


public class StandardGlobalResolver implements IGlobalResolver {
    private final InternedClasses internedClassMap;
    private final ClassLoader classLoader;

    public StandardGlobalResolver(InternedClasses internedClassMap, ClassLoader classLoader) {
        this.internedClassMap = internedClassMap;
        this.classLoader = classLoader;
    }

    @Override
    public String getAsInternalClassName(Object target) {
        return (target instanceof s.java.lang.Class)
                ? ((s.java.lang.Class<?>)target).getRealClass().getName()
                : null;
    }

    @Override
    public Object getClassObjectForInternalName(String internalClassName) {
        try {
            Class<?> underlyingClass = this.classLoader.loadClass(internalClassName);
            s.java.lang.Class<?> internedClass = this.internedClassMap.get(underlyingClass);
            return internedClass;
        } catch (ClassNotFoundException e) {
            // This can only fail if we were given the wrong loader.
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    @Override
    public int getAsConstant(Object target) {
        int constant = 0;

        RuntimeAssertionError.assertTrue(target instanceof s.java.lang.Object);
        if(((s.java.lang.Object) target).readIndex < -1)
        {
            constant = ConstantToken.getConstantIdFromReadIndex(((s.java.lang.Object) target).readIndex);
        }
        return constant;
    }

    @Override
    public Object getConstantForIdentifier(int constantIdentifier) {
        return NodeEnvironment.singleton.getConstantMap().get(constantIdentifier);
    }
}
