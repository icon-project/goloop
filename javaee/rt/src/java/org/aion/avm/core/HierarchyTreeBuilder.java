package org.aion.avm.core;

import java.util.HashMap;
import java.util.Map;

import org.aion.avm.core.types.ClassInfo;
import org.aion.avm.core.types.Forest;
import i.RuntimeAssertionError;


/**
 * Provides a minimal interface for quickly building the Forest() objects in tests.
 * Returns itself from addClass for easy chaining in boiler-plate test code.
 */
public class HierarchyTreeBuilder {
    private final Forest<String, ClassInfo> classHierarchy = new Forest<>();
    private final Map<String, Forest.Node<String, ClassInfo>> nameCache = new HashMap<>();

    public HierarchyTreeBuilder addClass(String name, String superclass, boolean isInterface, byte[] code) {
        // NOTE:  These are ".-style" names.
        RuntimeAssertionError.assertTrue(-1 == name.indexOf("/"));
        RuntimeAssertionError.assertTrue(-1 == superclass.indexOf("/"));

        // already added as parent
        if (this.nameCache.containsKey(name)){
            Forest.Node<String, ClassInfo> cur = this.nameCache.get(name);
            cur.setContent(new ClassInfo(isInterface, code));

            Forest.Node<String, ClassInfo> parent = this.nameCache.get(superclass);
            if (null == parent) {
                parent = new Forest.Node<>(superclass, null);
                this.nameCache.put(superclass,  parent);
            }
            this.classHierarchy.add(parent, cur);

        }else {

            Forest.Node<String, ClassInfo> parent = this.nameCache.get(superclass);
            if (null == parent) {
                // Must be a root.
                parent = new Forest.Node<>(superclass, null);
                this.nameCache.put(superclass, parent);
            }

            // Inject into tree.
            Forest.Node<String, ClassInfo> child = new Forest.Node<>(name, new ClassInfo(isInterface, code));

            // Cache result.
            this.nameCache.put(name, child);

            // Add connection.
            this.classHierarchy.add(parent, child);
        }
        
        return this;
    }

    public Forest<String, ClassInfo> asMutableForest() {
        return this.classHierarchy;
    }
}
