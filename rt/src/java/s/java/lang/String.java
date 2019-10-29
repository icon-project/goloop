package s.java.lang;

import a.ByteArray;
import a.CharArray;
import i.CodecIdioms;
import i.IInstrumentation;
import i.IObject;
import i.IObjectDeserializer;
import i.IObjectSerializer;

import java.nio.charset.StandardCharsets;

import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.io.Serializable;

public final class String extends Object implements Comparable<String>, CharSequence, Serializable {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public String() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_constructor);
        this.v = new java.lang.String();
    }

    public String(String original) {
        // Initialization is done in constant time and the new string holds a reference to the original values
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_constructor_1);
        this.v = new java.lang.String(original.getUnderlying());
    }

    public String(CharArray value) {
        int lengthForBilling = (null != value)
                ? value.length()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_constructor_2 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * lengthForBilling);
        this.v = new java.lang.String(value.getUnderlying());
    }

    public String(CharArray value, int offset, int count) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_constructor_3 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.max(count, 0));
        this.v = new java.lang.String(value.getUnderlying(), offset, count);
    }

    public String(ByteArray bytes, int offset, int length){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_constructor_7 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.max(length, 0));
        this.v = new java.lang.String(bytes.getUnderlying(), offset, length);
    }

    public String(ByteArray bytes){
        int lengthForBilling = (null != bytes)
                ? bytes.length()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_constructor_8 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * lengthForBilling);
        this.v = new java.lang.String(bytes.getUnderlying());
    }

    public String(StringBuffer buffer){
        int lengthForBilling = (null != buffer)
                ? buffer.internalLength()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_constructor_9 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * lengthForBilling);
        this.v = new java.lang.String(buffer.getUnderlying());
    }

    public String(StringBuilder builder) {
        int lengthForBilling = (null != builder)
                ? builder.internalLength()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_constructor_10 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * lengthForBilling);
        this.v = new java.lang.String(builder.getUnderlying());
    }

    public int avm_length(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_length);
        return internalLength();
    }

    public boolean avm_isEmpty() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_isEmpty);
        lazyLoad();
        return v.isEmpty();
    }

    public char avm_charAt(int index) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_charAt);
        lazyLoad();
        return this.v.charAt(index);
    }

    public void avm_getChars(int srcBegin, int srcEnd, CharArray dst, int dstBegin) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_getChars + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.max(srcEnd - srcBegin, 0));
        lazyLoad();
        this.v.getChars(srcBegin, srcEnd, dst.getUnderlying(), dstBegin);
    }

    public ByteArray avm_getBytes(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_getBytes_1 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        return new ByteArray(this.v.getBytes(StandardCharsets.UTF_8));
    }

    public boolean avm_equals(IObject anObject) {
        int otherLength = (anObject instanceof String)
                ? ((String) anObject).internalLength()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_equals + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.min(otherLength, internalLength()));
        if (!(anObject instanceof String)){
            return false;
        }

        String toComp = (String) anObject;
        toComp.lazyLoad();
        this.lazyLoad();

        return this.v.equals(toComp.v);
    }

    public boolean avm_contentEquals(StringBuffer sb) {
        int otherLength = (sb != null)
                ? sb.internalLength()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_contentEquals + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.min(otherLength, internalLength()));
        lazyLoad();
        return this.v.contentEquals(sb.getUnderlying());
    }

    public boolean avm_contentEquals(CharSequence cs){
        int otherLength = (null != cs)
                ? cs.avm_length()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_contentEquals_1 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.min(otherLength, internalLength()));
        lazyLoad();
        return this.v.contentEquals(cs.avm_toString().getUnderlying());
    }

    public boolean avm_equalsIgnoreCase(String anotherString) {
        int otherLength = (anotherString != null)
                ? anotherString.internalLength()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_equalsIgnoreCase + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.min(otherLength, internalLength()));
        lazyLoad();
        java.lang.String underlying = (null != anotherString)
                ? anotherString.getUnderlying()
                : null;
        return this.v.equalsIgnoreCase(underlying);
    }

    public int avm_compareTo(String anotherString) {
        int otherLength = (anotherString != null)
                ? anotherString.internalLength()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_compareTo + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.min(otherLength, internalLength()));
        lazyLoad();
        return this.v.compareTo(anotherString.getUnderlying());
    }

    public int avm_compareToIgnoreCase(String str){
        int otherLength = (str != null)
                ? str.internalLength()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_compareToIgnoreCase + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.min(otherLength, internalLength()));
        lazyLoad();
        return this.v.compareToIgnoreCase(str.v);
    }

    public boolean avm_regionMatches(int toffset, String other, int ooffset, int len) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_regionMatches + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.max(len, 0));
        lazyLoad();
        return this.v.regionMatches(toffset, other.v, ooffset, len);
    }

    public boolean avm_regionMatches(boolean ignoreCase, int toffset, String other, int ooffset, int len) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_regionMatches_1 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.max(len, 0));
        lazyLoad();
        return this.v.regionMatches(ignoreCase, toffset, other.v, ooffset, len);
    }

    public boolean avm_startsWith(String prefix, int toffset) {
        int lengthForBilling = (null != prefix)
                ? prefix.internalLength()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_startsWith + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * lengthForBilling);
        lazyLoad();
        return this.v.startsWith(prefix.v, toffset);
    }

    public boolean avm_startsWith(String prefix) {
        int lengthForBilling = (null != prefix)
                ? prefix.internalLength()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_startsWith_1 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * lengthForBilling);
        lazyLoad();
        return this.v.startsWith(prefix.v);
    }

    public boolean avm_endsWith(String prefix) {
        int lengthForBilling = (null != prefix)
                ? prefix.internalLength()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_endsWith + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * lengthForBilling);
        lazyLoad();
        return this.v.endsWith(prefix.v);
    }

    @Override
    public int avm_hashCode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_hashCode + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        return this.v.hashCode();
    }

    public int avm_indexOf(int ch) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_indexOf + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        return this.v.indexOf(ch);
    }

    public int avm_indexOf(int ch, int fromIndex) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_indexOf_1 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.max(internalLength() - fromIndex, 0));
        lazyLoad();
        return this.v.indexOf(ch, fromIndex);
    }

    public int avm_lastIndexOf(int ch) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_lastIndexOf + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        return this.v.lastIndexOf(ch);
    }

    public int avm_lastIndexOf(int ch, int fromIndex) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_lastIndexOf_1 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.max(internalLength() - fromIndex, 0));
        lazyLoad();
        return this.v.lastIndexOf(ch, fromIndex);
    }

    public int avm_indexOf(String str) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_indexOf_2 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        str.lazyLoad();
        return this.v.indexOf(str.v);
    }

    public int avm_lastIndexOf(String str) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_lastIndexOf_2 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        str.lazyLoad();
        return this.v.lastIndexOf(str.v);
    }

    public int avm_lastIndexOf(String str, int fromIndex) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_lastIndexOf_3 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.max(internalLength() - fromIndex, 0));
        lazyLoad();
        str.lazyLoad();
        return this.v.lastIndexOf(str.v, fromIndex);
    }

    public String avm_substring(int beginIndex) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_substring + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.max(internalLength() - beginIndex, 0));
        lazyLoad();
        return new String(this.v.substring(beginIndex));
    }

    public String avm_substring(int beginIndex, int endIndex) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_substring_1 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.max(endIndex - beginIndex, 0));
        lazyLoad();
        return new String(this.v.substring(beginIndex, endIndex));
    }

    public CharSequence avm_subSequence(int beginIndex, int endIndex){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_subSequence + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.max(endIndex - beginIndex, 0));
        lazyLoad();
        return new String(this.v.subSequence(beginIndex, endIndex).toString());
    }

    public String avm_concat(String str){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_concat + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * (str.internalLength() + internalLength()));
        lazyLoad();
        str.lazyLoad();
        return new String(this.v.concat(str.v));
    }

    public String avm_replace(char oldChar, char newChar) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_replace + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        return new String(this.v.replace(oldChar, newChar));
    }

    public boolean avm_matches(String regex){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_matches + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        regex.lazyLoad();
        return this.v.matches(regex.v);
    }

    public boolean avm_contains(CharSequence s){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_contains + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        ((Object)s).lazyLoad();
        return this.v.indexOf(s.avm_toString().getUnderlying()) >= 0;
    }

    public String avm_replaceFirst(String regex, String replacement){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_replaceFirst + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        regex.lazyLoad();
        replacement.lazyLoad();
        return new String(this.v.replaceFirst(regex.v, replacement.v));
    }

    public String avm_replaceAll(String regex, String replacement) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_replaceAll + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        regex.lazyLoad();
        replacement.lazyLoad();
        return new String(this.v.replaceAll(regex.v, replacement.v));
    }

    public String avm_replace(CharSequence target, CharSequence replacement){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_replace_1 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        ((Object)target).lazyLoad();
        ((Object)replacement).lazyLoad();
        return new String(this.v.replace(target.avm_toString().getUnderlying(),
                replacement.avm_toString().getUnderlying()));
    }

    //public String[] split(String regex, int limit) {}

    //public String[] split(String regex){}

    public String avm_toLowerCase(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_toLowerCase + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        return new String(this.v.toLowerCase());
    }

    public String avm_toUpperCase(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_toUpperCase + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        return new String(this.v.toUpperCase());
    }

    public String avm_trim() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_trim + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        return new String(this.v.trim());
    }

    public String avm_toString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_toString);
        return this;
    }

    public CharArray avm_toCharArray() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_toCharArray + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * internalLength());
        lazyLoad();
        return new CharArray(this.v.toCharArray());
    }


    public static String avm_valueOf(IObject obj) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_valueOf);
        // We don't want to use the java.lang.String version of this since it relies on calling toString(), but we need avm_toString().
        return internalValueOfObject(obj);
    }

    public static String avm_valueOf(CharArray a){
        int lengthForBilling = (null != a)
                ? a.length()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_valueOf_1 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * lengthForBilling);
        a.lazyLoad();
        return new String(java.lang.String.valueOf(a.getUnderlying()));
    }

    public static String avm_valueOf(CharArray data, int offset, int count){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_valueOf_2 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.max(count, 0));
        data.lazyLoad();
        return new String(java.lang.String.valueOf(data.getUnderlying(), offset, count));
    }

    public static String avm_copyValueOf(CharArray data, int offset, int count){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_copyValueOf + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * java.lang.Math.max(count, 0));
        data.lazyLoad();
        return new String(java.lang.String.copyValueOf(data.getUnderlying(), offset, count));
    }

    public static String avm_copyValueOf(CharArray a){
        int lengthForBilling = (null != a)
                ? a.length()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_copyValueOf_1 + RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2 * lengthForBilling);
        a.lazyLoad();
        return new String(java.lang.String.copyValueOf(a.getUnderlying()));
    }

    public static String avm_valueOf(boolean b){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_valueOf_3);
        return new String(java.lang.String.valueOf(b));
    }

    public static String avm_valueOf(char b){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_valueOf_4);
        return new String(java.lang.String.valueOf(b));
    }

    public static String avm_valueOf(int b){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_valueOf_5);
        return new String(java.lang.String.valueOf(b));
    }

    public static String avm_valueOf(long b){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_valueOf_6);
        return new String(java.lang.String.valueOf(b));
    }

    public static String avm_valueOf(float b){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_valueOf_7);
        return new String(java.lang.String.valueOf(b));
    }

    public static String avm_valueOf(double b){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.String_avm_valueOf_8);
        return new String(java.lang.String.valueOf(b));
    }

    //========================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    private java.lang.String v;

    // @Internal
    public String(java.lang.String underlying) {
        this.v = underlying;
    }

    // Deserializer support.
    public String(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(String.class, deserializer);
        this.v = CodecIdioms.deserializeString(deserializer);
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(String.class, serializer);
        CodecIdioms.serializeString(serializer, this.v);
    }

    @Override
    public boolean equals(java.lang.Object anObject) {
        return anObject instanceof String && this.v.equals(((String) anObject).v);
    }

    @Override
    public int hashCode() {
        lazyLoad();
        // We probably want a consistent hashCode answer, for strings, since they are data-defined.
        return this.v.hashCode();
    }

    // NOTE:  This toString() cannot be called by the contract code (it will call avm_toString()) but our runtime and test code can call this.
    @Override
    public java.lang.String toString() {
        lazyLoad();
        return this.v;
    }

    //internal
    public java.lang.String getUnderlying(){
        lazyLoad();
        return v;
    }

    public int internalLength(){
        lazyLoad();
        return v.length();
    }

    public static String internalValueOfObject(IObject obj){
        return (null != obj)
                ? obj.avm_toString()
                : new String("null");
    }

    //========================================================
    // Methods below are deprecated, we don't shadow them
    //========================================================

    //public String(byte ascii[], int hibyte, int offset, int count)

    //public String(byte ascii[], int hibyte)

    //public void getBytes(int srcBegin, int srcEnd, byte dst[], int dstBegin)


    //========================================================
    // Methods below are excluded from shadowing
    //========================================================

    //public String(byte bytes[], int offset, int length, Charset charset)

    //public String(byte bytes[], Charset charset)

    //public byte[] getBytes(Charset charset)

    //public static final Comparator<String> CASE_INSENSITIVE_ORDER

    //public static String join(CharSequence delimiter, CharSequence... elements)

    //public static String join(CharSequence delimiter, Iterable<? extends CharSequence> elements)

    //public String toLowerCase(Locale locale)

    //public String toUpperCase(Locale locale)

    //public IntStream chars()

    //public IntStream codePoints()

    //public static String format(Locale l, String format, Object... args) {

    //public String avm_intern()

    //public static String format(String format, Object... args)

}
