package i;

import java.util.*;

/**
 * CommonInstrumentation operates on the state of the Dapp running in a single thread, only used internally
 * Forces the execution to stop if the abort state is activated
 **/
public class CommonInstrumentation implements IInstrumentation {
    // Single-frame states (the currentFrame cannot also be in the callerFrame - this is just an optimization since the currentFrame access
    // is the common case and is in the critical path - may actually be worth fully-inlining these variables, at some point).
    private FrameState currentFrame;
    private final Stack<FrameState> callerFrames;

    // State which applies to the entire stack.
    private boolean abortState;

    public CommonInstrumentation() {
        this.callerFrames = new Stack<>();
        this.abortState = false;
    }

    public void enterNewFrame(ClassLoader contractLoader, long energyLeft, int nextHashCode, InternedClasses classWrappers) {
        RuntimeAssertionError.assertTrue(null != contractLoader);
        FrameState newFrame = new FrameState();
        newFrame.lateLoader = contractLoader;

        newFrame.energyLeft = energyLeft;
        newFrame.nextHashCode = nextHashCode;

        // Reset our interning state.
        // Note that we want to fail on any attempt to use the interned string map which isn't the initial call (since <clinit> needs it but any
        // other attempt to use it is an error).
        if (1 == nextHashCode) {
            newFrame.internedStringWrappers = new IdentityHashMap<String, s.java.lang.String>();
        }

        newFrame.internedClassWrappers = classWrappers;

        // setting up a default stack watcher.
        newFrame.stackWatcher = new StackWatcher();
        newFrame.stackWatcher.setPolicy(StackWatcher.POLICY_SIZE | StackWatcher.POLICY_DEPTH);
        newFrame.stackWatcher.setMaxStackDepth(512);
        newFrame.stackWatcher.setMaxStackSize(16 * 1024);
        
        // Install the frame.
        if (null != this.currentFrame) {
            this.callerFrames.push(this.currentFrame);
        }
        this.currentFrame = newFrame;
    }

    public void exitCurrentFrame() {
        // Remove the frame, potentially falling back to the caller.
        FrameState returningFrame = null;
        if (!this.callerFrames.isEmpty()) {
            returningFrame = this.callerFrames.pop();
        }
        this.currentFrame = returningFrame;
    }

    @SuppressWarnings("unchecked")
    @Override
    public <T> s.java.lang.Class<T> wrapAsClass(Class<T> input) {
        s.java.lang.Class<T> wrapper = null;
        if (null != input) {
            wrapper = (s.java.lang.Class<T>) this.currentFrame.internedClassWrappers.get(input);
        }
        return wrapper;
    }

    @Override
    public s.java.lang.String wrapAsString(String input) {
        s.java.lang.String wrapper = null;
        if (null != input) {
            wrapper = this.currentFrame.internedStringWrappers.get(input);
            if (null == wrapper) {
                wrapper = new s.java.lang.String(input);
                this.currentFrame.internedStringWrappers.put(input, wrapper);
            }
        }
        return wrapper;
    }

    @Override
    public s.java.lang.Object unwrapThrowable(Throwable t) {
        s.java.lang.Object shadow = null;
        AvmThrowable exceptionToRethrow = null;
        try {
            // NOTE:  This is called for both the cases where the throwable is a VM-generated "java.lang" exception or one of our wrappers.
            // We need to wrap the java.lang instance in a shadow and unwrap the other case to return the shadow.
            String throwableName = t.getClass().getName();
            if (throwableName.startsWith("java.lang.")) {
                // Note that there are 2 cases of VM-generated exceptions:  the kind we wrap for the user and the kind we interpret as a fatal node error.
                if (t instanceof VirtualMachineError) {
                    // This is a fatal node error:
                    // -create our fatal exception
                    JvmError error = new JvmError((VirtualMachineError)t);
                    // -store it in forceExitState
                    this.currentFrame.forceExitState = error;
                    // -throw it
                    throw error;
                }
                // This is VM-generated - we will have to instantiate a shadow, directly.
                shadow = convertVmGeneratedException(t);
            } else if (t instanceof AvmThrowable) {
                // There are cases where an AvmException might appear here during, for example, a finally clause.  We just want to re-throw it
                // since these aren't catchable within the user code.
                exceptionToRethrow = (AvmThrowable)t;
            } else {
                // This is one of our wrappers.
                e.s.java.lang.Throwable wrapper = (e.s.java.lang.Throwable)t;
                shadow = (s.java.lang.Object)wrapper.unwrap();
            }
        } catch (Throwable err) {
            // Unrecoverable internal error.
            throw RuntimeAssertionError.unexpected(err);
        }
        if (null != exceptionToRethrow) {
            throw exceptionToRethrow;
        }
        return shadow;
    }

