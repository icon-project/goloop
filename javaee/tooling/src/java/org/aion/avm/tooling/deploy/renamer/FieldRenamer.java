package org.aion.avm.tooling.deploy.renamer;

import org.aion.avm.tooling.deploy.eliminator.ClassInfo;
import org.objectweb.asm.tree.ClassNode;
import org.objectweb.asm.tree.FieldNode;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class FieldRenamer {
    private static boolean printEnabled = false;

    public static Map<String, String> renameFields(Map<String, ClassNode> classMap, Map<String, ClassInfo> classInfoMap) {
        Map<String, String> newFieldsMappingsForRemapper = new HashMap<>();
        NameGenerator generator = new NameGenerator();

        for (Map.Entry<String, ClassNode> e : classMap.entrySet()) {
            String className = e.getKey();
            List<FieldNode> fieldNodes = e.getValue().fields;

            for (FieldNode f : fieldNodes) {
                if (!newFieldsMappingsForRemapper.containsKey(makeFullFieldName(className, f.name))) {
                    String newName = generator.getNextMethodOrFieldName(null);
                    newFieldsMappingsForRemapper.put(makeFullFieldName(className, f.name), newName);
                    printInfo(className, f.name, newName);

                    for (ClassInfo c : classInfoMap.get(className).getChildren()) {
                        newFieldsMappingsForRemapper.put(makeFullFieldName(c.getClassName(), f.name), newName);
                        printInfo(c.getClassName(), f.name, newName);
                    }
                }
            }
        }
        return newFieldsMappingsForRemapper;
    }

    private static String makeFullFieldName(String className, String fieldName) {
        return className + "." + fieldName;
    }

    private static void printInfo(String className, String oldName, String newName) {
        if (printEnabled) {
            System.out.println("<field> Class " + className + ": " + oldName + " -> " + newName);
        }
    }
}
