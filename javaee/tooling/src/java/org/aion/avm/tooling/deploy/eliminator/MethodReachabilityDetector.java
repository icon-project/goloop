package org.aion.avm.tooling.deploy.eliminator;

import foundation.icon.ee.struct.Member;
import i.PackageConstants;
import org.objectweb.asm.Opcodes;

import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.Queue;

public class MethodReachabilityDetector {

    private final Map<String, ClassInfo> classInfoMap;
    private final Queue<MethodInfo> methodQueue;

    public static Map<String, ClassInfo> getClassInfoMap(String mainClassName, Map<String, byte[]> classMap, Map<String, List<Member>> keptMethods)
            throws Exception {
        MethodReachabilityDetector detector = new MethodReachabilityDetector(mainClassName, classMap, keptMethods);
        return detector.getClassInfoMap();
    }

    private MethodReachabilityDetector(String mainClassName, Map<String, byte[]> classMap, Map<String, List<Member>> keptMethods)
            throws Exception {
        // Use the JarDependencyCollector to build the classInfos we need
        classInfoMap = JarDependencyCollector.getClassInfoMap(classMap);

        // Starting with Main::main(), assess reachability
        ClassInfo mainClassInfo = classInfoMap.get(mainClassName);
        if (null == mainClassInfo) {
            throw new Exception("Main class info not found for class " + mainClassName);
        }

        methodQueue = new LinkedList<>();
        for (var e : keptMethods.entrySet()) {
            var ci = classInfoMap.get(e.getKey());
            if (ci == null) {
                continue;
            }
            for (var m : e.getValue()) {
                var cci = ci;
                MethodInfo mi = null;
                while (cci != null) {
                    mi = cci.getMethodMap().get(m.getMethodID());
                    if (mi != null) {
                        break;
                    }
                    cci = cci.getSuperclass();
                }
                assert mi != null;
                mi.isReachable = true;
                methodQueue.add(mi);
            }
        }

        for (ClassInfo classInfo : classInfoMap.values()) {
            methodQueue.addAll(classInfo.getAlwaysReachables());
        }

        traverse();
    }

    private void traverse() throws Exception {

        while (!methodQueue.isEmpty()) {
            MethodInfo methodInfo = methodQueue.remove();
            if (!methodInfo.isReachable) {
                throw new Exception("This method should have been marked as reachable!");
            }
            for (MethodInvocation invocation : methodInfo.methodInvocations) {
                ClassInfo ownerClass = classInfoMap.get(invocation.className);
                if (null != ownerClass) {
                    MethodInfo calledMethod = ownerClass.getMethodMap().get(invocation.methodIdentifier);
                    if (ownerClass.isSystemClass()) {
                        if (null == calledMethod) {
                            throw new UnsupportedOperationException(
                                    "Unsupported JCL method detected: " + invocation.className + "#" + invocation.methodIdentifier);
                        }
                    }
                    switch (invocation.invocationOpcode) {
                        case Opcodes.INVOKESPECIAL:
                            calledMethod = ownerClass.getConcreteImplementation(invocation.methodIdentifier);
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
                } else if (!canAccessClass(invocation.className)) {
                    throw new UnsupportedOperationException(
                            "Unsupported JCL class detected: " + invocation.className);
                }
            }
        }
    }

    private boolean canAccessClass(String className) {
        return className.startsWith(PackageConstants.kPublicApiSlashPrefix)
                || className.startsWith("[");
    }

    /* The logic about what should be marked reachable when we've tried to invoke method M in structure S (which can be a class or an interface)
     is a two-step process.

    Step 1) First, we try to find M in S. If it's present, we mark it reachable. If not, we walk up all of S's super classes
    up to Object, looking for M. If we find it, we enqueue it.
    If we still haven't found it, we can say that
        a) S must be an interface or an abstract class, and
        b) M must be declared in one of the interfaces that S implements / extends.
    We hence search through the parent interfaces to find a declaration of M.

    Step 2) We now need to think about S's children only if M is NOT static. For each child C,
        a) If C implements M, we must immediately mark M as reachable in C.
        b) We also have to consider the odd pattern that arises when S is an interface, and method M is implemented
            by an abstract class A, that concrete class C extends. Note that A does NOT implement S in this case.
            In this case, we walk up C's super classes searching for the concrete implementation that it uses, and mark that as reachable.

            Note that we only have to do this if S is an interface. If S is an abstract class, we will mark the appropriate
            concrete implementation as reachable in step 2 a.
     */

    // should only be called on methods that aren't constructors
    private void enqueueSelfAndChildren(ClassInfo classInfo, String methodId) {

        // Enqueue the declaration of this method
        MethodInfo methodInfo = classInfo.getDeclaration(methodId);
        if (null == methodInfo) {
            throw new UnsupportedOperationException("No declaration found for " + methodId + ", corrupt jar suspected");
        } else {
            enqueue(methodInfo);
        }

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
                else if (classInfo.isInterface() && !childClassInfo.isInterface() && !childClassInfo.isAbstract()) {
                    MethodInfo concreteImplInfo = childClassInfo.getConcreteImplementation(methodId);
                    if (null == concreteImplInfo) {
                        throw new IllegalArgumentException("No implementation found for " + methodId + ", corrupt jar suspected");
                    }
                    enqueue(concreteImplInfo);
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
