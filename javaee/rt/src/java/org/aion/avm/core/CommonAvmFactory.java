package org.aion.avm.core;

import foundation.icon.ee.score.AvmExecutor;
import foundation.icon.ee.score.Loader;
import i.CommonInstrumentation;
import i.IInstrumentation;
import i.IInstrumentationFactory;


/**
 * This is the top-level factory which should be called by embedding kernels and other tooling.
 * Anything below this point should be considered an implementation detail (IInstrumentationFactory, NodeEnvironment, etc).
 */
public class CommonAvmFactory {
    /**
     * Creates an AVM instance based on the given configuration object.
     *
     * @param config The configuration to use when assembling the AVM instance.
     * @param loader Score loader
     * @return An AVM executor
     */
    public static AvmExecutor createAvmExecutor(AvmConfiguration config, Loader loader) {
        // Ensure that NodeEnvironment has been initialized
        NodeEnvironment node = NodeEnvironment.getInstance();
        IInstrumentationFactory factory = new CommonInstrumentationFactory();
        AvmExecutor executor = new AvmExecutor(factory, config, loader);
        return executor;
    }

    private static class CommonInstrumentationFactory implements IInstrumentationFactory {
        @Override
        public IInstrumentation createInstrumentation() {
            return new CommonInstrumentation();
        }
        @Override
        public void destroyInstrumentation(IInstrumentation instance) {
            // Implementation requires no cleanup.
        }
    }
}
