package s.java.lang;

import i.*;
import org.aion.avm.RuntimeMethodFeeSchedule;


public final class Double extends Number implements Comparable<Double>{
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    // These are the constructors provided in the JDK but we mark them private since they are deprecated.
    // (in the future, we may change these to not exist - depends on the kind of error we want to give the user).
    private Double(double d){
        this.v = d;
    }
    @SuppressWarnings("unused")
    private Double(String s){
        throw RuntimeAssertionError.unimplemented("This is only provided for a consistent error to user code - not to be called");
    }

    public static final int avm_BYTES = java.lang.Double.BYTES;

    public static final int avm_MAX_EXPONENT = java.lang.Double.MAX_EXPONENT;

    public static final int avm_MIN_EXPONENT = java.lang.Double.MIN_EXPONENT;

    public static final double avm_MAX_VALUE = java.lang.Double.MAX_VALUE;

    public static final double avm_MIN_VALUE = java.lang.Double.MIN_VALUE;

    public static final double avm_MIN_NORMAL = java.lang.Double.MIN_NORMAL;

    public static final double avm_POSITIVE_INFINITY = java.lang.Double.POSITIVE_INFINITY;

    public static final double avm_NEGATIVE_INFINITY = java.lang.Double.NEGATIVE_INFINITY;

    public static final double avm_NaN = java.lang.Double.NaN;

    public static final int avm_SIZE = java.lang.Double.SIZE;

    // Note that this type is actually NOT "java.lang.Double" but "double" - the underlying JDK does an incorrect cast and we are inheriting it.
    public static final Class<java.lang.Double> avm_TYPE = new Class<java.lang.Double>(java.lang.Double.TYPE, new ConstantToken(ShadowClassConstantId.Double_avm_TYPE));

    public static String avm_toHexString(double a)
    {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_toHexString);
        return new String(java.lang.Double.toHexString(a));
    }

    public static String avm_toString(double a)
    {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_toString);
        return internalToString(a);
    }

    public static Double avm_valueOf(String a) throws NumberFormatException
    {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_valueOf);
        return new Double(internalParseDouble(a));
    }

    public static Double avm_valueOf(double origValue) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_valueOf_1);
        return new Double(origValue);
    }

    public static double avm_parseDouble(String a) throws NumberFormatException
    {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_parseDouble);
        return internalParseDouble(a);
    }

    public static boolean avm_isNaN(double v)
    {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_isNaN);
        return java.lang.Double.isNaN(v);
    }

    public static boolean avm_isInfinite(double v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_isInfinite);
        return internalIsInfinite(v);
    }

    public static boolean avm_isFinite(double d) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_isFinite);
        return java.lang.Math.abs(d) <= Double.avm_MAX_VALUE;
    }

    public boolean avm_isNaN() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_isNaN_1);
        lazyLoad();
        return java.lang.Double.isNaN(this.v);
    }

    public boolean avm_isInfinite() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_isInfinite_1);
        lazyLoad();
        return internalIsInfinite(v);
    }

    public String avm_toString()
    {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_toString_1);
        lazyLoad();
        return internalToString(v);
    }

    public byte avm_byteValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_byteValue);
        lazyLoad();
        return (byte) v;
    }

    public short avm_shortValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_shortValue);
        lazyLoad();
        return (short) v;
    }

    public int avm_intValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_intValue);
        lazyLoad();
        return (int) v;
    }

    public long avm_longValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_longValue);
        lazyLoad();
        return (long) v;
    }

    public float avm_floatValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_floatValue);
        lazyLoad();
        return (float) v;
    }

    public double avm_doubleValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_doubleValue);
        lazyLoad();
        return v;
    }

    public int avm_hashCode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_hashCode);
        lazyLoad();
        return java.lang.Double.hashCode(this.v);
    }

    public static int avm_hashCode(double value) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_hashCode_1);
        return java.lang.Double.hashCode(value);
    }

    public static long avm_doubleToLongBits(double value) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_doubleToLongBits);
        return java.lang.Double.doubleToLongBits(value);
    }

    public static double avm_longBitsToDouble(long bits){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_longBitsToDouble);
        return java.lang.Double.longBitsToDouble(bits);
    }

    public int avm_compareTo(Double anotherDouble) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_compareTo);
        lazyLoad();
        anotherDouble.lazyLoad();
        return java.lang.Double.compare(this.v, anotherDouble.v);
    }

    public static int avm_compare(double d1, double d2){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_compare);
        return java.lang.Double.compare(d1, d2);
    }

    public static double avm_sum(double a, double b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_sum);
        return a + b;
    }

    public static double avm_max(double a, double b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_max);
        return java.lang.Math.max(a, b);
    }

    public static double avm_min(double a, double b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_min);
        return java.lang.Math.min(a, b);
    }

    private static String internalToString(double a)
    {
        return new String(java.lang.Double.toString(a));
    }

    private static double internalParseDouble(String a)
    {
        return java.lang.Double.parseDouble(a.getUnderlying());
    }

    private static boolean internalIsInfinite(double v) {
        return (v == avm_POSITIVE_INFINITY) || (v == avm_NEGATIVE_INFINITY);
    }

    public boolean avm_equals(IObject obj) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Double_avm_equals);
        if (obj instanceof Double) {
            Double other = (Double) obj;
            lazyLoad();
            other.lazyLoad();
            return java.lang.Double.doubleToLongBits(this.v) == java.lang.Double.doubleToLongBits(other.v);
        }
        return false;
    }
    //========================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    public Double(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    private double v;

    public double getUnderlying() {
        lazyLoad();
        return this.v;
    }

    //========================================================
    // Methods below are excluded from shadowing
    //========================================================

}
