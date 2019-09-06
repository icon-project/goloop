package org.aion.avm.tooling.deploy.eliminator;

import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

public class ClassInfo {

    private final String className;

    private final Map<String, MethodInfo> methodMap;

    // These are methods that we want to always flag as reachable for some reason, usually because they are very fundamental
    // Examples include overriden implementations of Object.hashcode() and equals()
    private final List<MethodInfo> alwaysReachables;

    private final Set<ClassInfo> parents = new HashSet<>();
    private final Set<ClassInfo> children = new HashSet<>();

    private ClassInfo superInfo;

    private final boolean isInterface;
    private final boolean isAbstract;

    public ClassInfo(String className, boolean isInterface, boolean isAbstract, Map<String, MethodInfo> methodMap, List<MethodInfo> alwaysReachables) {
        this.className = className;
        this.isInterface = isInterface;
        this.isAbstract = isAbstract;
        this.methodMap = methodMap;
        this.alwaysReachables = alwaysReachables;
    }

    public void setSuperclass(ClassInfo superInfo) {
        this.superInfo = superInfo;
        this.addToParents(superInfo);
    }

    public ClassInfo getSuperclass() {
        return superInfo;
    }

    // Given a method identifier, this method walks up the class hierarchy finding the first concrete implementation of that method,
    // and returns the corresponding MethodInfp
    public MethodInfo getDeclaration(String methodId) {
        MethodInfo methodInfo = methodMap.get(methodId);
        return (null != methodInfo) ? methodInfo : superInfo.getDeclaration(methodId);
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

    public Set<ClassInfo> getParents() {
        return parents;
    }

    public Set<ClassInfo> getChildren() {
        return children;
    }
}
