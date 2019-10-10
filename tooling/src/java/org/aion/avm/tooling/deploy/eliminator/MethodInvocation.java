package org.aion.avm.tooling.deploy.eliminator;

import java.util.Objects;

public class MethodInvocation {
    public final String className;
    public final String methodIdentifier;
    public final int invocationOpcode;

    public MethodInvocation(String className, String methodIdentifier, int invocationOpcode) {
        this.className = className;
        this.methodIdentifier = methodIdentifier;
        this.invocationOpcode = invocationOpcode;
    }

    @Override
    public int hashCode() {
        return Objects.hash(className, methodIdentifier, invocationOpcode);
    }

    @Override
    public boolean equals(Object obj) {
        if (null == obj || !(obj instanceof MethodInvocation)) {
            return false;
        } else {
            MethodInvocation invocation = (MethodInvocation) obj;
            return invocation.methodIdentifier.equals(methodIdentifier)
                && invocation.className.equals(className)
                && invocation.invocationOpcode == invocationOpcode;
        }
    }
}
