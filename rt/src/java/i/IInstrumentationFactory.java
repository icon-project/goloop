package i;


/**
 * The abstract description of something which can generate IInstrumentation instances.
 * This is required since the AVM owns the actual threads it uses, meaning it needs to call out to create/destroy the IInstrumentation
 * instances it wants to attach to these threads.
 * Note that the interface is symmetric in case the implementation requires cleanup.  The "destroy" call is made after the instance
 * has been detached from its thread.
 */
public interface IInstrumentationFactory {
    public IInstrumentation createInstrumentation();
    public void destroyInstrumentation(IInstrumentation instance);
}
