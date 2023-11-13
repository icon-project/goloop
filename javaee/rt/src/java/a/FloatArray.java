package a;

import i.*;
import java.util.Arrays;

import org.aion.avm.EnergyCalculator;
import org.aion.avm.RuntimeMethodFeeSchedule;

public class FloatArray extends Array {

    private float[] underlying;

    /**
     * Static FloatArray factory
     *
     * After instrumentation, NEWARRAY bytecode (with float as type) will be replaced by a INVOKESTATIC to
     * this method.
     *
     * @param size Size of the float array
     *
     * @return New empty float array wrapper
     */
    public static FloatArray initArray(int size){
        chargeEnergyInitArray(size, ArrayElement.FLOAT.getEnergy());
        return new FloatArray(size);
    }

    @Override
    public int length() {
        lazyLoad();
        return this.underlying.length;
    }

    public float get(int idx) {
        lazyLoad();
        return this.underlying[idx];
    }

    public void set(int idx, float val) {
        lazyLoad();
        this.underlying[idx] = val;
    }

    @Override
    public IObject avm_clone() {
        EnergyCalculator.chargeEnergyClone(RuntimeMethodFeeSchedule.FloatArray_avm_clone,
                length(), ArrayElement.FLOAT.getEnergy());
        lazyLoad();
        return new FloatArray(Arrays.copyOf(underlying, underlying.length));
    }

    @Override
    public IObject clone() {
        lazyLoad();
        return new FloatArray(Arrays.copyOf(underlying, underlying.length));
    }

    //========================================================
    // Internal Helper
    //========================================================

    public FloatArray(int c) {
        EnergyCalculator.chargeEnergy(RuntimeMethodFeeSchedule.FloatArray_avm_constructor);
        this.underlying = new float[c];
    }

    public FloatArray(float[] underlying) {
        RuntimeAssertionError.assertTrue(null != underlying);
        this.underlying = underlying;
    }

    public float[] getUnderlying() {
        lazyLoad();
        return underlying;
    }

    @Override
    public void setUnderlyingAsObject(java.lang.Object u){
        RuntimeAssertionError.assertTrue(null != u);
        lazyLoad();
        this.underlying = (float[]) u;
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

    public FloatArray(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(FloatArray.class, deserializer);

        int length = deserializer.readInt();
        this.underlying = new float[length];
        for (int i = 0; i < length; ++i) {
            this.underlying[i] = deserializer.readFloat();
        }
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(FloatArray.class, serializer);

        serializer.writeInt(this.underlying.length);
        for (int i = 0; i < this.underlying.length; ++i) {
            serializer.writeFloat(this.underlying[i]);
        }
    }
}