    @Override
    public Throwable wrapAsThrowable(s.java.lang.Object arg) {
        Throwable result = null;
        try {
            // In this case, we just want to look up the appropriate wrapper (using reflection) and instantiate a wrapper for this.
            String objectClass = arg.getClass().getName();
            // Note that there are currently 2 cases related to the argument:
            // 1) This is an object from our "java/lang" shadows.
            // 2) This is an object defined by the user, and mapped into our "user" package.
            // Determine which case it is to strip off that prefix and apply the common wrapper prefix to look up the class.

            RuntimeAssertionError.assertTrue(isLoadedByCurrentClassLoader(arg.getClass()) || objectClass.startsWith(PackageConstants.kShadowDotPrefix));

            // Note that, since we currently declare the "java.lang." inside the constant for JDK shadows, we need to avoid curring that off.
            String wrapperClassName = PackageConstants.kExceptionWrapperDotPrefix + objectClass;
            Class<?> wrapperClass = this.currentFrame.lateLoader.loadClass(wrapperClassName);
            result = (Throwable)wrapperClass.getConstructor(Object.class).newInstance(arg);
        } catch (Throwable err) {
            // Unrecoverable internal error.
            throw RuntimeAssertionError.unexpected(err);
        } 
        return result;
    }

    @Override
    public void chargeEnergy(long cost) throws OutOfEnergyException {
        // This is called at the beginning of a block so see if we are being asked to exit.
        if (null != this.currentFrame.forceExitState) {
            throw this.currentFrame.forceExitState;
        }

        RuntimeAssertionError.assertTrue(cost >= 0);
        RuntimeAssertionError.assertTrue(cost < Math.pow(2, 30));

        // Bill for the block.
        this.currentFrame.energyLeft -= cost;
        if (this.currentFrame.energyLeft < 0) {
            // Note that this is a reason to force the exit so set this.
            OutOfEnergyException error = new OutOfEnergyException();
            this.currentFrame.forceExitState = error;
            throw error;
        }

        // Check if we are in abort state.
        if (abortState){
            EarlyAbortException error = new EarlyAbortException();
            this.currentFrame.forceExitState = error;
            throw error;
        }
    }

    @Override
    public long energyLeft() {
        return this.currentFrame.energyLeft;
    }

    @Override
    public int getNextHashCodeAndIncrement() {
        // NOTE:  In the case of a Class object, this value is swapped out, temporarily.
        return this.currentFrame.nextHashCode++;
    }

    @Override
    public void setAbortState() {
        abortState = true;
    }

    @Override
    public void clearAbortState() {
        abortState = false;
    }

    @Override
    public int getCurStackSize() {
        return this.currentFrame.stackWatcher.getCurStackSize();
    }

    @Override
    public int getCurStackDepth() {
        return this.currentFrame.stackWatcher.getCurStackDepth();
    }

    @Override
    public void enterMethod(int frameSize) {
        // may be redundant with class metering
        if (null != this.currentFrame.forceExitState) {
            throw this.currentFrame.forceExitState;
        }

        try {
            this.currentFrame.stackWatcher.enterMethod(frameSize);
        } catch (OutOfStackException ex) {
            this.currentFrame.forceExitState = ex;
        }
    }

