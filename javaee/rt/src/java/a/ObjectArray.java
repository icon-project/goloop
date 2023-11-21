package a;

import i.*;
import java.util.Arrays;

import org.aion.avm.EnergyCalculator;
import org.aion.avm.RuntimeMethodFeeSchedule;


/**
 * Note that the IObjectArray interface exists here to provide type unification capabilities between the Class[] objects and Interface[] objects
 * where Class implements Interface.  They need to unify in an interface, and be usable, but the actual implementation is always ObjectArray.
 * In the future, all generated intermediary array interfaces also need to implement IObjectArray.  Much like IObject, in the class space,
 * IObjectArray is the "top" of the array space.
 */
public class ObjectArray extends Array implements IObjectArray {

    protected Object[] underlying;

    /**
     * Static ObjectArray factory
     *
     * After instrumentation, NEWARRAY bytecode (with reference as type) will be replaced by a INVOKESTATIC to
     * this method.
     *
     * @param size Size of the object array
     *
     * @return New empty object array wrapper
     */
    public static ObjectArray initArray(int size){
        chargeEnergyInitArray(size, ArrayElement.REF.getEnergy());
        return new ObjectArray(size);
    }

    @Override
    public int length() {
        lazyLoad();
        return this.underlying.length;
    }

    public Object get(int idx) {
        lazyLoad();
        return this.underlying[idx];
    }

    public void set(int idx, Object val) {
        lazyLoad();
        this.underlying[idx] = val;
    }

    @Override
    public IObject avm_clone() {
        EnergyCalculator.chargeEnergyClone(RuntimeMethodFeeSchedule.ObjectArray_avm_clone,
                length(), ArrayElement.REF.getEnergy());
        lazyLoad();
        return new ObjectArray(Arrays.copyOf(underlying, underlying.length));
    }

    public static ObjectArray newWithCharge(Object[] src) {
        EnergyCalculator.chargeEnergyMultiply(RuntimeMethodFeeSchedule.ObjectArray_avm_constructor,
                src.length, ArrayElement.REF.getEnergy());
        return new ObjectArray(src);
    }

    @Override
    public IObject clone() {
        lazyLoad();
        return new ObjectArray(Arrays.copyOf(underlying, underlying.length));
    }

    //========================================================
    // Internal Helper
    //========================================================

    public ObjectArray(int c) {
        EnergyCalculator.chargeEnergy(RuntimeMethodFeeSchedule.ObjectArray_avm_constructor);
        this.underlying = new Object[c];
    }

    public ObjectArray() {
        EnergyCalculator.chargeEnergy(RuntimeMethodFeeSchedule.ObjectArray_avm_constructor_1);
    }

    public ObjectArray(Object[] underlying) {
        RuntimeAssertionError.assertTrue(null != underlying);
        this.underlying = underlying;
    }

    public Object[] getUnderlying() {
        lazyLoad();
        return underlying;
    }

    @Override
    public void setUnderlyingAsObject(java.lang.Object u){
        RuntimeAssertionError.assertTrue(null != u);
        lazyLoad();
        this.underlying = (Object[]) u;
    }

    @Override
    public java.lang.Object getUnderlyingAsObject(){
        lazyLoad();
        return underlying;
    }

    @Override
    public java.lang.Object getAsObject(int idx){
        lazyLoad();
        return this.underlying[idx];
    }

    //========================================================
    // Persistent Memory Support
    //========================================================

    public ObjectArray(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(ObjectArray.class, deserializer);

        int length = deserializer.readInt();
        this.underlying = new Object[length];
        for (int i = 0; i < length; ++i) {
            this.underlying[i] = deserializer.readObject();
        }
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(ObjectArray.class, serializer);

        serializer.writeInt(this.underlying.length);
        for (int i = 0; i < this.underlying.length; ++i) {
            serializer.writeObject((s.java.lang.Object)this.underlying[i]);
        }
    }
}
