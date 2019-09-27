package s.java.lang;

import i.*;
import org.aion.avm.RuntimeMethodFeeSchedule;

public final class Byte extends Number implements Comparable<Byte> {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public static final byte avm_MIN_VALUE = java.lang.Byte.MIN_VALUE;

    public static final byte avm_MAX_VALUE = java.lang.Byte.MAX_VALUE;

    public static final Class<Byte> avm_TYPE = new Class(java.lang.Byte.TYPE, new ConstantToken(ShadowClassConstantId.Byte_avm_TYPE));

    public static String avm_toString(byte b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_toString);
        return new String(java.lang.Byte.toString(b));
    }

    public static Byte avm_valueOf(byte b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_valueOf);
        return internalValueOf(b);
    }

    public static byte avm_parseByte(String s, int radix) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_parseByte);
        return internalParseByte(s, radix);
    }

    public static byte avm_parseByte(String s) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_parseByte_1);
        return internalParseByte(s, 10);
    }

    public static Byte avm_valueOf(String s, int radix) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_valueOf_1);
        return internalValueOf(internalParseByte(s, radix));
    }

    public static Byte avm_valueOf(String s) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_valueOf_2);
        return internalValueOf(internalParseByte(s, 10));
    }

    public static Byte avm_decode(String nm) throws NumberFormatException {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_decode);
        return new Byte(java.lang.Byte.decode(nm.getUnderlying()).byteValue());
    }

    // These are the constructors provided in the JDK but we mark them private since they are deprecated.
    // (in the future, we may change these to not exist - depends on the kind of error we want to give the user).
    private Byte(byte v) {
        this.v = v;
    }
    @SuppressWarnings("unused")
    private Byte(String s) {
        throw RuntimeAssertionError.unimplemented("This is only provided for a consistent error to user code - not to be called");
    }

    public byte avm_byteValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_byteValue);
        lazyLoad();
        return v;
    }

    public short avm_shortValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_shortValue);
        lazyLoad();
        return (short) v;
    }

    public int avm_intValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_intValue);
        lazyLoad();
        return (int) v;
    }

    public long avm_longValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_longValue);
        lazyLoad();
        return (long) v;
    }

    public float avm_floatValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_floatValue);
        lazyLoad();
        return (float) v;
    }

    public double avm_doubleValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_doubleValue);
        lazyLoad();
        return (double) v;
    }

    public String avm_toString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_toString_1);
        lazyLoad();
        return new String(java.lang.Byte.toString(this.v));
    }

    @Override
    public int avm_hashCode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_hashCode);
        lazyLoad();
        return internalHashCode(this.v);
    }

    public static int avm_hashCode(byte value) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_hashCode_1);
        return internalHashCode(value);
    }

    public boolean avm_equals(IObject obj) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_equals);
        boolean isEqual = false;
        if (obj instanceof Byte) {
            Byte other = (Byte)obj;
            lazyLoad();
            other.lazyLoad();
            isEqual = v == other.v;
        }
        return isEqual;
    }

    public int avm_compareTo(Byte anotherByte) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_compareTo);
        lazyLoad();
        anotherByte.lazyLoad();
        return internalCompare(this.v, anotherByte.v);
    }

    public static int avm_compare(byte x, byte y) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_compare);
        return internalCompare(x, y);
    }

    public static int avm_compareUnsigned(byte x, byte y) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_compareUnsigned);
        return internalToUnsignedInt(x) - internalToUnsignedInt(y);
    }

    public static int avm_toUnsignedInt(byte x) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_toUnsignedInt);
        return internalToUnsignedInt(x);
    }

    public static long avm_toUnsignedLong(byte x) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Byte_avm_toUnsignedLong);
        return ((long) x) & 0xffL;
    }

    public static final int avm_SIZE = java.lang.Byte.SIZE;

    public static final int avm_BYTES = java.lang.Byte.BYTES;

    private static Byte internalValueOf(byte b) {
        return new Byte(b);
    }

    private static byte internalParseByte(String s, int radix){
        return java.lang.Byte.parseByte(s.getUnderlying(), radix);
    }

    private static int internalHashCode(byte value) {
        return (int)value;
    }

    private static int internalCompare(byte x, byte y) {
        return x - y;
    }

    private static int internalToUnsignedInt(byte x) {
        return ((int) x) & 0xff;
    }

    //=======================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    public Byte(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    private byte v;

    public byte getUnderlying() {
        lazyLoad();
        return this.v;
    }

    //========================================================
    // Methods below are excluded from shadowing
    //========================================================

}
