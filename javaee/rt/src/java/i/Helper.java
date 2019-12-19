package i;


/**
 * The common instrumentation support class.  Our instrumentation changes to the user's code assume that they can statically call these functions
 * on a class named RUNTIME_HELPER_NAME so this is installed to address that need.
 * However, this is merely a callout point to the real implementation, accessed via the "target" static variable.
 * Note that there is a copy of this class loaded into every DApp class loader.
 * Each Helper class can only be attached to the instrumentation of a specific thread at any point.  To maintain the simplicity of this design,
 * each thread also can only expose its instrumentation to a single Helper at any point.  This maintains a sort of symmetry where entering a DApp,
 * either for the first time, or reentrantly, always has the same assumption:  it is not currently attached to anyone.
 */
public class Helper implements IRuntimeSetup {
    public static final String RUNTIME_HELPER_NAME = "H";

    private static IInstrumentation target;


    public static <T> s.java.lang.Class<T> wrapAsClass(Class<T> input) {
        return target.wrapAsClass(input);
    }

    /**
     * Note:  This is called by instrumented <clinit> methods to intern String constants defined in the contract code.
     * It should not be called anywhere else.
     * 
     * @param input The original String constant.
     * @return The interned shadow String wrapper.
     */
    public static s.java.lang.String wrapAsString(String input) {
        return target.wrapAsString(input);
    }

    public static s.java.lang.Object unwrapThrowable(Throwable t) {
        return target.unwrapThrowable(t);
    }

    public static Throwable wrapAsThrowable(s.java.lang.Object arg) {
        return target.wrapAsThrowable(arg);
    }

    public static void chargeEnergy(int cost) throws OutOfEnergyException {
        target.chargeEnergy(cost);
    }

    public static int getCurStackSize(){
        return target.getCurStackSize();
    }

    public static int getCurStackDepth(){
        return target.getCurStackDepth();
    }

    public static void enterMethod(int frameSize) {
        target.enterMethod(frameSize);
    }

    public static void exitMethod(int frameSize) {
        target.exitMethod(frameSize);
    }

    public static void enterCatchBlock(int depth, int size) {
        target.enterCatchBlock(depth, size);
    }

    @Override
    public void attach(IInstrumentation instrumentation) {
        RuntimeAssertionError.assertTrue(null == target);
        target = instrumentation;
    }
    @Override
    public void detach(IInstrumentation instrumentation) {
        RuntimeAssertionError.assertTrue(instrumentation == target);
        target = null;
    }
}
