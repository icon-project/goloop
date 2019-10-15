package org.aion.avm.core;

import foundation.icon.ee.score.AvmExecutor;
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
     * @param capabilities The external capabilities which this AVM instance can use.
     * @param configuration The configuration to use when assembling the AVM instance.
     * @return An AVM instance.
     */
    public static AvmImpl buildAvmInstanceForConfiguration(IExternalCapabilities capabilities, AvmConfiguration configuration) {
        IInstrumentationFactory factory = new CommonInstrumentationFactory();
        return NodeEnvironment.singleton.buildAvmInstance(factory, capabilities, configuration);
    }

    public static AvmExecutor getAvmInstance(IExternalCapabilities capabilities, AvmConfiguration config) {
        // Ensure that NodeEnvironment has been initialized
        NodeEnvironment node = NodeEnvironment.getInstance();
        IInstrumentationFactory factory = new CommonInstrumentationFactory();
        AvmExecutor executor = new AvmExecutor(factory, capabilities, config);
        executor.start();
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
