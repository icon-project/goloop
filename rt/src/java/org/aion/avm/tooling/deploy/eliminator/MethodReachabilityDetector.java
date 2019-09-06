package org.aion.avm.tooling.deploy.eliminator;

import java.util.LinkedList;
import java.util.Map;
import java.util.Queue;
import org.objectweb.asm.Opcodes;

public class MethodReachabilityDetector {

    private final Map<String, ClassInfo> classInfoMap;
    private final Queue<MethodInfo> methodQueue;

    public static Map<String, ClassInfo> getClassInfoMap(String mainClassName, Map<String, byte[]> classMap)
        throws Exception {
        MethodReachabilityDetector detector = new MethodReachabilityDetector(mainClassName, classMap);
        return detector.getClassInfoMap();
    }

    private MethodReachabilityDetector(String mainClassName, Map<String, byte[]> classMap)
        throws Exception {

        // Use the JarDependencyCollector to build the classInfos we need

        classInfoMap = JarDependencyCollector.getClassInfoMap(classMap);

        // Starting with Main::main(), assess reachability
        ClassInfo mainClassInfo = classInfoMap.get(mainClassName);
        if (null == mainClassInfo) {
            throw new Exception("Main class info not found for class " + mainClassName);
        }
        MethodInfo mainMethodInfo = mainClassInfo.getMethodMap().get("main()[B");
        if (null == mainMethodInfo) {
            throw new Exception("Main method info not found!");
        }

        methodQueue = new LinkedList<>();

        mainMethodInfo.isReachable = true;
        methodQueue.add(mainMethodInfo);

        for (ClassInfo classInfo : classInfoMap.values()) {
            methodQueue.addAll(classInfo.getAlwaysReachables());
        }

        traverse();
    }

    private void traverse()
        throws Exception {

        while (!methodQueue.isEmpty()) {
            MethodInfo methodInfo = methodQueue.remove();
            if (!methodInfo.isReachable) {
                throw new Exception("This method should have been marked as reachable!");
            }
            for (MethodInvocation invocation : methodInfo.methodInvocations) {
                ClassInfo ownerClass = classInfoMap.get(invocation.className);
                // if this class isn't in the classInfoMap, it's not part of usercode, so just proceed
                if (null != ownerClass) {
                    MethodInfo calledMethod = ownerClass.getMethodMap()
                        .get(invocation.methodIdentifier);
                    switch (invocation.invocationOpcode) {
                        case Opcodes.INVOKESPECIAL:
                            // this is the easy case: we just mark the methodInfo as reachable and enqueue it
                            enqueue(calledMethod);
                            break;
                        case Opcodes.INVOKEVIRTUAL:
                        case Opcodes.INVOKEDYNAMIC:
                        case Opcodes.INVOKEINTERFACE:
                            // INVOKESTATIC can be inherited even though it's not in the class bytecode
                        case Opcodes.INVOKESTATIC:
                            enqueueSelfAndChildren(ownerClass, invocation.methodIdentifier);
                            break;
                        default:
                            throw new Exception("This is not an invoke method opcode");
                    }
                }
            }
        }
    }

    // should only be called on methods that aren't constructors
    private void enqueueSelfAndChildren(ClassInfo classInfo, String methodId) {

        // Enqueue the declaration of this method
        MethodInfo methodInfo = classInfo.getDeclaration(methodId);
        enqueue(methodInfo);

        if (!methodInfo.isStatic) {
            // For each child, enqueue the concrete implementation that the child uses
            for (ClassInfo childClassInfo : classInfo.getChildren()) {
                MethodInfo childMethodInfo = childClassInfo.getMethodMap().get(methodId);
                // if a child overrides this method, mark that as reachable
                if (null != childMethodInfo) {
                    enqueue(childMethodInfo);
                }
                // if not, we need to mark the concrete implementation as reachable if
                // - we are currently examining an interface
                // - the child we are examining is a non-abstract class
                else if (classInfo.isInterface() && !childClassInfo.isInterface() && !childClassInfo
                    .isAbstract()) {
                    enqueue(childClassInfo.getDeclaration(methodId));
                }
            }
        }
    }

    private void enqueue(MethodInfo methodInfo) {
        if (!methodInfo.isReachable) {
            methodInfo.isReachable = true;
            methodQueue.add(methodInfo);
        }
    }

    private Map<String, ClassInfo> getClassInfoMap() {
        return classInfoMap;
    }
}

