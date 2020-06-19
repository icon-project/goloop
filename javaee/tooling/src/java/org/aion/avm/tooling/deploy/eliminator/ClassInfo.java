package org.aion.avm.tooling.deploy.eliminator;

import java.lang.reflect.Modifier;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.Set;

public class ClassInfo {

    private final String className;
    private final Map<String, MethodInfo> methodMap;

    // These are methods that we want to always flag as reachable for some reason, usually because they are very fundamental
    // Examples include overridden implementations of Object.hashCode() and equals()
    private final List<MethodInfo> alwaysReachables;

    private final Set<ClassInfo> parents = new HashSet<>();
    private final Set<ClassInfo> children = new HashSet<>();

    private ClassInfo superInfo;

    private final boolean isInterface;
    private final boolean isAbstract;
    private final boolean isSystemClass;

    public ClassInfo(String className, boolean isInterface, boolean isAbstract,
                     Map<String, MethodInfo> methodMap, List<MethodInfo> alwaysReachables) {
        this.className = className;
        this.isInterface = isInterface;
        this.isAbstract = isAbstract;
        this.methodMap = methodMap;
        this.alwaysReachables = alwaysReachables;
        this.isSystemClass = false;
    }

    public ClassInfo(String className, Map<String, MethodInfo> methodMap, int modifiers) {
        this.className = className;
        this.methodMap = methodMap;
        this.isInterface = Modifier.isInterface(modifiers);
        this.isAbstract = Modifier.isAbstract(modifiers) && !isInterface;
        this.alwaysReachables = new ArrayList<>();
        this.isSystemClass = true;
    }

    public void setSuperclass(ClassInfo superInfo) {
        this.superInfo = superInfo;
        this.addToParents(superInfo);
    }

    public ClassInfo getSuperclass() {
        return superInfo;
    }

    // Given a method identifier, this method returns the matching declaration in the class hierarchy
    // If the class itself declares the method, it returns that methodInfo
    // If not, it walks up the *class* hierarchy, and returns the closest matching methodInfo
    // If not, it simply goes through the parent interfaces, and returns the first matching methodInfo it finds.
    // It should not return null on any legal code
    public MethodInfo getDeclaration(String methodId) {

        MethodInfo methodInfo = getConcreteImplementation(methodId);

        // Superclasses didn't have the method, try the interfaces now
        // If multiple parent interfaces declare this method, there is no guarantee about which one will be returned
        for (Iterator<ClassInfo> iterator = parents.iterator(); iterator.hasNext() && null == methodInfo; ) {
            ClassInfo parentInfo = iterator.next();
            if (parentInfo.isInterface()) {
                methodInfo = parentInfo.getMethodMap().get(methodId);
            }
        }

        return methodInfo;
    }

    // Given a method identifier, this method returns the matching declaration in the class hierarchy
    // If the class itself declares the method, it returns that methodInfo
    // If not, it walks up the *class* hierarchy, and returns the closest matching methodInfo
    public MethodInfo getConcreteImplementation(String methodId) {

        // Does the class itself declare this method?
        MethodInfo methodInfo = methodMap.get(methodId);

        // Class didn't have the method, walk up the superclass
        if (null == methodInfo && null != superInfo) {
            methodInfo = superInfo.getConcreteImplementation(methodId);
        }

        return methodInfo;
    }

    public void addToParents(ClassInfo parent) {
        this.parents.add(parent);
        this.parents.addAll(parent.parents);
        for (ClassInfo childClassInfo : children) {
            childClassInfo.addToParents(parent);
        }
    }

    public void addToChildren(ClassInfo child) {
        this.children.add(child);
        this.children.addAll(child.children);
        for (ClassInfo parentsClassInfo : parents) {
            parentsClassInfo.addToChildren(child);
        }
    }

    public Map<String, MethodInfo> getMethodMap() {
        return methodMap;
    }

    public String getClassName() {
        return className;
    }

    public List<MethodInfo> getAlwaysReachables() {
        return alwaysReachables;
    }

    public boolean isInterface() {
        return isInterface;
    }

    public boolean isAbstract() {
        return isAbstract;
    }

    public boolean isSystemClass() {
        return isSystemClass;
    }

    public Set<ClassInfo> getParents() {
        return parents;
    }

    public Set<ClassInfo> getChildren() {
        return children;
    }
}
