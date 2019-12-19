package i;

import java.lang.invoke.MethodHandle;
import java.lang.invoke.MethodType;
import java.lang.invoke.MethodHandles.Lookup;


/**
 * Utility class containing a few common checks required in invokedynamic bootstrap methods.
 */
public class InvokeDynamicChecks {
    /**
     * Checks that the owner of the callsite is user code.
     * 
     * @param owner The class owning the invokedynamic callsite.
     */
    public static void checkOwner(Lookup owner) {
        RuntimeAssertionError.assertTrue (IInstrumentation.attachedThreadInstrumentation.get().isLoadedByCurrentClassLoader(owner.lookupClass()));
    }

    /**
     * The MethodHandle called for a lambda must only operate on safe types.
     * 
     * @param handle The implementation of the lambda.
     */
    public static void checkMethodHandle(MethodHandle handle) {
        MethodType type = handle.type();
        checkSafeClass(type.returnType());
        for (Class<?> argType : type.parameterArray()) {
            checkSafeClass(argType);
        }
    }

    private static void checkSafeClass(Class<?> type) {
        // All classes must be safe (derived from IObject) or primitive (including void).
        RuntimeAssertionError.assertTrue(
                IObject.class.isAssignableFrom(type)
                || type.isPrimitive()
        );
    }

    /**
     * AKI-130: the bootstrap method cannot take additional arguments, since that could be an attack vector as it would require we generated
     * additional classes, dynamically.
     * 
     * @param invokedType The type description of the bootstrap method.
     */
    public static void checkBootstrapMethodType(MethodType invokedType) {
        // We also should have stripped out any lambda which was taking parameters to the invokedType.
        RuntimeAssertionError.assertTrue(0 == invokedType.parameterCount());
    }
}
