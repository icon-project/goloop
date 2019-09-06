package org.aion.avm.tooling.deploy.eliminator;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Map.Entry;
import org.objectweb.asm.ClassReader;

public class JarDependencyCollector {

    private final List<ClassDependencyVisitor> classVisitors = new ArrayList<>();
    private final Map<String, ClassInfo> classInfoMap;

    // The ClassInfos in the returned map have all methods marked as NOT reachable.
    public static Map<String, ClassInfo> getClassInfoMap(Map<String, byte[]> classMap) {
        JarDependencyCollector jarDependencyCollector = new JarDependencyCollector(classMap);
        return jarDependencyCollector.getClassInfoMap();
    }

    private JarDependencyCollector(Map<String, byte[]> classMap) {

        classInfoMap = WhitelistPopulator.getWhitelistedClassInfos();

        for (Entry<String, byte[]> entry: classMap.entrySet()) {
            visitClass(entry.getKey(), entry.getValue());
        }

        setParentsAndChildren();
    }

    // Should only be called once per class
    private void visitClass(String classSlashName, byte[] classBytes) {

        ClassReader reader = new ClassReader(classBytes);

        ClassDependencyVisitor classVisitor = new ClassDependencyVisitor(classSlashName);
        classVisitors.add(classVisitor);
        reader.accept(classVisitor, 0);

        // We put in an "incomplete" ClassInfo object at this stage. Information about the type hierarchy happens in the next step.
        ClassInfo classInfo = new ClassInfo(classSlashName, classVisitor.isInterface(),
            classVisitor.isAbstract(), classVisitor.getMethodMap(), classVisitor.getAlwaysReachables());
        classInfoMap.put(classSlashName, classInfo);
    }

    private void setParentsAndChildren() {
        for (ClassDependencyVisitor visitor : classVisitors) {
            String classSlashName = visitor.getClassSlashName();
            String superSlashName = visitor.getSuperSlashName();
            String[] interfaces = visitor.getInterfaces();
            ClassInfo classInfo = classInfoMap.get(classSlashName);

            if (null == superSlashName && !classSlashName.equals("java/lang/Object")) {
                throw new RuntimeException("All classses except Object must have a superclass");
            } else {
                ClassInfo superInfo = classInfoMap.get(superSlashName);
                classInfo.setSuperclass(superInfo);
                superInfo.addToChildren(classInfo);
                classInfo.addToParents(superInfo);
            }

            // ASM's documentation says it's possible for interfaces to be null, so we check here
            // It appears that the interfaces object is empty when no interfaces exist (instead of being null)
            if (null != interfaces) {
                for (String interfaceName : interfaces) {
                    ClassInfo interfaceInfo = classInfoMap.get(interfaceName);
                    interfaceInfo.addToChildren(classInfo);
                    classInfo.addToParents(interfaceInfo);
                }
            }
        }
    }

    private Map<String, ClassInfo> getClassInfoMap() {
        return classInfoMap;
    }
}
