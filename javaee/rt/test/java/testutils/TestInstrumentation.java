package testutils;

import i.FrameContext;
import i.IInstrumentation;
import i.InternedClasses;
import i.OutOfEnergyException;
import i.RuntimeAssertionError;
import s.java.lang.Class;
import s.java.lang.Object;


/**
 * A very restricted implementation of IInstrumentation which really only supports creating instances (all with hashcode 1).
 */
public class TestInstrumentation implements IInstrumentation {
    @Override
    public void enterNewFrame(ClassLoader contractLoader, long energyLeft, int nextHashCode, InternedClasses classWrappers, FrameContext ctx) {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public void exitCurrentFrame() {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public <T> Class<T> wrapAsClass(java.lang.Class<T> input) {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public s.java.lang.String wrapAsString(String input) {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public Object unwrapThrowable(Throwable t) {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public Throwable wrapAsThrowable(Object arg) {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public void chargeEnergy(long cost) throws OutOfEnergyException {
    }
    @Override
    public boolean tryChargeEnergy(long cost) {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public long energyLeft() {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public int getNextHashCodeAndIncrement() {
        // Just return a non-zero constant.
        return 1;
    }
    @Override
    public int getCurStackSize() {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public int getCurStackDepth() {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public void enterMethod(int frameSize) {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public void exitMethod(int frameSize) {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public void enterCatchBlock(int depth, int size) {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public int peekNextHashCode() {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public void forceNextHashCode(int nextHashCode) {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
    @Override
    public void bootstrapOnly() {
        // These tests aren't using the NodeEnvironment to bootstrap the JCL so we need to handle that case.
    }

    @Override
    public boolean isLoadedByCurrentClassLoader(java.lang.Class<?> userClass) {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }

    public FrameContext getFrameContext() {
        throw RuntimeAssertionError.unreachable("Not expected in this test");
    }
}
