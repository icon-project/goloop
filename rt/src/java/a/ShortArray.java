package a;

import i.*;
import java.util.Arrays;

import org.aion.avm.RuntimeMethodFeeSchedule;

public class ShortArray extends Array {

    private short[] underlying;

    /**
     * Static ShortArray factory
     *
     * After instrumentation, NEWARRAY bytecode (with short as type) will be replaced by a INVOKESTATIC to
     * this method.
     *
     * @param size Size of the short array
     *
     * @return New empty short array wrapper
     */
    public static ShortArray initArray(int size){
        chargeEnergy(size * ArrayElement.SHORT.getEnergy());
        return new ShortArray(size);
    }

    @Override
    public int length() {
        lazyLoad();
        return this.underlying.length;
    }

    public short get(int idx) {
        lazyLoad();
        return this.underlying[idx];
    }

    public void set(int idx, short val) {
        lazyLoad();
        this.underlying[idx] = val;
    }

    @Override
    public IObject clone() {
        lazyLoad();
        return new ShortArray(Arrays.copyOf(underlying, underlying.length));
    }

    @Override
    public IObject avm_clone() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ShortArray_avm_clone + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * length());
        lazyLoad();
        return new ShortArray(Arrays.copyOf(underlying, underlying.length));
    }

    //========================================================
    // Internal Helper
    //========================================================

    public ShortArray(int c) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ShortArray_avm_constructor);
        this.underlying = new short[c];
    }

    public ShortArray(short[] underlying) {
        RuntimeAssertionError.assertTrue(null != underlying);
        this.underlying = underlying;
    }

    public short[] getUnderlying() {
        lazyLoad();
        return underlying;
    }

    @Override
    public java.lang.Object getUnderlyingAsObject(){
        lazyLoad();
        return underlying;
    }

    @Override
    public void setUnderlyingAsObject(java.lang.Object u){
        RuntimeAssertionError.assertTrue(null != u);
        lazyLoad();
        this.underlying = (short[]) u;
    }

    @Override
    public java.lang.Object getAsObject(int idx){
        lazyLoad();
        return this.underlying[idx];
    }

    //========================================================
    // Persistent Memory Support
    //========================================================

    public ShortArray(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(ShortArray.class, deserializer);

        int length = deserializer.readInt();
        this.underlying = new short[length];
        for (int i = 0; i < length; ++i) {
            this.underlying[i] = deserializer.readShort();
        }
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(ShortArray.class, serializer);

        serializer.writeInt(this.underlying.length);
        for (int i = 0; i < this.underlying.length; ++i) {
            serializer.writeShort(this.underlying[i]);
        }
    }
}
