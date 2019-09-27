package s.java.lang.invoke;

import java.lang.invoke.ConstantCallSite;
import java.lang.invoke.LambdaConversionException;

import i.FunctionFactory;
import i.IInstrumentation;
import i.InvokeDynamicChecks;
import i.RunnableFactory;
import i.RuntimeAssertionError;


public final class LambdaMetafactory extends s.java.lang.Object {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public static java.lang.invoke.CallSite avm_metafactory(java.lang.invoke.MethodHandles.Lookup owner,
                                                            java.lang.String invokedName,
                                                            java.lang.invoke.MethodType invokedType,
                                                            java.lang.invoke.MethodType samMethodType,
                                                            java.lang.invoke.MethodHandle implMethod,
                                                            java.lang.invoke.MethodType instantiatedMethodType)
            throws LambdaConversionException {
        InvokeDynamicChecks.checkOwner(owner);
        // We don't expect any uses of this to be able to exist without the "avm_" prefix.
        RuntimeAssertionError.assertTrue(invokedName.startsWith("avm_"));
        InvokeDynamicChecks.checkBootstrapMethodType(invokedType);
        InvokeDynamicChecks.checkMethodHandle(implMethod);
        
        // We directly interpret the Runnable and Function, but everything else is invalid and should have been rejected, earlier.
        Class<?> returnType = invokedType.returnType();
        java.lang.invoke.CallSite callSite = null;
        if (s.java.lang.Runnable.class == returnType) {
            // Create the Runnable factory.
            RunnableFactory factory = new RunnableFactory(owner, implMethod);
            // Create a callsite for the factory's instantiate method so each evaluation of the invokedynamic will get a unique instance.
            java.lang.invoke.MethodHandle target = null;
            try {
                target = java.lang.invoke.MethodHandles.lookup()
                        .findVirtual(RunnableFactory.class, "instantiate", invokedType)
                        .bindTo(factory);
            } catch (NoSuchMethodException | IllegalAccessException e) {
                // This would be a static error, internally.
                throw RuntimeAssertionError.unexpected(e);
            }
            callSite = new ConstantCallSite(target);
        } else if (s.java.util.function.Function.class == returnType) {
            // Create the Function factory.
            FunctionFactory factory = new FunctionFactory(owner, implMethod);
            // Create a callsite for the factory's instantiate method so each evaluation of the invokedynamic will get a unique instance.
            java.lang.invoke.MethodHandle target = null;
            try {
                target = java.lang.invoke.MethodHandles.lookup()
                        .findVirtual(FunctionFactory.class, "instantiate", invokedType)
                        .bindTo(factory);
            } catch (NoSuchMethodException | IllegalAccessException e) {
                // This would be a static error, internally.
                throw RuntimeAssertionError.unexpected(e);
            }
            callSite = new ConstantCallSite(target);
        } else {
            throw RuntimeAssertionError.unreachable("Invalid invokeType in LambdaMetaFactory (return type: " + returnType + ")");
        }
        
        return callSite;
    }

    // Cannot be instantiated.
    private LambdaMetafactory() {}
    // Note:  No instances can be created so no deserialization constructor required.
}
