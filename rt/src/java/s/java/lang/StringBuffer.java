package s.java.lang;

import a.CharArray;
import i.*;
import org.aion.avm.EnergyCalculator;
import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.io.Serializable;


public final class StringBuffer extends Object implements CharSequence, Serializable, Appendable{
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public StringBuffer() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_constructor);
        this.v = new java.lang.StringBuffer();
    }

    public StringBuffer(int capacity) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_constructor_1, java.lang.Math.max(capacity, 0)));
        this.v = new java.lang.StringBuffer(capacity);
    }

    public StringBuffer(String str) {
        int lengthForBilling = (null != str)
                ? str.internalLength()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_constructor_2, lengthForBilling));
        this.v = new java.lang.StringBuffer(str.getUnderlying());
    }

    public StringBuffer(CharSequence seq) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_constructor_3);
        this.v = new java.lang.StringBuffer();
        avm_append(seq);
    }

    public int avm_length() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_length);
        return internalLength();
    }

    public int avm_capacity() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_capacity);
        return this.v.capacity();
    }

    public void avm_ensureCapacity(int minimumCapacity){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_ensureCapacity);
        this.v.ensureCapacity(minimumCapacity);
    }

    public void avm_trimToSize() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_trimToSize, internalLength()));
        this.v.trimToSize();
    }

    public void avm_setLength(int newLength) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_setLength);
        this.v.setLength(newLength);
    }

    public char avm_charAt(int index) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_charAt);
        return this.v.charAt(index);
    }

    public void avm_getChars(int srcBegin, int srcEnd, CharArray dst,
                             int dstBegin)
    {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_getChars, java.lang.Math.max(srcEnd - srcBegin, 0)));
        this.v.getChars(srcBegin, srcEnd, dst.getUnderlying(), dstBegin);
    }

    public void avm_setCharAt(int index, char ch) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_setCharAt);
        this.v.setCharAt(index, ch);
    }

    public StringBuffer avm_append(IObject obj) {
        //delegating the call to avm_append
        this.avm_append(String.internalValueOfObject(obj));
        return this;
    }

    public StringBuffer avm_append(String str) {
        int lengthForBilling = (null != str)
                ? str.internalLength()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_append_1, lengthForBilling));
        java.lang.String underlying = (null != str)
                ? str.getUnderlying()
                : null;
        this.v = this.v.append(underlying);
        return this;
    }

    public StringBuffer avm_append(StringBuffer sb) {
        int lengthForBilling = (null != sb)
                ? sb.internalLength()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_append_2, lengthForBilling));
        java.lang.StringBuffer underlying = (null != sb)
                ? sb.getUnderlying()
                : null;
        this.v = this.v.append(underlying);
        return this;
    }

    public StringBuffer avm_append(CharSequence s){
        int lengthForBilling = (null != s)
                ? s.avm_length()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_append_3, lengthForBilling));
        java.lang.String underlying = (null != s)
                ? s.avm_toString().getUnderlying()
                : null;
        this.v = this.v.append(underlying);
        return this;
    }

    public StringBuffer avm_append(CharSequence s, int start, int end){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_append_4, java.lang.Math.max(end - start, 0)));
        java.lang.String underlying = (null != s)
                ? s.avm_toString().getUnderlying()
                : null;
        this.v = this.v.append(underlying, start, end);
        return this;
    }

    public StringBuffer avm_append(CharArray str) {
        int lengthForBilling = (null != str)
                ? str.length()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_append_5, lengthForBilling));
        // Note the underlying value is not used since this will actually throw NPE if given null.
        this.v = this.v.append(str.getUnderlying());
        return this;
    }

    public StringBuffer avm_append(CharArray str, int offset, int len) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_append_6, (java.lang.Math.max(len, 0) + java.lang.Math.max(internalLength() - offset, 0))));
        this.v = this.v.append(str.getUnderlying(), offset, len);
        return this;
    }

    public StringBuffer avm_append(boolean b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_append_7);
        this.v = this.v.append(b);
        return this;
    }

    public StringBuffer avm_append(char c) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_append_8);
        this.v = this.v.append(c);
        return this;
    }

    public StringBuffer avm_append(int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_append_9);
        this.v = this.v.append(i);
        return this;
    }

    public StringBuffer avm_append(long lng) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_append_10);
        this.v = this.v.append(lng);
        return this;
    }

    public StringBuffer avm_append(float f) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_append_11);
        this.v = this.v.append(f);
        return this;
    }

    public StringBuffer avm_append(double d) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_append_12);
        this.v = this.v.append(d);
        return this;
    }

    public StringBuffer avm_delete(int start, int end) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_delete, java.lang.Math.max(internalLength() - start, 0)));
        this.v = this.v.delete(start, end);
        return this;
    }

    public StringBuffer avm_deleteCharAt(int index) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_deleteCharAt, java.lang.Math.max(internalLength() - index, 0)));
        this.v = this.v.deleteCharAt(index);
        return this;
    }

    public StringBuffer avm_replace(int start, int end, String str) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_replace, java.lang.Math.max(internalLength() - start, 0)));
        this.v = this.v.replace(start, end, str.getUnderlying());
        return this;
    }

    public String avm_substring(int start) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_substring, java.lang.Math.max(internalLength() - start, 0)));
        return new String(this.v.substring(start));
    }

    public CharSequence avm_subSequence(int start, int end){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_subSequence, java.lang.Math.max(end - start, 0)));
        return new String(this.v.subSequence(start, end).toString());
    }

    public String avm_substring(int start, int end) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_substring_1, java.lang.Math.max(end - start, 0)));
        return new String(this.v.substring(start, end));
    }

    public StringBuffer avm_insert(int index, CharArray str, int offset,
                                            int len)
    {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_insert, (java.lang.Math.max(len, 0) + java.lang.Math.max(internalLength() - index, 0))));
        this.v.insert(index, str.getUnderlying(), offset, len);
        return this;
    }

    public StringBuffer avm_insert(int offset, IObject obj) {
        // delegating the call to avm_insert
        avm_insert(offset, String.internalValueOfObject(obj));
        return this;
    }

    public StringBuffer avm_insert(int offset, String str) {
        int lengthForBilling = (null != str)
                ? str.internalLength()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_insert_2, (lengthForBilling + java.lang.Math.max(internalLength() - offset, 0))));
        java.lang.String underlying = (null != str)
                ? str.getUnderlying()
                : null;
        this.v.insert(offset, underlying);
        return this;
    }

    public StringBuffer avm_insert(int offset, CharArray str) {
        int lengthForBilling = (null != str)
                ? str.length()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_insert_3, (lengthForBilling + java.lang.Math.max(internalLength() - offset, 0))));
        // Note the underlying value is not used since this will actually throw NPE if given null.
        this.v.insert(offset, str.getUnderlying());
        return this;
    }

    public StringBuffer avm_insert(int dstOffset, CharSequence s){
        int lengthForBilling = (null != s)
                ? s.avm_length()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_insert_4, (lengthForBilling + java.lang.Math.max(internalLength() - dstOffset, 0))));
        java.lang.String underlying = (null != s)
                ? s.avm_toString().getUnderlying()
                : null;
        this.v.insert(dstOffset, underlying);
        return this;
    }

    public StringBuffer avm_insert(int dstOffset, CharSequence s, int start, int end) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_insert_5, (java.lang.Math.max(end - start, 0) + java.lang.Math.max(internalLength() - dstOffset, 0))));
        java.lang.String underlying = (null != s)
                ? s.avm_toString().getUnderlying()
                : "null";
        this.v.insert(dstOffset, underlying.subSequence(start, end));
        return this;
    }

    public StringBuffer avm_insert(int offset, boolean b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_insert_6);
        this.v.insert(offset, b);
        return this;
    }

    public StringBuffer avm_insert(int offset, char c) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_insert_7);
        this.v.insert(offset, c);
        return this;
    }

    public StringBuffer avm_insert(int offset, int i) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_insert_8);
        this.v.insert(offset, i);
        return this;
    }

    public StringBuffer avm_insert(int offset, long l) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_insert_9);
        this.v.insert(offset, l);
        return this;
    }

    public StringBuffer avm_insert(int offset, float f) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_insert_10);
        this.v.insert(offset, f);
        return this;
    }

    public StringBuffer avm_insert(int offset, double d) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.StringBuffer_avm_insert_11);
        this.v.insert(offset, d);
        return this;
    }

    public int avm_indexOf(String str) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_indexOf, internalLength()));
        return this.v.indexOf(str.getUnderlying());
    }

    public int avm_indexOf(String str, int fromIndex) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_indexOf_1, java.lang.Math.max(internalLength() - fromIndex, 0)));
        return this.v.indexOf(str.getUnderlying(), fromIndex);
    }

    public int avm_lastIndexOf(String str) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_lastIndexOf, internalLength()));
        return this.v.lastIndexOf(str.getUnderlying());
    }

    public int avm_lastIndexOf(String str, int fromIndex) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_lastIndexOf_1, java.lang.Math.max(internalLength() - fromIndex, 0)));
        return this.v.lastIndexOf(str.getUnderlying(), fromIndex);
    }

    public StringBuffer avm_reverse() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_reverse, internalLength()));
        this.v.reverse();
        return this;
    }

    public String avm_toString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.StringBuffer_avm_toString, internalLength()));
        return new String(this);
    }

    //========================================================
    // Methods below are used by runtime and test code only!
    //========================================================
    private java.lang.StringBuffer v;

    public java.lang.StringBuffer getUnderlying() {
        return v;
    }

    // Deserializer support.
    public StringBuffer(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(StringBuffer.class, deserializer);
        
        // We serialize this as a string.
        java.lang.String simpler = CodecIdioms.deserializeString(deserializer);
        this.v = new java.lang.StringBuffer(simpler);
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(StringBuffer.class, serializer);
        
        // We serialize this as a string.
        CodecIdioms.serializeString(serializer, this.v.toString());
    }

    public int internalLength(){
        return this.v.length();
    }
    //========================================================
    // Methods below are deprecated
    //========================================================



    //========================================================
    // Methods below are excluded from shadowing
    //========================================================
}
