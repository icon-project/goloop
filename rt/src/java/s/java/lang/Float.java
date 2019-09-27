package s.java.lang;

import i.*;
import org.aion.avm.RuntimeMethodFeeSchedule;

public final class Float extends Number implements Comparable<Float> {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public static final float avm_POSITIVE_INFINITY = java.lang.Float.POSITIVE_INFINITY;

    public static final float avm_NEGATIVE_INFINITY = java.lang.Float.NEGATIVE_INFINITY;

    public static final float avm_NaN = java.lang.Float.NaN;

    public static final float avm_MAX_VALUE = java.lang.Float.MAX_VALUE;

    public static final float avm_MIN_NORMAL = java.lang.Float.MIN_NORMAL;

    public static final float avm_MIN_VALUE = java.lang.Float.MIN_VALUE;

    public static final int avm_MAX_EXPONENT = java.lang.Float.MAX_EXPONENT;

    public static final int avm_MIN_EXPONENT = java.lang.Float.MIN_EXPONENT;

    public static final int avm_SIZE = java.lang.Float.SIZE;

    public static final int avm_BYTES = java.lang.Float.BYTES;

    public static final Class<Float> avm_TYPE = new Class(java.lang.Float.TYPE, new ConstantToken(ShadowClassConstantId.Float_avm_TYPE));

    // These are the constructors provided in the JDK but we mark them private since they are deprecated.
    // (in the future, we may change these to not exist - depends on the kind of error we want to give the user).
    private Float(float f) {
        this.v = f;
    }
    @SuppressWarnings("unused")
    private Float(double f) {
        throw RuntimeAssertionError.unimplemented("This is only provided for a consistent error to user code - not to be called");
    }
    @SuppressWarnings("unused")
    private Float(String f) {
        throw RuntimeAssertionError.unimplemented("This is only provided for a consistent error to user code - not to be called");
    }

    public static String avm_toString(float f){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_toString);
        return new String(java.lang.Float.toString(f));
    }

    public static String avm_toHexString(float a){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_toHexString);
        return new String(java.lang.Float.toHexString(a));
    }

    public static Float avm_valueOf(String s) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_valueOf);
        return new Float(internalParseFloat(s));
    }

    public static Float avm_valueOf(float f) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_valueOf_1);
        return new Float(f);
    }

    public static float avm_parseFloat(String s) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_parseFloat);
        return internalParseFloat(s);
    }

    public static boolean avm_isNaN(float v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_isNaN);
        return (v != v);
    }

    public static boolean avm_isInfinite(float v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_isInfinite);
        return internalIsInfinite(v);
    }

    public static boolean avm_isFinite(float f) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_isFinite);
        return java.lang.Float.isFinite(f);
    }

    public boolean avm_isNaN() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_isNaN_1);
        lazyLoad();
        return java.lang.Float.isNaN(this.v);
    }

    public boolean avm_isInfinite() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_isInfinite_1);
        lazyLoad();
        return internalIsInfinite(v);
    }

    public String avm_toString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_toString_1);
        lazyLoad();
        return new String(java.lang.Float.toString(this.v));
    }

    public byte avm_byteValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_byteValue);
        lazyLoad();
        return (byte) v;
    }

    public short avm_shortValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_shortValue);
        lazyLoad();
        return (short) v;
    }

    public int avm_intValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_intValue);
        lazyLoad();
        return (int) v;
    }

    public long avm_longValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_longValue);
        lazyLoad();
        return (long) v;
    }

    public float avm_floatValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_floatValue);
        lazyLoad();
        return v;
    }

    public double avm_doubleValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_doubleValue);
        lazyLoad();
        return (double) v;
    }

    public int avm_hashCode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_hashCode);
        lazyLoad();
        return internalHashCode(v);
    }

    public static int avm_hashCode(float value) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_hashCode_1);
        return internalHashCode(value);
    }

    public boolean avm_equals(IObject obj) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_equals);
        boolean isEqual = false;
        if (obj instanceof Float) {
            Float other = (Float) obj;
            lazyLoad();
            other.lazyLoad();
            isEqual = java.lang.Float.floatToIntBits(this.v) == java.lang.Float.floatToIntBits(other.v);
        }
        return isEqual;
    }

    public static int avm_floatToIntBits(float value) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_floatToIntBits);
        return internalFloatToIntBits(value);
    }

    public static float avm_intBitsToFloat(int bits){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_intBitsToFloat);
        return java.lang.Float.intBitsToFloat(bits);
    }

    public int avm_compareTo(Float anotherFloat) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_compareTo);
        lazyLoad();
        anotherFloat.lazyLoad();
        return java.lang.Float.compare(this.v, anotherFloat.v);
    }

    public static int avm_compare(float f1, float f2) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_compare);
        return java.lang.Float.compare(f1, f2);
    }

    public static float avm_sum(float a, float b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_sum);
        return a + b;
    }

    public static float avm_max(float a, float b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_max);
        return java.lang.Math.max(a, b);
    }

    public static float avm_min(float a, float b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Float_avm_min);
        return java.lang.Math.min(a, b);
    }

    private static float internalParseFloat(String s) throws NumberFormatException {
        return java.lang.Float.parseFloat(s.getUnderlying());
    }

    private static boolean internalIsInfinite(float v) {
        return (v == avm_POSITIVE_INFINITY) || (v == avm_NEGATIVE_INFINITY);
    }

    private static int internalHashCode(float value) {
        return internalFloatToIntBits(value);
    }

    private static int internalFloatToIntBits(float value) {
        return java.lang.Float.floatToIntBits(value);
    }


    //========================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    public Float(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    private float v;

    public float getUnderlying() {
        lazyLoad();
        return this.v;
    }

    //========================================================
    // Methods below are excluded from shadowing
    //========================================================


}
