package i;


/**
 * The interface required to support the Helper injected class which provides the global callout points from within
 * the instrumented code.
 * NOTE:  This is currently a work-in-progress for issue-308, directly mirroring the refactored implementation from
 * within Helper.
 */
public interface IInstrumentation {
    // The instrumentation instance associated with the given thread and also installed into the Helper of the currently-running DApp.
    ThreadLocal<IInstrumentation> attachedThreadInstrumentation = new ThreadLocal<>();

    static IInstrumentation charge(long cost) throws OutOfEnergyException {
        IInstrumentation ins = attachedThreadInstrumentation.get();
        ins.chargeEnergy(cost);
        return ins;
    }

    static long getEnergyLeft() {
        return attachedThreadInstrumentation.get().energyLeft();
    }

    static FrameContext getCurrentFrameContext() {
        return attachedThreadInstrumentation.get().getFrameContext();
    }

    void enterNewFrame(ClassLoader contractLoader, long energyLeft, int nextHashCode, InternedClasses classWrappers, FrameContext frameContext);
    void exitCurrentFrame();

    <T> s.java.lang.Class<T> wrapAsClass(Class<T> input);
    s.java.lang.String wrapAsString(String input);
    s.java.lang.Object unwrapThrowable(Throwable t);
    Throwable wrapAsThrowable(s.java.lang.Object arg);
    void chargeEnergy(long cost) throws OutOfEnergyException;
    boolean tryChargeEnergy(long cost);
    long energyLeft();

    default void chargeEnergyImmediately(long cost) throws OutOfEnergyException {
        chargeEnergy(cost);
    }

    /**
     * Used to get the next hash code and then increment it.
     * @return The next hash code, prior to the increment.
     */
    int getNextHashCodeAndIncrement();

    int getCurStackSize();
    int getCurStackDepth();
    void enterMethod(int frameSize);
    void exitMethod(int frameSize);
    void enterCatchBlock(int depth, int size);
    
    // Used to read/write hashcode value around internal calls (since we only update the next hash code if the callee succeeded).
    /**
     * Allows read-only access to the next hash code (this will NOT increment it).
     * 
     * @return The next hash code.
     */
    int peekNextHashCode();

    /**
     * Sets the next hash code to the given value.  This is used to update the hash code in a caller frame if a callee succeeds.
     * @param nextHashCode The hash code to use for the next object allocated.
     */
    void forceNextHashCode(int nextHashCode);
    
    void bootstrapOnly();

    /**
     * @return id the class has been loaded by the classloader associated to stackFrame
     */
    boolean isLoadedByCurrentClassLoader(Class<?> userClass);

    FrameContext getFrameContext();
}
