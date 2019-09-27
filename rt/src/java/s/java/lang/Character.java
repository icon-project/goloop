package s.java.lang;

import i.ConstantToken;
import i.IInstrumentation;
import org.aion.avm.RuntimeMethodFeeSchedule;
import i.IObject;
import s.java.io.Serializable;
import i.ShadowClassConstantId;

public final class Character extends Object implements Serializable, Comparable<Character> {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public static final int avm_MIN_RADIX = 2;

    public static final int avm_MAX_RADIX = 36;

    public static final char avm_MIN_VALUE = '\u0000';

    public static final char avm_MAX_VALUE = '\uFFFF';

    public static final Class<Character> avm_TYPE = new Class(java.lang.Character.TYPE, new ConstantToken(ShadowClassConstantId.Character_avm_TYPE));

    // These are the constructors provided in the JDK but we mark them private since they are deprecated.
    // (in the future, we may change these to not exist - depends on the kind of error we want to give the user).
    private Character(char c) {
        this.v = c;
    }

    public static Character avm_valueOf(char c) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_valueOf);
        return new Character(c);
    }

    public char avm_charValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_charValue);
        lazyLoad();
        return v;
    }

    @Override
    public int avm_hashCode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_hashCode);
        lazyLoad();
        return internalHashCode(v);
    }

    public static int avm_hashCode(char value) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_hashCode_1);
        return internalHashCode(value);
    }

    public boolean avm_equals(IObject obj) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_equals);
        boolean isEqual = false;
        if (obj instanceof Character) {
            Character other = (Character) obj;
            lazyLoad();
            other.lazyLoad();
            isEqual = this.v == other.v;
        }
        return isEqual;
    }

    private static int internalHashCode(char value) {
        return (int)value;
    }

    public String avm_toString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_toString);
        lazyLoad();
        return new String(java.lang.Character.toString(this.v));
    }

    public static String avm_toString(char c) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_toString_1);
        return new String(java.lang.Character.toString(c));
    }

    public static boolean avm_isLowerCase(char ch){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_isLowerCase);
        return java.lang.Character.isLowerCase(ch);
    }

    public static boolean avm_isUpperCase(char ch){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_isUpperCase);
        return java.lang.Character.isUpperCase(ch);
    }

    public static boolean avm_isDigit(char ch){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_isDigit);
        return java.lang.Character.isDigit(ch);
    }

    public static boolean avm_isLetter(char ch){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_isLetter);
        return java.lang.Character.isLetter(ch);
    }

    public static boolean avm_isLetterOrDigit(char ch){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_isLetterOrDigit);
        return java.lang.Character.isLetterOrDigit(ch);
    }

    public static char avm_toLowerCase(char ch){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_toLowerCase);
        return java.lang.Character.toLowerCase(ch);
    }

    public static char avm_toUpperCase(char ch){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_toUpperCase);
        return java.lang.Character.toUpperCase(ch);
    }

    public static int avm_digit(char ch, int radix){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_digit);
        return java.lang.Character.digit(ch, radix);
    }

    public static int avm_getNumericValue(char ch){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_getNumericValue);
        return java.lang.Character.getNumericValue(ch);
    }

    public static boolean avm_isSpaceChar(char ch){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_isSpaceChar);
        return java.lang.Character.isSpaceChar(ch);
    }

    public static boolean avm_isWhitespace(char ch){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_isWhitespace);
        return java.lang.Character.isWhitespace(ch);
    }

    public static char avm_forDigit(int digit, int radix) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_forDigit);
        return java.lang.Character.forDigit(digit, radix);
    }

    public int avm_compareTo(Character anotherCharacter) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_compareTo);
        lazyLoad();
        anotherCharacter.lazyLoad();
        return this.v - anotherCharacter.v;
    }

    public static int avm_compare(char x, char y) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Character_avm_compare);
        return x - y;
    }

    public static final int avm_SIZE = java.lang.Character.SIZE;

    public static final int avm_BYTES = java.lang.Character.BYTES;

    //========================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    public Character(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    private char v;

    public char getUnderlying() {
        lazyLoad();
        return this.v;
    }

    //========================================================
    // Methods below are excluded from shadowing
    //========================================================

    // public static boolean isJavaLetter(char ch)

    // public static boolean isJavaLetterOrDigit(char ch)

    // public static boolean isSpace(char ch)

}
