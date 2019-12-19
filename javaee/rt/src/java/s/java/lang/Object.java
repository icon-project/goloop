package s.java.lang;

import org.aion.avm.ClassNameExtractor;
import i.*;
import org.aion.avm.RuntimeMethodFeeSchedule;


/**
 * The shadow implementation of the {@link java.lang.Object}.
 */
public class Object extends java.lang.Object implements IObject {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }
    public static final int NEW_INSTANCE_READ_INDEX = -1;

    private int hashCode;

    // The readIndex is only used in cases of reentrant calls (this is the serialization index of the instance in the caller frame when serialized for the reentrant call).
    public final int readIndex;

    public Object() {
        this.hashCode = IInstrumentation.attachedThreadInstrumentation.get().getNextHashCodeAndIncrement();
        this.readIndex = NEW_INSTANCE_READ_INDEX;
    }

    // Special constructor only invoked when instantiating through the persistence system (the use of true Void here is just to make sure the user can't define it).
    public Object(java.lang.Void ignore, int readIndex) {
        this.readIndex = readIndex;
    }

    // We created this special allocator for cases where we don't want to change the hashcode (Classes, for example).
    protected Object(java.lang.Void ignore, java.lang.Void ignore2, int hashCode) {
        this.hashCode = hashCode;
        this.readIndex = NEW_INSTANCE_READ_INDEX;
    }

    // Special constructor only invoked when instantiating constants
    protected Object(ConstantToken constantToken) {
        this.hashCode = constantToken.constantId;
        //readIndex will be used to identify constants during serialization. It can also be used to derive constantId
        this.readIndex = ConstantToken.getReadIndexFromConstantId(hashCode);
        ConstantsHolder.addConstant(hashCode, this);
    }

    /**
     * This exists purely for serialization of INode instances for the delta hash.
     * 
     * @return The identity hash of the instance.
     */
    public int getIdentityHashCode() {
        return this.hashCode;
    }

    @Override
    public Class<?> avm_getClass() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Object_avm_getClass);
        return IInstrumentation.attachedThreadInstrumentation.get().wrapAsClass(this.getClass());
    }

    @Override
    public int avm_hashCode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Object_avm_hashCode);
        return internalHashcode();
    }

    @Override
    public boolean avm_equals(IObject obj) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Object_avm_equals);
        // By default, we are only instance-equal.
        return (this == obj);
    }

    protected IObject avm_clone() throws CloneNotSupportedException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Object_avm_clone);
        throw new CloneNotSupportedException();
    }

    @Override
    public String avm_toString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Object_avm_toString);
        //using the public facing method since the user can override this
        return new String(ClassNameExtractor.getOriginalClassName(getClass().getName()) + "@" + java.lang.Integer.toHexString(avm_hashCode()));
    }

    @Override
    public int hashCode() {
        // NOTE:  This is not called in normal operation but is called in cases where we run a contract without transformation.
        return avm_hashCode();
    }

    @Override
    public boolean equals(java.lang.Object obj) {
        // NOTE:  This is not called in normal operation but is called in cases where we run a contract without transformation.
        return (obj instanceof IObject)
                ? avm_equals((IObject)obj)
                : false;
    }

    /**
     * The call which causes this instance to become loaded.
     * Note that this is final since the protected "deserializeSelf" should be over-ridden.
     */
    public final void lazyLoad() {
        // This now does nothing - will be removed later if we are certain we don't want the lazy loading.
        // It was originally how the lazy loading system worked when we had the incremental loading design.
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        // We only operate on our hashCode.
        this.hashCode = deserializer.readInt();
        
        // NOTE:  It would probably be a better design to special-case the handling of the hashCode, in the automatic implementation,
        // since this otherwise means that we have a "real" implementation which we pretend is not "real" so we can automatically
        // deserialize our sub-class.  This should improve performance, though.
        deserializer.automaticallyDeserializeFromRoot((null == firstRealImplementation) ? Object.class : firstRealImplementation, this);
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        // We only operate on our hashCode.
        serializer.writeInt(this.hashCode);
        
        // NOTE:  It would probably be a better design to special-case the handling of the hashCode, in the automatic implementation,
        // since this otherwise means that we have a "real" implementation which we pretend is not "real" so we can automatically
        // serialize our sub-class.  This should improve performance, though.
        serializer.automaticallySerializeToRoot((null == firstRealImplementation) ? Object.class : firstRealImplementation, this);
    }

    public int internalHashcode(){
        lazyLoad();
        return this.hashCode;
    }
}
