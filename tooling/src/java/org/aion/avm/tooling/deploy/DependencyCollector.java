package org.aion.avm.tooling.deploy;

import org.objectweb.asm.Type;

import java.util.HashSet;
import java.util.Set;

public class DependencyCollector {
    private final Set<String> dependencies = new HashSet<String>();

    Set<String> getDependencies() {
        return dependencies;
    }

    void addMethodDescriptor(final String desc) {
        add(Type.getReturnType(desc));

        for (Type t : Type.getArgumentTypes(desc)) {
            add(t);
        }
    }

    void addDescriptor(final String desc) {
        add(Type.getType(desc));
    }

    void addType(String type) {
        if (type != null)
            add(Type.getObjectType(type));
    }

    private void add(final Type type) {
        int t = type.getSort();
        if (t == Type.ARRAY) {
            add(type.getElementType());
        } else if (t == Type.OBJECT) {
            addClassName(type.getClassName());
        }
    }

    private void addClassName(String name) {
        if (name == null) return;
        dependencies.add(name);
    }
}
