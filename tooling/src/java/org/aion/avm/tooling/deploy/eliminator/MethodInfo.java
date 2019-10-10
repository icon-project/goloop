package org.aion.avm.tooling.deploy.eliminator;

import java.util.HashSet;
import java.util.Set;

public class MethodInfo {

    public final String methodIdentifier;
    public final boolean isStatic;
    public final Set<MethodInvocation> methodInvocations;
    public boolean isReachable = false;

    public MethodInfo(String methodIdentifier, boolean isStatic) {
        this.methodIdentifier = methodIdentifier;
        this.isStatic = isStatic;
        this.methodInvocations = new HashSet<>();
    }

    public MethodInfo(String methodIdentifier, boolean isStatic, Set<MethodInvocation> methodInvocations) {
        this.methodIdentifier = methodIdentifier;
        this.isStatic = isStatic;
        this.methodInvocations = methodInvocations;
    }
}
