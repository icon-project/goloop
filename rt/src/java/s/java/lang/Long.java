package s.java.lang;

import i.*;
import org.aion.avm.RuntimeMethodFeeSchedule;

public final class Long extends Number implements Comparable<Long> {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public static final long avm_MIN_VALUE = 0x8000000000000000L;

    public static final long avm_MAX_VALUE = 0x7fffffffffffffffL;

    public static final Class<Long> avm_TYPE = new Class(java.lang.Long.TYPE, new ConstantToken(ShadowClassConstantId.Long_avm_TYPE));

    public static String avm_toString(long i, int radix) {
        // Billing associated with this method is set to level 4 because of slow execution time of radix 2
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_toString);
        return new String(java.lang.Long.toString(i, radix));
    }

    public static String avm_toUnsignedString(long i, int radix){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_toUnsignedString);
        return new String(java.lang.Long.toUnsignedString(i, radix));
    }

    public static String avm_toHexString(long i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_toHexString);
        return new String(java.lang.Long.toHexString(i));
    }

    public static String avm_toOctalString(long i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_toOctalString);
        return new String(java.lang.Long.toOctalString(i));
    }

    public static String avm_toBinaryString(long i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_toBinaryString);
        return new String(java.lang.Long.toBinaryString(i));
    }

    public static String avm_toString(long i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_toString_1);
        return internalToString(i);
    }

    public static String avm_toUnsignedString(long i){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_toUnsignedString_1);
        return new String(java.lang.Long.toUnsignedString(i));
    }

    public static long avm_parseLong(String s, int radix) throws NumberFormatException{
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_parseLong);
        return internalParseLong(s, radix);
    }

    public static long avm_parseLong(String s) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_parseLong_1);
        return java.lang.Long.parseLong(s.getUnderlying(), 10);
    }

    public static long avm_parseLong(CharSequence s, int beginIndex, int endIndex, int radix)
            throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_parseLong_2);
        return java.lang.Long.parseLong(s.avm_toString().getUnderlying(), beginIndex, endIndex, radix);
    }

    public static long avm_parseUnsignedLong(String s, int radix) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_parseUnsignedLong);
        return java.lang.Long.parseUnsignedLong(s.getUnderlying(), radix);
    }

    public static long avm_parseUnsignedLong(String s) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_parseUnsignedLong_1);
        return java.lang.Long.parseUnsignedLong(s.getUnderlying(), 10);
    }

    public static long avm_parseUnsignedLong(CharSequence s, int beginIndex, int endIndex, int radix)
            throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_parseUnsignedLong_2);
        return java.lang.Long.parseUnsignedLong(s.avm_toString().getUnderlying(), beginIndex, endIndex, radix);
    }

    public static Long avm_valueOf(String s, int radix) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_valueOf);
        return new Long(internalParseLong(s, radix));
    }

    public static Long avm_valueOf(String s) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_valueOf_1);
        return new Long(internalParseLong(s, 10));
    }

    public static Long avm_valueOf(long l) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_valueOf_2);
        return new Long(l);
    }

    public static Long avm_decode(String nm) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_decode);
        return new Long(java.lang.Long.decode(nm.getUnderlying()).longValue());
    }

    // These are the constructors provided in the JDK but we mark them private since they are deprecated.
    // (in the future, we may change these to not exist - depends on the kind of error we want to give the user).
    private Long(long v) {
        this.v = v;
    }
    @SuppressWarnings("unused")
    private Long(String s) {
        throw RuntimeAssertionError.unimplemented("This is only provided for a consistent error to user code - not to be called");
    }

    public byte avm_byteValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_byteValue);
        lazyLoad();
        return (byte) v;
    }

    public short avm_shortValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_shortValue);
        lazyLoad();
        return (short) v;
    }

    public int avm_intValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_intValue);
        lazyLoad();
        return (int) v;
    }

    public long avm_longValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_longValue);
        lazyLoad();
        return v;
    }

    public float avm_floatValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_floatValue);
        lazyLoad();
        return (float) v;
    }

    public double avm_doubleValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_doubleValue);
        lazyLoad();
        return (double) v;
    }

    public String avm_toString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_toString_2);
        lazyLoad();
        return internalToString(this.v);
    }

    public int avm_hashCode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_hashCode);
        lazyLoad();
        return internalHashCode(this.v);
    }

    public static int avm_hashCode(long value) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_hashCode_1);
        return internalHashCode(value);
    }

    public boolean avm_equals(IObject obj) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_equals);
        if (obj instanceof Long) {
            Long other = (Long) obj;
            lazyLoad();
            other.lazyLoad();
            return this.v == other.v;
        }
        return false;
    }

    public int avm_compareTo(Long anotherLong) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_compareTo);
        return internalCompare(this.v, anotherLong.v);
    }

    public static int avm_compare(long x, long y) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_compare);
        return internalCompare(x, y);
    }

    public static int avm_compareUnsigned(long x, long y) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_compareUnsigned);
        return internalCompare(x + avm_MIN_VALUE, y + avm_MIN_VALUE);
    }

    public static long avm_divideUnsigned(long dividend, long divisor){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_divideUnsigned);
        return java.lang.Long.divideUnsigned(dividend, divisor);
    }

    public static long avm_remainderUnsigned(long dividend, long divisor){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_remainderUnsigned);
        return java.lang.Long.remainderUnsigned(dividend, divisor);
    }

    public static final int avm_SIZE = java.lang.Long.SIZE;

    public static final int avm_BYTES = java.lang.Long.BYTES;

    public static long avm_highestOneBit(long i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_highestOneBit);
        return java.lang.Long.highestOneBit(i);
    }

    public static long avm_lowestOneBit(long i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_lowestOneBit);
        return java.lang.Long.lowestOneBit(i);
    }

    public static int avm_numberOfLeadingZeros(long i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_numberOfLeadingZeros);
        return java.lang.Long.numberOfLeadingZeros(i);
    }

    public static int avm_numberOfTrailingZeros(long i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_numberOfTrailingZeros);
        return java.lang.Long.numberOfTrailingZeros(i);
    }

    public static int avm_bitCount(long i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_bitCount);
        return java.lang.Long.bitCount(i);
    }

    public static long avm_rotateLeft(long i, int distance) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_rotateLeft);
        return (i << distance) | (i >>> -distance);
    }

    public static long avm_rotateRight(long i, int distance) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_rotateRight);
        return (i >>> distance) | (i << -distance);
    }

    public static long avm_reverse(long i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_reverse);
        return java.lang.Long.reverse(i);
    }

    public static int avm_signum(long i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_signum);
        return (int) ((i >> 63) | (-i >>> 63));
    }

    public static long avm_reverseBytes(long i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_reverseBytes);
        return java.lang.Long.reverseBytes(i);
    }

    public static long avm_sum(long a, long b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_sum);
        return a + b;
    }

    public static long avm_max(long a, long b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_max);
        return java.lang.Math.max(a, b);
    }

    public static long avm_min(long a, long b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Long_avm_min);
        return java.lang.Math.min(a, b);
    }

    private static String internalToString(long i) {
        return new String(java.lang.Long.toString(i));
    }

    private static long internalParseLong(String s, int radix) throws NumberFormatException{
        return java.lang.Long.parseLong(s.getUnderlying(), radix);
    }

    private static int internalHashCode(long value) {
        return (int)(value ^ (value >>> 32));
    }

    private static int internalCompare(long x, long y) {
        return (x < y) ? -1 : ((x == y) ? 0 : 1);
    }

    //========================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    public Long(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    private long v;

    public long getUnderlying() {
        lazyLoad();
        return this.v;
    }

    //========================================================
    // Methods below are excluded from shadowing
    //========================================================

    // public static Long avm_getLong(String nm) {}

    // public static Long avm_getLong(String nm, long val) {}

    // public static Long avm_getLong(String nm, Long val) {}

}
