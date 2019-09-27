package s.java.math;

import i.*;
import s.java.io.Serializable;
import s.java.lang.Object;
import s.java.lang.String;

import org.aion.avm.RuntimeMethodFeeSchedule;

public final class MathContext extends Object implements Serializable {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public static final MathContext avm_UNLIMITED =
            new MathContext(0, RoundingMode.avm_HALF_UP, new ConstantToken(ShadowClassConstantId.MathContext_avm_UNLIMITED));

    public static final MathContext avm_DECIMAL32 =
            new MathContext(7, RoundingMode.avm_HALF_EVEN, new ConstantToken(ShadowClassConstantId.MathContext_avm_DECIMAL32));

    public static final MathContext avm_DECIMAL64 =
            new MathContext(16, RoundingMode.avm_HALF_EVEN, new ConstantToken(ShadowClassConstantId.MathContext_avm_DECIMAL64));

    public static final MathContext avm_DECIMAL128 =
            new MathContext(34, RoundingMode.avm_HALF_EVEN, new ConstantToken(ShadowClassConstantId.MathContext_avm_DECIMAL128));

    public MathContext(int setPrecision) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.MathContext_avm_constructor);
        v = new java.math.MathContext(setPrecision);
    }

    public MathContext(int setPrecision,
                       RoundingMode setRoundingMode) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.MathContext_avm_constructor_1);
        v = new java.math.MathContext(setPrecision, setRoundingMode.getUnderlying());
    }

    public MathContext(String val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.MathContext_avm_constructor_2);
        v = new java.math.MathContext(val.getUnderlying());
    }

    private MathContext(int setPrecision, RoundingMode setRoundingMode, ConstantToken constantToken) {
        super(constantToken);
        v = new java.math.MathContext(setPrecision, setRoundingMode.getUnderlying());
    }

    public int avm_getPrecision() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.MathContext_avm_getPrecision);
        lazyLoad();
        return this.v.getPrecision();
    }

    public RoundingMode avm_getRoundingMode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.MathContext_avm_getRoundingMode);
        lazyLoad();
        return RoundingMode.internalValueOf(new String(this.v.getRoundingMode().name()));
    }

    public boolean avm_equals(IObject x){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.MathContext_avm_equals);
        boolean isEqual = false;
        if (x instanceof MathContext) {
            MathContext other = (MathContext) x;
            lazyLoad();
            other.lazyLoad();
            isEqual = this.v.equals(other.v);
        }
        return isEqual;
    }

    public int avm_hashCode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.MathContext_avm_hashCode);
        lazyLoad();
        RoundingMode roundingMode = RoundingMode.internalValueOf(new String(this.v.getRoundingMode().name()));
        return this.v.getPrecision() + roundingMode.internalHashcode() * 59;
    }

    public String avm_toString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.MathContext_avm_toString);
        lazyLoad();
        return new String(v.toString());
    }

    //========================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    private java.math.MathContext v;

    public java.math.MathContext getUnderlying() {
        return v;
    }

    // Deserializer support.
    public MathContext(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(MathContext.class, deserializer);
        
        // We store this as the precision (int) and the RoundingMode (stub).
        int precision = deserializer.readInt();
        RoundingMode mode = (RoundingMode)deserializer.readObject();
        // Note that this will be null in our pre-pass, so check that.
        if (null != mode) {
            this.v = new java.math.MathContext(precision, java.math.RoundingMode.valueOf(mode.getName().getUnderlying()));
        }
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(MathContext.class, serializer);
        
        // We store this as the precision (int) and the RoundingMode (stub).
        serializer.writeInt(this.v.getPrecision());
        RoundingMode roundingMode = RoundingMode.internalValueOf(new String(this.v.getRoundingMode().name()));
        serializer.writeObject(roundingMode);
    }

    //========================================================
    // Methods below are excluded from shadowing
    //========================================================

    //private void readObject(java.io.ObjectInputStream s)

}
