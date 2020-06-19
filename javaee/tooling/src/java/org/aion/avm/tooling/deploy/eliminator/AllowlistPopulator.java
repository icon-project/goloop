package org.aion.avm.tooling.deploy.eliminator;

import org.aion.avm.core.util.AllowlistProvider;
import org.aion.avm.utilities.Utilities;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class AllowlistPopulator {

    public static Map<String, ClassInfo> getClassInfoMap() {
        Map<String, ClassInfo> classInfoMap = new HashMap<>();

        try {
            Map<Class<?>, List<AllowlistProvider.MethodDescriptor>> allowlist = AllowlistProvider.getClassLibraryMap();
            allowlist.forEach((clazz, methodDescriptors) -> {
                Map<String, MethodInfo> methodMap = new HashMap<>();
                methodDescriptors.forEach(md -> {
                    String methodName = md.name + md.parameters;
                    methodMap.put(methodName, new MethodInfo(methodName, md.isStatic));
                });
                String className = Utilities.fullyQualifiedNameToInternalName(clazz.getName());
                ClassInfo ci = new ClassInfo(className, methodMap, clazz.getModifiers());
                classInfoMap.put(className, ci);
            });
        } catch (ClassNotFoundException e) {
            e.printStackTrace();
        }

        return classInfoMap;
    }
}