    @Override
    public void exitMethod(int frameSize) {
        // may be redundant with class metering
        if (null != this.currentFrame.forceExitState) {
            throw this.currentFrame.forceExitState;
        }

        try {
            this.currentFrame.stackWatcher.exitMethod(frameSize);
        } catch (OutOfStackException ex) {
            this.currentFrame.forceExitState = ex;
        }
    }

    @Override
    public void enterCatchBlock(int depth, int size) {
        // may be redundant with class metering
        if (null != this.currentFrame.forceExitState) {
            throw this.currentFrame.forceExitState;
        }

        try {
            this.currentFrame.stackWatcher.enterCatchBlock(depth, size);
        } catch (OutOfStackException ex) {
            this.currentFrame.forceExitState = ex;
        }
    }

    @Override
    public int peekNextHashCode() {
        return this.currentFrame.nextHashCode;
    }
    @Override
    public void forceNextHashCode(int nextHashCode) {
        this.currentFrame.nextHashCode = nextHashCode;
    }

    @Override
    public void bootstrapOnly() {
        throw RuntimeAssertionError.unreachable("NOT a bootstrap IInstrumentation");
    }

    @Override
    public boolean isLoadedByCurrentClassLoader(Class userClass) {
        // If this is the same classloader, they will both be obviously the same instance.
        return (userClass.getClassLoader() == this.currentFrame.lateLoader);
    }

    // Private helpers used internally.
    private s.java.lang.Throwable convertVmGeneratedException(Throwable t) throws Exception {
        // First step is to convert the message and cause into shadow objects, as well.
        String originalMessage = t.getMessage();
        s.java.lang.String message = (null != originalMessage)
                ? new s.java.lang.String(originalMessage)
                : null;
        // (note that converting the cause is recusrive on the causal chain)
        Throwable originalCause = t.getCause();
        s.java.lang.Throwable cause = (null != originalCause)
                ? convertVmGeneratedException(originalCause)
                : null;
        
        // Then, use reflection to find the appropriate wrapper.
        String throwableName = t.getClass().getName();
        Class<?> shadowClass = this.currentFrame.lateLoader.loadClass(PackageConstants.kShadowDotPrefix + throwableName);
        return (s.java.lang.Throwable)shadowClass.getConstructor(s.java.lang.String.class, s.java.lang.Throwable.class).newInstance(message, cause);
    }


    /**
     * The CommonInstrumentation contains the logic to operate on state and the state which is shared for the entire stack of DApp
     * invocations running in a single thread.
     * This class contains the state specific to a single frame of this invocation path (that is, a single DApp invocation).
     * NOTE:  public ONLY for tests.
     */
    public static class FrameState {
        public StackWatcher stackWatcher;

        private ClassLoader lateLoader;
        private long energyLeft;
        private int nextHashCode;

        /**
         * Note that we need to consider instance equality for strings and classes:
         * -String instance quality isn't normally important but some cases, such as constant identifiers, are sometimes expected to be instance-equal.
         *  In our implementation, we are only going to preserve this for the <clinit> methods of the contract classes and, other than that, actively
         *  avoid any observable instance equality beyond instance preservation in the object graph (no relying on the same Class instance giving the
         *  same String instance back on successive calls, for example).
         * -Class instance equality is generally more important since classes don't otherwise have a clear definition of "equality"
         * Therefore, we will only create a map for interning strings if we suspect that this is the first call (a 1 nextHashCode - we may make this
         * explicit, in the future) but we will always create the map for interning classes.
         * The persistence layer also knows that classes are encoded differently so it will correctly resolve instance through this interning map.
         */
        private IdentityHashMap<String, s.java.lang.String> internedStringWrappers;
        private InternedClasses internedClassWrappers;

        // Set forceExitState to non-null to re-throw at the entry to every block (forces the contract to exit).
        private AvmThrowable forceExitState;
    }
}
