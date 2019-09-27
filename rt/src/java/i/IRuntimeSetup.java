package i;


/**
 * This just exists to make it easier to call into the Helper class when mapped into the user class loader (as "H").
 * This means we just ask it for an instance, and we can use that for setup/teardown of its state, instead of carrying
 * around a lot of logic related to reflection.
 */
public interface IRuntimeSetup {
    public void attach(IInstrumentation instrumentation);
    public void detach(IInstrumentation instrumentation);
}
