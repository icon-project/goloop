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

        // Set the inheritance relationships manually
        ClassInfo comparableInfo = classInfoMap.get("java/lang/Comparable");
        ClassInfo iterableInfo = classInfoMap.get("java/lang/Iterable");
        ClassInfo serializableInfo = classInfoMap.get("java/io/Serializable");
        ClassInfo throwableInfo = classInfoMap.get("java/lang/Throwable");
        ClassInfo exceptionInfo = classInfoMap.get("java/lang/Exception");
        ClassInfo runtimeExceptionInfo = classInfoMap.get("java/lang/RuntimeException");
        ClassInfo enumInfo = classInfoMap.get("java/lang/Enum");
        ClassInfo collectionInfo = classInfoMap.get("java/util/Collection");
        ClassInfo setInfo = classInfoMap.get("java/util/Set");
        ClassInfo listInfo = classInfoMap.get("java/util/List");
        ClassInfo iteratorInfo = classInfoMap.get("java/util/Iterator");
        ClassInfo listIteratorInfo = classInfoMap.get("java/util/ListIterator");

        collectionInfo.addToParents(iterableInfo);
        setInfo.addToParents(collectionInfo);
        throwableInfo.addToParents(serializableInfo);
        exceptionInfo.setSuperclass(throwableInfo);
        enumInfo.addToParents(comparableInfo);
        enumInfo.addToParents(serializableInfo);
        listIteratorInfo.addToParents(iteratorInfo);
        runtimeExceptionInfo.setSuperclass(exceptionInfo);
        listInfo.addToParents(collectionInfo);

        return classInfoMap;
    }
}
