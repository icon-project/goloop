package s.java.lang;

import i.*;
import org.aion.avm.RuntimeMethodFeeSchedule;

public final class Integer extends Number implements Comparable<Integer> {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public static final int avm_MAX_VALUE = java.lang.Integer.MAX_VALUE;

    public static final int avm_MIN_VALUE = java.lang.Integer.MIN_VALUE;

    public static final int avm_SIZE = java.lang.Integer.SIZE;

    public static final int avm_BYTES = java.lang.Integer.BYTES;

    public static final Class<Integer> avm_TYPE = new Class(java.lang.Integer.TYPE, new ConstantToken(ShadowClassConstantId.Integer_avm_TYPE));

    public static String avm_toString(int i, int radix) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_toString);
        return new String(java.lang.Integer.toString(i, radix));
    }

    public static String avm_toUnsignedString(int i, int radix) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_toUnsignedString);
        return new String(java.lang.Integer.toUnsignedString(i, radix));
    }

    public static String avm_toHexString(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_toHexString);
        return new String(java.lang.Integer.toHexString(i));
    }

    public static String avm_toOctalString(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_toOctalString);
        return new String(java.lang.Integer.toOctalString(i));
    }

    public static String avm_toBinaryString(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_toBinaryString);
        return new String(java.lang.Integer.toBinaryString(i));
    }

    public static String avm_toString(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_toString_1);
        return new String(java.lang.Integer.toString(i));
    }

    public static String avm_toUnsignedString(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_toUnsignedString_1);
        return new String(java.lang.Integer.toUnsignedString(i));
    }

    public static int avm_parseInt(String s, int radix) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_parseInt);
        return internalParseInt(s, radix);
    }

    public static int avm_parseInt(String s) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_parseInt_1);
        return java.lang.Integer.parseInt(s.getUnderlying());
    }

    public static int avm_parseInt(CharSequence s, int beginIndex, int endIndex, int radix)
            throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_parseInt_2);
        return java.lang.Integer.parseInt(s.avm_toString().getUnderlying(), beginIndex, endIndex, radix);
    }

    public static int avm_parseUnsignedInt(String s, int radix) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_parseUnsignedInt);
        return java.lang.Integer.parseUnsignedInt(s.getUnderlying(), radix);
    }

    public static int avm_parseUnsignedInt(String s) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_parseUnsignedInt_1);
        return java.lang.Integer.parseUnsignedInt(s.getUnderlying());
    }

    public static int avm_parseUnsignedInt(CharSequence s, int beginIndex, int endIndex, int radix)
            throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_parseUnsignedInt_2);
        return java.lang.Integer.parseUnsignedInt(s.avm_toString().getUnderlying(), beginIndex, endIndex, radix);
    }

    public static Integer avm_valueOf(String s, int radix) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_valueOf);
        return new Integer(internalParseInt(s, radix));
    }

    public static Integer avm_valueOf(String s) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_valueOf_1);
        return new Integer(internalParseInt(s, 10));
    }

    public static Integer avm_valueOf(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_valueOf_2);
        return new Integer(i);
    }

    // These are the constructors provided in the JDK but we mark them private since they are deprecated.
    // (in the future, we may change these to not exist - depends on the kind of error we want to give the user).
    private Integer(int v) {
        this.v = v;
    }
    @SuppressWarnings("unused")
    private Integer(String s) {
        throw RuntimeAssertionError.unimplemented("This is only provided for a consistent error to user code - not to be called");
    }

    public byte avm_byteValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_byteValue);
        lazyLoad();
        return (byte) v;
    }

    public short avm_shortValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_shortValue);
        lazyLoad();
        return (short) v;
    }

    public int avm_intValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_intValue);
        lazyLoad();
        return v;
    }

    public long avm_longValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_longValue);
        lazyLoad();
        return (long) v;
    }

    public float avm_floatValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_floatValue);
        lazyLoad();
        return (float) v;
    }

    public double avm_doubleValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_doubleValue);
        lazyLoad();
        return (double) v;
    }

    public String avm_toString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_toString_2);
        lazyLoad();
        return new String(java.lang.Integer.toString(this.v));
    }

    @Override
    public int avm_hashCode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_hashCode);
        lazyLoad();
        return this.v;
    }

    public static int avm_hashCode(int value) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_hashCode_1);
        return value;
    }

    public boolean avm_equals(IObject obj) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_equals);
        boolean isEqual = false;
        if (obj instanceof Integer) {
            Integer other = (Integer) obj;
            lazyLoad();
            other.lazyLoad();
            isEqual = this.v == other.v;
        }
        return isEqual;
    }

    public static Integer avm_decode(String nm) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_decode);
        return new Integer(java.lang.Integer.decode(nm.getUnderlying()).intValue());
    }

    public int avm_compareTo(Integer anotherInteger) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_compareTo);
        lazyLoad();
        anotherInteger.lazyLoad();
        return internalCompare(this.v, anotherInteger.v);
    }

    public static int avm_compare(int x, int y) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_compare);
        return internalCompare(x, y);
    }

    public static int avm_compareUnsigned(int x, int y) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_compareUnsigned);
        return internalCompare(x + avm_MIN_VALUE, y + avm_MIN_VALUE);
    }

    public static long avm_toUnsignedLong(int x) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_toUnsignedLong);
        return internalToUnsignedLong(x);
    }

    public static int avm_divideUnsigned(int dividend, int divisor) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_divideUnsigned);
        // In lieu of tricky code, for now just use long arithmetic.
        return (int)(internalToUnsignedLong(dividend) / internalToUnsignedLong(divisor));
    }

    public static int avm_remainderUnsigned(int dividend, int divisor) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_remainderUnsigned);
        // In lieu of tricky code, for now just use long arithmetic.
        return (int)(internalToUnsignedLong(dividend) % internalToUnsignedLong(divisor));
    }

    public static int avm_highestOneBit(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_highestOneBit);
        return java.lang.Integer.highestOneBit(i);
    }

    public static int avm_lowestOneBit(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_lowestOneBit);
        return java.lang.Integer.lowestOneBit(i);
    }

    public static int avm_numberOfLeadingZeros(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_numberOfLeadingZeros);
        return java.lang.Integer.numberOfLeadingZeros(i);
    }

    public static int avm_numberOfTrailingZeros(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_numberOfTrailingZeros);
        return java.lang.Integer.numberOfTrailingZeros(i);
    }

    public static int avm_bitCount(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_bitCount);
        return java.lang.Integer.bitCount(i);
    }

    public static int avm_reverse(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_reverse);
        return java.lang.Integer.reverse(i);
    }

    public static int avm_signum(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_signum);
        return (i >> 31) | (-i >>> 31);
    }

    public static int avm_reverseBytes(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_reverseBytes);
        return java.lang.Integer.reverseBytes(i);
    }

    public static int avm_sum(int a, int b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_sum);
        return a + b;
    }

    public static int avm_max(int a, int b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_max);
        return java.lang.Math.max(a, b);
    }

    public static int avm_min(int a, int b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Integer_avm_min);
        return java.lang.Math.min(a, b);
    }

    private static int internalParseInt(String s, int radix) throws NumberFormatException {
        return java.lang.Integer.parseInt(s.getUnderlying(), radix);
    }

    private static int internalCompare(int x, int y) {
        return (x < y) ? -1 : ((x == y) ? 0 : 1);
    }

    private static long internalToUnsignedLong(int x) {
        return ((long) x) & 0xffffffffL;
    }

    //========================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    public Integer(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    private int v;

    public int getUnderlying() {
        lazyLoad();
        return this.v;
    }

    //========================================================
    // Methods below are excluded from shadowing
    //========================================================

    // public static Integer avm_getInteger(String nm){}

    // public static Integer avm_getInteger(String nm, int val) {}

    // public static Integer avm_getInteger(String nm, Integer val) {}


}
