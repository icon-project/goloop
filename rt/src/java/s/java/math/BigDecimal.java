package s.java.math;

import i.CodecIdioms;
import i.IInstrumentation;
import i.IObject;
import i.IObjectDeserializer;
import i.IObjectSerializer;
import s.java.lang.Comparable;
import s.java.lang.String;
import s.java.lang.Number;
import org.aion.avm.RuntimeMethodFeeSchedule;


public class BigDecimal extends Number implements Comparable<BigDecimal>{
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public BigDecimal(String val){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_constructor_4);

        java.lang.String underlying = val.getUnderlying();
        if (!isValidString(underlying)) {
            throw new NumberFormatException();
        }

        v = new java.math.BigDecimal(underlying);
    }

    public BigDecimal(String val, MathContext mc){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_constructor_5);

        java.lang.String underlying = val.getUnderlying();
        if (!isValidString(underlying)) {
            throw new NumberFormatException();
        }

        v = new java.math.BigDecimal(underlying, mc.getUnderlying());
    }

    public BigDecimal(double val){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_constructor_6);
        v = new java.math.BigDecimal(val);
    }

    public BigDecimal(double val, MathContext mc) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_constructor_7);
        v = new java.math.BigDecimal(val, mc.getUnderlying());
    }

    public BigDecimal(int val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_constructor_12);
        v = new java.math.BigDecimal(val);
    }

    public BigDecimal(int val, MathContext mc) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_constructor_13);
        v = new java.math.BigDecimal(val, mc.getUnderlying());
    }

    public BigDecimal(long val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_constructor_14);
        v = new java.math.BigDecimal(val);
    }

    public BigDecimal(long val, MathContext mc) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_constructor_15);
        v = new java.math.BigDecimal(val, mc.getUnderlying());
    }

    public static BigDecimal avm_valueOf(long val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_valueOf_1);
        return new BigDecimal(java.math.BigDecimal.valueOf(val));
    }

    public static BigDecimal avm_valueOf(double val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_valueOf_2);
        return new BigDecimal(java.math.BigDecimal.valueOf(val));
    }

    public int avm_compareTo(BigDecimal val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_compareTo);
        lazyLoad();
        val.lazyLoad();
        return v.compareTo(val.v);
    }

    public int avm_hashCode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_hashCode);
        lazyLoad();
        return v.hashCode();
    }

    public String avm_toString(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_toString);
        lazyLoad();
        return new String(v.toString());
    }

    public String avm_toPlainString(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_toPlainString);
        lazyLoad();
        return new String(v.toPlainString());
    }

    public BigInteger avm_toBigInteger() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_toBigInteger);
        lazyLoad();
        return new BigInteger(v.toBigInteger());
    }

    public BigInteger avm_toBigIntegerExact() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_toBigIntegerExact);
        lazyLoad();
        return new BigInteger(v.toBigIntegerExact());
    }

    public long avm_longValue(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_longValue);
        lazyLoad();
        return v.longValue();
    }

    public long avm_longValueExact(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_longValueExact);
        lazyLoad();
        return v.longValueExact();
    }

    public int avm_intValue(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_intValue);
        lazyLoad();
        return v.intValue();
    }

    public int avm_intValueExact() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_intValueExact);
        lazyLoad();
        return v.intValueExact();
    }

    public short avm_shortValueExact() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_shortValueExact);
        lazyLoad();
        return v.shortValueExact();
    }

    public byte avm_byteValueExact() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_byteValueExact);
        lazyLoad();
        return v.byteValueExact();
    }

    public float avm_floatValue(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_floatValue);
        lazyLoad();
        return v.floatValue();
    }

    public double avm_doubleValue(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_doubleValue);
        lazyLoad();
        return v.doubleValue();
    }

    public boolean avm_equals(IObject x) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.BigDecimal_avm_equals);
        if (x == this) {
            return true;
        }

        if (!(x instanceof BigDecimal)) {
            return false;
        }

        BigDecimal xInt = (BigDecimal) x;
        lazyLoad();
        xInt.lazyLoad();
        return v.equals(xInt.v);
    }

    //========================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    private java.math.BigDecimal v;

    public BigDecimal(java.math.BigDecimal u) {
        v = u;
    }

    public java.math.BigDecimal getUnderlying() {
        lazyLoad();
        return v;
    }

    // Deserializer support.
    public BigDecimal(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(BigDecimal.class, deserializer);
        
        // We deserialize this as a string.
        java.lang.String simpler = CodecIdioms.deserializeString(deserializer);
        this.v = new java.math.BigDecimal(simpler);
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(BigDecimal.class, serializer);
        
        // We serialize this as a string.
        CodecIdioms.serializeString(serializer, this.v.toString());
    }

    /**
     * Returns true only if the given string is a valid string to create a BigDecimal with. A valid
     * string must be:
     *
     * 1. Length 78 or less. Length 78 is the smallest length at which a conversion to BigInteger fails
     * when a sign character is present.
     * Note that there are still a subset of length 78 strings that convert to valid BigInteger's.
     *
     * 2. All characters in the string must be ASCII digits (zero through nine) with the exception of
     * the initial character, which may also be one of the two sign characters '+' or '-'. Nothing
     * else is permitted.
     *
     * These changes have been put in place as a result of AKI-254.
     *
     * @param string The string.
     * @return whether the string is valid or not.
     */
    private static boolean isValidString(java.lang.String string) {
        boolean isValid = true;

        if (string.length() > 78) {
            isValid = false;
        } else {
            boolean isFirstChar = true;
            for (char character : string.toCharArray()) {
                if (isFirstChar) {
                    if ((character < '0' || character > '9') && (character != '+') && (character != '-')) {
                        isValid = false;
                    }
                } else {
                    if (character < '0' || character > '9') {
                        isValid = false;
                    }
                }
                isFirstChar = false;
            }
        }

        return isValid;
    }

    //========================================================
    // Methods below are deprecated
    //========================================================

    //public BigDecimal divide(BigDecimal divisor, int scale, int roundingMode)

    //public BigDecimal divide(BigDecimal divisor, int roundingMode)

    //public BigDecimal setScale(int newScale, int roundingMode)

    //========================================================
    // Methods below are excluded from shadowing
    //========================================================

    //public java.math.BigDecimal[] divideAndRemainder(java.math.BigDecimal divisor)

    //public BigDecimal[] divideAndRemainder(BigDecimal divisor, MathContext mc)
}
