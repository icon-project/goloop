package org.aion.avm.tooling.deploy.renamer;

import org.objectweb.asm.tree.ClassNode;

import java.util.HashMap;
import java.util.Map;
import java.util.Set;

public class ClassRenamer {
    private static boolean printEnabled = false;

    //NOTE package name is removed
    public static Map<String, String> renameClasses(Map<String, ClassNode> classMap, String mainClassName) {

        // Key should be class name (slash format)
        Map<String, String> classNameMap = new HashMap<>();
        NameGenerator generator = new NameGenerator();

        for (String className : classMap.keySet()) {
            String newClassName;
            if (className.contains("$")) {
                newClassName = classNameMap.get(className.substring(0, className.lastIndexOf('$'))) + "$" + generator.getNextClassName();
                classNameMap.put(className, newClassName);
            } else {
                newClassName = className.equals(mainClassName) ? NameGenerator.getNewMainClassName() : generator.getNextClassName();
                classNameMap.put(className, newClassName);
            }
            if (printEnabled) {
                System.out.println("Renaming class " + className + " to " + newClassName);
            }
        }
        return classNameMap;
    }

}
