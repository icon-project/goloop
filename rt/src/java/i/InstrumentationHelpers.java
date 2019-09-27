package i;


/**
 * Helpers related to how we attach or otherwise interact with IInstrumentation.
 */
public class InstrumentationHelpers {
    public static void attachThread(IInstrumentation instrumentation) {
        RuntimeAssertionError.assertTrue(null == IInstrumentation.attachedThreadInstrumentation.get());
        IInstrumentation.attachedThreadInstrumentation.set(instrumentation);
    }
    public static void detachThread(IInstrumentation instrumentation) {
        RuntimeAssertionError.assertTrue(instrumentation == IInstrumentation.attachedThreadInstrumentation.get());
        IInstrumentation.attachedThreadInstrumentation.remove();
    }

    public static void pushNewStackFrame(IRuntimeSetup runtimeSetup, ClassLoader contractLoader, long energyLeft, int nextHashCode, InternedClasses classWrappers) {
        // Get the instrumentation for this thread (must be attached).
        IInstrumentation instrumentation = IInstrumentation.attachedThreadInstrumentation.get();
        RuntimeAssertionError.assertTrue(null != instrumentation);
        
        // Tell the instrumentation to create the new frame for the DApp we are entering.
        instrumentation.enterNewFrame(contractLoader, energyLeft, nextHashCode, classWrappers);
        
        // Tell the underlying static instrumentation receiver to attach to this instrumentation.
        runtimeSetup.attach(instrumentation);
    }
    public static void popExistingStackFrame(IRuntimeSetup runtimeSetup) {
        // Get the instrumentation for this thread (must be attached).
        IInstrumentation instrumentation = IInstrumentation.attachedThreadInstrumentation.get();
        RuntimeAssertionError.assertTrue(null != instrumentation);
        
        // Tell the underlying static instrumentation receiver to detach to this instrumentation.
        runtimeSetup.detach(instrumentation);
        
        // Tell the instrumentation to discard this frame and return to the previous (if there was one).
        instrumentation.exitCurrentFrame();
    }

    public static void temporarilyExitFrame(IRuntimeSetup runtimeSetup) {
        // Get the instrumentation for this thread (must be attached).
        IInstrumentation instrumentation = IInstrumentation.attachedThreadInstrumentation.get();
        RuntimeAssertionError.assertTrue(null != instrumentation);
        
        // We want to detach from the underlying DApp so we can re-enter it freshly, later.
        runtimeSetup.detach(instrumentation);
    }
    public static void returnToExecutingFrame(IRuntimeSetup runtimeSetup) {
        // Get the instrumentation for this thread (must be attached).
        IInstrumentation instrumentation = IInstrumentation.attachedThreadInstrumentation.get();
        RuntimeAssertionError.assertTrue(null != instrumentation);
        
        // We want to re-attach to the DApp configured by this IRuntimeSetup.
        runtimeSetup.attach(instrumentation);
    }
}
