package org.aion.avm.tooling.deploy.renamer;

import org.objectweb.asm.tree.ClassNode;

import java.util.HashMap;
import java.util.Map;

public class ClassRenamer {
    private static final boolean printEnabled = false;

    // NOTE package name is removed
    public static Map<String, String> renameClasses(Map<String, ClassNode> classMap) {

        // Key should be class name (slash format)
        Map<String, String> classNameMap = new HashMap<>();
        NameGenerator generator = new NameGenerator();

        for (String className : classMap.keySet()) {
            String newClassName;
            if (className.contains("$")) {
                String outerClassName = className.substring(0, className.lastIndexOf('$'));
                String newOuterClassName = classNameMap.get(outerClassName);
                if (newOuterClassName == null) {
                    newOuterClassName = generator.getNextClassName();
                    classNameMap.put(className, newOuterClassName);
                }
                newClassName = newOuterClassName + "$" + generator.getNextClassName();
                classNameMap.put(className, newClassName);
            } else {
                newClassName = generator.getNextClassName();
                classNameMap.put(className, newClassName);
            }
            if (printEnabled) {
                System.out.println("Renaming class " + className + " to " + newClassName);
            }
        }
        return classNameMap;
    }
}
