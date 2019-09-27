package org.aion.avm.core.util;

import i.IInstrumentation;
import i.InstrumentationHelpers;
import i.InternedClasses;
import i.OutOfEnergyException;
import i.RuntimeAssertionError;
import s.java.lang.Class;


/**
 * Implements the IInstrumentation interface for tests which need to create runtime objects or otherwise interact with the parts of the system
 * which assume that there is an IInstrumentation installed.
 * It automatically installs itself as the helper and provides utilities to install and remove itself from IInstrumentation.currentContractHelper.
 * Additionally, it provides some common static helpers for common cases of its use.
 */
public class TestingHelper implements IInstrumentation {

    /**
     * A special entry-point used only the test wallet when running the constract, inline.  This allows the helper to be setup for constant initialization.
     * 
     * @param invocation The invocation to run under the helper.
     */
    public static void runUnderBoostrapHelper(Runnable invocation) {
        TestingHelper helper = new TestingHelper(true);
        try {
            invocation.run();
        } finally {
            helper.remove();
        }
    }


    private final boolean isBootstrapOnly;
    private final int constantHashCode;

    private TestingHelper(boolean isBootstrapOnly) {
        this.isBootstrapOnly = isBootstrapOnly;
        // If this is a helper created for bootstrap purposes, use the "placeholder hash code" we rely on for constants.
        // Otherwise, use something else so we know we aren't accidentally being used for constant init.
        this.constantHashCode = isBootstrapOnly ? Integer.MIN_VALUE : -1;
        install();
    }

    private void install() {
        InstrumentationHelpers.attachThread(this);
    }
    private void remove() {
        InstrumentationHelpers.detachThread(this);
    }

    @Override
    public void chargeEnergy(long cost) throws OutOfEnergyException {
        // Free!
    }

    @Override
    public long energyLeft() {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }

    @Override
    public <T> Class<T> wrapAsClass(java.lang.Class<T> input) {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }

    @Override
    public int getNextHashCodeAndIncrement() {
        return this.constantHashCode;
    }

    @Override
    public void bootstrapOnly() {
        if (!this.isBootstrapOnly) {
            throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
        }
    }

    @Override
    public void setAbortState() {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }
    @Override
    public void clearAbortState() {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }
    @Override
    public void enterNewFrame(ClassLoader contractLoader, long energyLeft, int nextHashCode, InternedClasses classWrappers) {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }
    @Override
    public void exitCurrentFrame() {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }
    @Override
    public s.java.lang.String wrapAsString(String input) {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }
    @Override
    public s.java.lang.Object unwrapThrowable(Throwable t) {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }
    @Override
    public Throwable wrapAsThrowable(s.java.lang.Object arg) {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }
    @Override
    public int getCurStackSize() {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }
    @Override
    public int getCurStackDepth() {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }
    @Override
    public void enterMethod(int frameSize) {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }
    @Override
    public void exitMethod(int frameSize) {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }
    @Override
    public void enterCatchBlock(int depth, int size) {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }
    @Override
    public int peekNextHashCode() {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }
    @Override
    public void forceNextHashCode(int nextHashCode) {
        throw RuntimeAssertionError.unreachable("Shouldn't be called in the testing code");
    }
    @Override
    public boolean isLoadedByCurrentClassLoader(java.lang.Class userClass) {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
}
