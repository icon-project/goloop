package s.java.lang;

import a.CharArray;
import i.CodecIdioms;
import i.IInstrumentation;
import i.IObject;
import i.IObjectDeserializer;
import i.IObjectSerializer;
import org.aion.avm.EnergyCalculator;
import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.io.Serializable;


public final class StringBuilder extends Object implements CharSequence, Serializable, Appendable {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public StringBuilder() {
        EnergyCalculator.chargeEnergy(RuntimeMethodFeeSchedule.StringBuilder_avm_constructor);
        this.v = new java.lang.StringBuilder();
    }

    public StringBuilder(int capacity) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_constructor_1,
                java.lang.Math.max(capacity, 0));
        this.v = new java.lang.StringBuilder(capacity);
    }

    public StringBuilder(String str) {
        int lengthForBilling = (null != str) ? str.internalLength() : 0;
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_constructor_2, lengthForBilling);
        this.v = new java.lang.StringBuilder(str.getUnderlying());
    }

    public StringBuilder(CharSequence seq) {
        int lengthForBilling = (null != seq) ? seq.avm_length() : 0;
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_constructor_3, lengthForBilling);
        this.v = new java.lang.StringBuilder();
        internalAppend(seq);
    }

    public StringBuilder avm_append(IObject obj) {
        String str = String.internalValueOfObject(obj);
        // Note that we want to convert this to a string, at our level, so we can call avm_toString() - the lower-level will call toString().
        int strLen = (null != str) ? str.internalLength() : 0;
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_append, strLen, internalLength() + strLen);
        internalAppend(str);
        return this;
    }

    public StringBuilder avm_append(String str) {
        int strLen = (null != str) ? str.internalLength() : 0;
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_append_1, strLen, internalLength() + strLen);
        internalAppend(str);
        return this;
    }

    public StringBuilder avm_append(StringBuffer sb) {
        int strLen = (null != sb) ? sb.internalLength() : 0;
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_append_2, strLen, internalLength() + strLen);
        java.lang.StringBuffer underlying = (null != sb)
                ? sb.getUnderlying()
                : null;
        this.v.append(underlying);
        return this;
    }

    public StringBuilder avm_append(CharArray str) {
        int strLen = (null != str) ? str.length() : 0;
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_append_3, strLen, internalLength() + strLen);
        this.v.append(str.getUnderlying());
        return this;
    }

    public StringBuilder avm_append(CharArray str, int offset, int len) {
        int oldLen = java.lang.Math.max(len, 0) + java.lang.Math.max(internalLength() - offset, 0);
        int newLen = internalLength() + java.lang.Math.max(len, 0);
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_append_4, oldLen, newLen);
        char[] underlying = (null != str)
                ? str.getUnderlying()
                : null;
        this.v.append(underlying, offset, len);
        return this;
    }

    public StringBuilder avm_append(CharSequence s) {
        int csLen = (null != s) ? s.avm_length() : 0;
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_append_5, csLen, internalLength() + csLen);
        internalAppend(s);
        return this;
    }

    public StringBuilder avm_append(CharSequence s, int start, int end) {
        int csLen = java.lang.Math.max(end - start, 0);
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_append_6, csLen, internalLength() + csLen);
        java.lang.String asString = (null != s)
                ? s.avm_toString().getUnderlying()
                : null;
        this.v.append(asString, start, end);
        return this;
    }

    public StringBuilder avm_append(boolean b) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_append_7, 0, internalLength());
        this.v.append(b);
        return this;
    }

    public StringBuilder avm_append(char c) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_append_8, 0, internalLength());
        this.v.append(c);
        return this;
    }

    public StringBuilder avm_append(int i) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_append_9, 0, internalLength());
        this.v.append(i);
        return this;
    }

    public StringBuilder avm_append(long l) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_append_10, 0, internalLength());
        this.v.append(l);
        return this;
    }

    public StringBuilder avm_append(float f) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_append_11, 0, internalLength());
        this.v.append(f);
        return this;
    }

    public StringBuilder avm_append(double d) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_append_12, 0, internalLength());
        this.v.append(d);
        return this;
    }

    public StringBuilder avm_delete(int start, int end) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_delete,
                java.lang.Math.max(internalLength() - start, 0));
        this.v.delete(start, end);
        return this;
    }

    public StringBuilder avm_deleteCharAt(int index) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_deleteCharAt,
                java.lang.Math.max(internalLength() - index, 0));
        this.v.deleteCharAt(index);
        return this;
    }

    public StringBuilder avm_replace(int start, int end, String str) {
        int strLen = (null != str) ? str.internalLength() : 0;
        int oldLen = java.lang.Math.max(internalLength() - start, 0);
        int newLen = internalLength() + strLen - java.lang.Math.max(end - start, 0);
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_replace, oldLen, newLen);
        java.lang.String underlying = (null != str)
                ? str.getUnderlying()
                : null;
        this.v.replace(start, end, underlying);
        return this;
    }

    public StringBuilder avm_insert(int index, CharArray str, int offset, int len) {
        int oldLen = java.lang.Math.max(len, 0) + java.lang.Math.max(internalLength() - index, 0);
        int newLen = internalLength() + java.lang.Math.max(len, 0);
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_insert, oldLen, newLen);
        this.v.insert(index, str.getUnderlying(), offset, len);
        return this;
    }

    public StringBuilder avm_insert(int offset, IObject obj) {
        //delegating the call to avm_insert
        avm_insert(offset, String.internalValueOfObject(obj));
        return this;
    }

    public StringBuilder avm_insert(int offset, String str) {
        int strLen = (null != str) ? str.internalLength() : 0;
        int oldLen = strLen + java.lang.Math.max(internalLength() - offset, 0);
        int newLen = internalLength() + strLen;
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_insert_2, oldLen, newLen);
        java.lang.String underlying = (null != str)
                ? str.getUnderlying()
                : null;
        this.v.insert(offset, underlying);
        return this;
    }

    public StringBuilder avm_insert(int offset, CharArray str) {
        int strLen = (null != str) ? str.length() : 0;
        int oldLen = strLen + java.lang.Math.max(internalLength() - offset, 0);
        int newLen = internalLength() + strLen;
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_insert_3, oldLen, newLen);
        // Note the underlying value is not used since this will actually throw NPE if given null.
        this.v.insert(offset, str.getUnderlying());
        return this;
    }

    public StringBuilder avm_insert(int dstOffset, CharSequence s) {
        int csLen = (null != s) ? s.avm_length() : 0;
        int oldLen = csLen + java.lang.Math.max(internalLength() - dstOffset, 0);
        int newLen = internalLength() + csLen;
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_insert_4, oldLen, newLen);
        java.lang.String underlying = (null != s)
                ? s.avm_toString().getUnderlying()
                : null;
        this.v.insert(dstOffset, underlying);
        return this;
    }

    public StringBuilder avm_insert(int dstOffset, CharSequence s, int start, int end) {
        int oldLen = java.lang.Math.max(end - start, 0) + java.lang.Math.max(internalLength() - dstOffset, 0);
        int newLen = internalLength() + java.lang.Math.max(end - start, 0);
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_insert_5, oldLen, newLen);
        java.lang.String underlying = (null != s)
                ? s.avm_toString().getUnderlying()
                : null;
        this.v.insert(dstOffset, underlying, start, end);
        return this;
    }

    public StringBuilder avm_insert(int offset, boolean b) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_insert_6, 0, internalLength());
        this.v.insert(offset, b);
        return this;
    }

    public StringBuilder avm_insert(int offset, char c) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_insert_7, 0, internalLength());
        this.v.insert(offset, c);
        return this;
    }

    public StringBuilder avm_insert(int offset, int i) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_insert_8, 0, internalLength());
        this.v.insert(offset, i);
        return this;
    }

    public StringBuilder avm_insert(int offset, long l) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_insert_9, 0, internalLength());
        this.v.insert(offset, l);
        return this;
    }

    public StringBuilder avm_insert(int offset, float f) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_insert_10, 0, internalLength());
        this.v.insert(offset, f);
        return this;
    }

    public StringBuilder avm_insert(int offset, double d) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_insert_11, 0, internalLength());
        this.v.insert(offset, d);
        return this;
    }

    public int avm_indexOf(String str) {
        int strLen = (null != str) ? str.internalLength() : 0;
        EnergyCalculator.chargeEnergyForIndexOf(RuntimeMethodFeeSchedule.StringBuilder_avm_indexOf,
                internalLength(), strLen, 0);
        return this.v.indexOf(str.getUnderlying());
    }

    public int avm_indexOf(String str, int fromIndex) {
        int strLen = (null != str) ? str.internalLength() : 0;
        EnergyCalculator.chargeEnergyForIndexOf(RuntimeMethodFeeSchedule.StringBuilder_avm_indexOf_1,
                internalLength(), strLen, fromIndex);
        return this.v.indexOf(str.getUnderlying(), fromIndex);
    }

    public int avm_lastIndexOf(String str) {
        int strLen = (null != str) ? str.internalLength() : 0;
        EnergyCalculator.chargeEnergyForLastIndexOf(RuntimeMethodFeeSchedule.StringBuilder_avm_lastIndexOf,
                internalLength(), strLen);
        return this.v.lastIndexOf(str.getUnderlying());
    }

    public int avm_lastIndexOf(String str, int fromIndex) {
        int strLen = (null != str) ? str.internalLength() : 0;
        EnergyCalculator.chargeEnergyForLastIndexOf(RuntimeMethodFeeSchedule.StringBuilder_avm_lastIndexOf_1,
                internalLength(), strLen, fromIndex);
        return this.v.lastIndexOf(str.getUnderlying(), fromIndex);
    }

    public StringBuilder avm_reverse() {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_reverse, internalLength());
        this.v.reverse();
        return this;
    }

    public String avm_toString() {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_toString, internalLength());
        return internalToString();
    }

    public char avm_charAt(int index) {
        EnergyCalculator.chargeEnergy(RuntimeMethodFeeSchedule.StringBuilder_avm_charAt);
        return this.v.charAt(index);
    }

    public CharSequence avm_subSequence(int start, int end) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_subSequence,
                java.lang.Math.max(end - start, 0));
        // Call substring instead of subSequence, since our String wrapper wraps a String, not a CharSequence.
        return new String(this.getUnderlying().subSequence(start, end).toString());
    }

    public int avm_length() {
        EnergyCalculator.chargeEnergy(RuntimeMethodFeeSchedule.StringBuilder_avm_length);
        return internalLength();
    }

    public void avm_setLength(int newLength) {
        EnergyCalculator.chargeEnergyLevel2(RuntimeMethodFeeSchedule.StringBuilder_avm_setLength, 0, newLength);
        this.v.setLength(newLength);
    }

    //========================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    private java.lang.StringBuilder v;

    public java.lang.StringBuilder getUnderlying() {
        return v;
    }

    // Deserializer support.
    public StringBuilder(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(StringBuilder.class, deserializer);
        // We serialize this as a string.
        java.lang.String simpler = CodecIdioms.deserializeString(deserializer);
        this.v = new java.lang.StringBuilder(simpler);
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(StringBuilder.class, serializer);
        // We serialize this as a string.
        CodecIdioms.serializeString(serializer, this.v.toString());
    }

    public int internalLength() {
        return getUnderlying().length();
    }

    public String internalToString() {
        return new String(new java.lang.String(getUnderlying()));
    }

    private void internalAppend(CharSequence s) {
        java.lang.String asString = (null != s)
                ? s.avm_toString().getUnderlying()
                : null;
        this.v.append(asString);
    }

    private void internalAppend(String str) {
        java.lang.String underlying = (null != str)
                ? str.getUnderlying()
                : null;
        this.v.append(underlying);
    }
}
