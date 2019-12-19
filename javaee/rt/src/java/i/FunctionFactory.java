package i;

import java.lang.invoke.MethodHandle;
import java.lang.invoke.MethodHandles;

import p.avm.InternalFunction;


public final class FunctionFactory {
    private final MethodHandles.Lookup lookup;
    private final MethodHandle target;

    public FunctionFactory(MethodHandles.Lookup lookup, MethodHandle target) {
        this.lookup = lookup;
        this.target = target;
    }

    public s.java.util.function.Function instantiate() {
        return InternalFunction.createFunction(this.lookup, this.target);
    }
}
