package s.java.math;

import a.ByteArray;
import i.*;
import s.java.lang.Comparable;
import s.java.lang.String;
import s.java.lang.Number;

import org.aion.avm.RuntimeMethodFeeSchedule;

public class BigInteger extends Number implements Comparable<BigInteger> {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public BigInteger(ByteArray val, int off, int len) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_constructor);
        setUnderlying(new java.math.BigInteger(val.getUnderlying(), off, len));
    }

    public BigInteger(ByteArray val) {
        this(val, 0, val.length());
    }

    public BigInteger(int signum, ByteArray magnitude, int off, int len) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_constructor_2);
        setUnderlying(new java.math.BigInteger(signum, magnitude.getUnderlying(), off, len));
    }

    public BigInteger(int signum, ByteArray magnitude) {
        this(signum, magnitude, 0, magnitude.length());
    }

    public BigInteger(String val, int radix) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_constructor_4);
        setUnderlying(new java.math.BigInteger(val.getUnderlying(), radix));
    }

    public BigInteger(String val) {
        this(val, 10);
    }

    public static BigInteger avm_valueOf(long val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_valueOf);
        return new BigInteger(java.math.BigInteger.valueOf(val));
    }

    public static final BigInteger avm_ZERO = new BigInteger(java.math.BigInteger.ZERO, new ConstantToken(ShadowClassConstantId.BigInteger_avm_ZERO));

    public static final BigInteger avm_ONE = new BigInteger(java.math.BigInteger.ONE, new ConstantToken(ShadowClassConstantId.BigInteger_avm_ONE));

    public static final BigInteger avm_TWO = new BigInteger(java.math.BigInteger.TWO, new ConstantToken(ShadowClassConstantId.BigInteger_avm_TWO));

    public static final BigInteger avm_TEN = new BigInteger(java.math.BigInteger.TEN, new ConstantToken(ShadowClassConstantId.BigInteger_avm_TEN));

    public BigInteger avm_add(BigInteger val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_add);
        lazyLoad();
        val.lazyLoad();
        return new BigInteger(v.add(val.v));
    }

    public BigInteger avm_subtract(BigInteger val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_subtract);
        lazyLoad();
        val.lazyLoad();
        return new BigInteger(v.subtract(val.v));
    }

    public BigInteger avm_multiply(BigInteger val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_multiply);
        lazyLoad();
        val.lazyLoad();
        return new BigInteger(v.multiply(val.v));
    }

    public BigInteger avm_divide(BigInteger val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_divide);
        lazyLoad();
        val.lazyLoad();
        return new BigInteger(v.divide(val.v));
    }

    public BigInteger avm_remainder(BigInteger val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_remainder);
        lazyLoad();
        val.lazyLoad();
        return new BigInteger(v.remainder(val.v));
    }

    public BigInteger avm_sqrt() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_sqrt);
        lazyLoad();
        return new BigInteger(v.sqrt());
    }

    public BigInteger avm_gcd(BigInteger val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_gcd);
        lazyLoad();
        val.lazyLoad();
        return new BigInteger(v.gcd(val.v));
    }

    public BigInteger avm_abs() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_abs);
        lazyLoad();
        return new BigInteger(v.abs());
    }

    public BigInteger avm_negate() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_negate);
        lazyLoad();
        return new BigInteger(v.negate());
    }

    public int avm_signum() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_signum);
        lazyLoad();
        return v.signum();
    }

    public BigInteger avm_mod(BigInteger val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_mod);
        lazyLoad();
        val.lazyLoad();
        return new BigInteger(v.mod(val.v));
    }

    public BigInteger avm_modPow(BigInteger exponent, BigInteger m) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_modPow);
        lazyLoad();
        exponent.lazyLoad();
        m.lazyLoad();
        return new BigInteger(v.modPow(exponent.v, m.v));
    }

    public BigInteger avm_modInverse(BigInteger val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_modInverse);
        lazyLoad();
        val.lazyLoad();
        return new BigInteger(v.modInverse(val.v));
    }

    public BigInteger avm_shiftLeft(int n) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_shiftLeft);
        verifyBitLength(n);
        lazyLoad();
        return new BigInteger(v.shiftLeft(n));
    }

    public BigInteger avm_shiftRight(int n) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_shiftRight);
        verifyBitLength(n);
        lazyLoad();
        return new BigInteger(v.shiftRight(n));
    }

    public BigInteger avm_and(BigInteger val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_and);
        lazyLoad();
        val.lazyLoad();
        return new BigInteger(v.and(val.v));
    }

    public BigInteger avm_or(BigInteger val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_or);
        lazyLoad();
        val.lazyLoad();
        return new BigInteger(v.or(val.v));
    }

    public BigInteger avm_xor(BigInteger val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_xor);
        lazyLoad();
        val.lazyLoad();
        return new BigInteger(v.xor(val.v));
    }

    public BigInteger avm_not() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_not);
        lazyLoad();
        return new BigInteger(v.not());
    }

    public BigInteger avm_andNot(BigInteger val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_andNot);
        lazyLoad();
        val.lazyLoad();
        return new BigInteger(v.andNot(val.v));
    }

    public boolean avm_testBit(int n) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_testBit);
        verifyBitLength(n);
        lazyLoad();
        return v.testBit(n);
    }

    public BigInteger avm_setBit(int n) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_setBit);
        verifyBitLength(n);
        lazyLoad();
        return new BigInteger(v.setBit(n));
    }

    public BigInteger avm_clearBit(int n) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_clearBit);
        verifyBitLength(n);
        lazyLoad();
        return new BigInteger(v.clearBit(n));
    }

    public BigInteger avm_flipBit(int n) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_flipBit);
        verifyBitLength(n);
        lazyLoad();
        return new BigInteger(v.flipBit(n));
    }

    public int avm_getLowestSetBit() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_getLowestSetBit);
        lazyLoad();
        return v.getLowestSetBit();
    }

    public int avm_bitLength() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_bitLength);
        lazyLoad();
        return v.bitLength();
    }

    public int avm_bitCount() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_bitCount);
        lazyLoad();
        return v.bitCount();
    }

    public int avm_compareTo(BigInteger val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_compareTo);
        lazyLoad();
        val.lazyLoad();
        return v.compareTo(val.v);
    }

    public boolean avm_equals(IObject x) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_equals);
        if (x == this)
            return true;

        if (!(x instanceof BigInteger))
            return false;

        BigInteger xInt = (BigInteger) x;
        lazyLoad();
        xInt.lazyLoad();
        return v.equals(xInt.v);
    }

    public BigInteger avm_min(BigInteger val){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_min);
        lazyLoad();
        val.lazyLoad();
        return new BigInteger(v.min(val.v));
    }

    public BigInteger avm_max(BigInteger val){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_max);
        lazyLoad();
        val.lazyLoad();
        return new BigInteger(v.max(val.v));
    }

    public int avm_hashCode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_hashCode);
        lazyLoad();
        return v.hashCode();
    }

    public String avm_toString(int radix){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_toString);
        lazyLoad();
        return new String(v.toString(radix));
    }

    public String avm_toString(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_toString_1);
        lazyLoad();
        return new String(v.toString());
    }

    public ByteArray avm_toByteArray() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_toByteArray);
        lazyLoad();
        return new ByteArray(v.toByteArray());
    }

    public int avm_intValue(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_intValue);
        lazyLoad();
        return v.intValue();
    }

    public long avm_longValue(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_longValue);
        lazyLoad();
        return v.longValue();
    }

    public float avm_floatValue(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_floatValue);
        lazyLoad();
        return v.floatValue();
    }

    public double avm_doubleValue(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_doubleValue);
        lazyLoad();
        return v.doubleValue();
    }

    public long avm_longValueExact(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_longValueExact);
        lazyLoad();
        return v.longValueExact();
    }

    public int avm_intValueExact() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_intValueExact);
        lazyLoad();
        return v.intValueExact();
    }

    public short avm_shortValueExact() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_shortValueExact);
        lazyLoad();
        return v.shortValueExact();
    }

    public byte avm_byteValueExact() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigInteger_avm_byteValueExact);
        lazyLoad();
        return v.byteValueExact();
    }

    //========================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    private java.math.BigInteger v;

    public BigInteger(java.math.BigInteger u) {
        setUnderlying(u);
    }

    public static BigInteger newWithCharge(java.math.BigInteger u) {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.BigInteger_avm_constructor);
        return new BigInteger(u);
    }

    private void setUnderlying(java.math.BigInteger u) {
        if (isValidRange(u)) {
            v = u;
        }
    }

    public java.math.BigInteger getUnderlying() {
        lazyLoad();
        return v;
    }

    private BigInteger(java.math.BigInteger u, ConstantToken constantToken) {
        super(constantToken);
        setUnderlying(u);
    }

    // Deserializer support.
    public BigInteger(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(BigInteger.class, deserializer);
        
        // We can deserialize this as its actual 2s compliment byte array.
        this.v = new java.math.BigInteger(CodecIdioms.deserializeByteArray(deserializer));
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(BigInteger.class, serializer);
        
        // We can serialize this as its actual 2s compliment byte array.
        CodecIdioms.serializeByteArray(serializer, this.v.toByteArray());
    }

    private boolean isValidRange(java.math.BigInteger u) {
        if (u.abs().bitLength() > 512) {
            throw new ArithmeticException("Out of the supported range");
        }
        return true;
    }

    private void verifyBitLength(int length) {
        if (Math.abs(length) > 512) {
            throw new ArithmeticException("Out of the supported range");
        }
    }
    //========================================================
    // Methods below are excluded from shadowing
    //========================================================

    //public BigInteger(int numBits, Random rnd)

    //public BigInteger(int bitLength, int certainty, Random rnd)

    //public static BigInteger probablePrime(int bitLength, Random rnd)

    //private static BigInteger smallPrime(int bitLength, int certainty, Random rnd)

    //private static BigInteger largePrime(int bitLength, int certainty, Random rnd)

    //public BigInteger[] divideAndRemainder(BigInteger val)

    //public BigInteger[] sqrtAndRemainder()

    //public boolean isProbablePrime(int certainty)



}
