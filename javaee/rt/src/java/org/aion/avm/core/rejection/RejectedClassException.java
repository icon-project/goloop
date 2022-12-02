package org.aion.avm.core.rejection;

import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.PredefinedException;


/**
 * Throw by RejectionVisitor when it detects a violation of one of its rules.
 * This is a RuntimeException since it is thrown from deep within the visitor machinery and we want to catch it at the top-level.
 */
public class RejectedClassException extends PredefinedException {
    private static final long serialVersionUID = 1L;

    public static void unsupportedClassVersion(int version) {
        throw new RejectedClassException("Unsupported class version: " + version);
    }

    public static void blockedOpcode(int opcode) {
        throw new RejectedClassException("Blocked opcode detected: 0x" + Integer.toHexString(opcode));
    }

    public static RejectedClassException notAllowedClass(String className) {
        throw new RejectedClassException("Class is not on allowlist: " + className);
    }

    public static void forbiddenMethodOverride(String methodName) {
        throw new RejectedClassException("Attempted to override forbidden method: " + methodName);
    }

    public static void invalidMethodFlag(String methodName, String flagName) {
        throw new RejectedClassException("Method \"" + methodName + "\" has invalid/forbidden access flag: " + flagName);
    }

    public static void restrictedSuperclass(String className, String superName) {
        throw new RejectedClassException(className + " attempted to subclass restricted class: " + superName);
    }

    public static void jclMethodNotImplemented(String receiver, String name, String descriptor) {
        throw new RejectedClassException("JCL implementation missing method: " + receiver + "#" + name + descriptor);
    }

    public static void nameTooLong(String className) {
        throw new RejectedClassException("Class name is too long: " + className);
    }
    public static void unsupportedPackageName(String className) {
        throw new RejectedClassException("score package name is restricted: " + className);
    }

    public static void arrayDimensionTooBig(String desc) {
        throw new RejectedClassException("Array dimension should not be more than 3: " + desc);
    }

    public static RejectedClassException invokeDynamicBootstrapMethodArguments(String methodDescriptor) {
        throw new RejectedClassException("Unsupported invokedynamic: bootstrap method cannot take additional arguments: \"" + methodDescriptor + "\"");
    }

    public static RejectedClassException invokeDynamicUnsupportedMethodOwner(String origMethodName, String methodOwner) {
        throw new RejectedClassException("Unsupported invokedynamic: bootstrap:" + origMethodName + " owner:" + methodOwner);
    }

    public static RejectedClassException invokeDynamicLambdaType(String methodDescriptor) {
        throw new RejectedClassException("Unsupported invokedynamic lambda type: \"" + methodDescriptor + "\"");
    }

    public static RejectedClassException invokeDynamicHandleType(int handleKind, String methodDescriptor) {
        throw new RejectedClassException("Unsupported invokedynamic method handle: method descriptor: " + methodDescriptor +", reference kind: " + handleKind);
    }

    public static RejectedClassException tooManyInstanceVariables(String className) {
        throw new RejectedClassException("Class exceeds instance variable limit: " + className);
    }

    public static RejectedClassException maximumMethodSizeExceeded(String className) {
        throw new RejectedClassException("Class exceeds maximum method size: " + className);
    }

    public static RejectedClassException maximumExceptionTableEntriesExceeded(String className) {
        throw new RejectedClassException("Class exceeds maximum exception table size for a method: " + className);
    }

    public static RejectedClassException maximumOperandStackDepthExceeded(String className) {
        throw new RejectedClassException("Class exceeds maximum operand stack depth for a method: " + className);
    }

    public static RejectedClassException maximumLocalVariableCountExceeded(String className) {
        throw new RejectedClassException("Class exceeds maximum number of local variables for a method: " + className);
    }

    public static RejectedClassException maximumMethodCountExceeded(String className) {
        throw new RejectedClassException("Class exceeds maximum number of methods: " + className);
    }

    public static RejectedClassException maximumConstantPoolEntriesExceeded(String className) {
        throw new RejectedClassException("Class exceeds maximum number of constant pool entries: " + className);
    }

    public RejectedClassException(String message) {
        super(message);
    }

    public RejectedClassException(String message, Throwable cause) {
        super(message, cause);
    }

    public int getCode() {
        return Status.IllegalFormat;
    }
}
