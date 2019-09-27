package s.java.lang;

import i.*;
import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.io.Serializable;

public final class Boolean extends Object implements Serializable, Comparable<Boolean> {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public static final Boolean avm_TRUE = new Boolean(true, new ConstantToken(ShadowClassConstantId.Boolean_avm_TRUE));

    public static final Boolean avm_FALSE = new Boolean(false, new ConstantToken(ShadowClassConstantId.Boolean_avm_FALSE));

    public static final Class<Boolean> avm_TYPE = new Class(java.lang.Boolean.TYPE, new ConstantToken(ShadowClassConstantId.Boolean_avm_TYPE));

    // These are the constructors provided in the JDK but we mark them private since they are deprecated.
    // (in the future, we may change these to not exist - depends on the kind of error we want to give the user).
    private Boolean(boolean b) {
        this.v = b;
    }
    @SuppressWarnings("unused")
    private Boolean(String s) {
        throw RuntimeAssertionError.unimplemented("This is only provided for a consistent error to user code - not to be called");
    }

    public static boolean avm_parseBoolean(String s){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Boolean_avm_parseBoolean);
        return internalParseBoolean(s);
    }

    public boolean avm_booleanValue() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Boolean_avm_booleanValue);
        return v;
    }

    public static Boolean avm_valueOf(boolean b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Boolean_avm_valueOf);
        return b ? avm_TRUE : avm_FALSE;
    }

    public static Boolean avm_valueOf(String s) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Boolean_avm_valueOf_1);
        return internalParseBoolean(s) ? avm_TRUE : avm_FALSE;
    }

    public static String avm_toString(boolean b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Boolean_avm_toString);
        return b ? (new String("true")) : (new String("false"));
    }

    public String avm_toString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Boolean_avm_toString_1);
        return v ? (new String("true")) : (new String("false"));
    }

    @Override
    public int avm_hashCode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Boolean_avm_hashCode);
        return internalHashCode(this.v);
    }

    public static int avm_hashCode(boolean value) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Boolean_avm_hashCode_1);
        return internalHashCode(value);
    }

    public boolean avm_equals(IObject obj) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Boolean_avm_equals);
        if (obj instanceof Boolean) {
            Boolean other = (Boolean)obj;
            return this.v == other.v;
        }
        return false;
    }

    public int avm_compareTo(Boolean b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Boolean_avm_compareTo);
        return internalCompare(this.v, b.v);
    }

    public static int avm_compare(boolean x, boolean y) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Boolean_avm_compare);
        return internalCompare(x, y);
    }

    public static boolean avm_logicalAnd(boolean a, boolean b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Boolean_avm_logicalAnd);
        return a && b;
    }

    public static boolean avm_logicalOr(boolean a, boolean b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Boolean_avm_logicalOr);
        return a || b;
    }

    public static boolean avm_logicalXor(boolean a, boolean b) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Boolean_avm_logicalXor);
        return a ^ b;
    }

    private static boolean internalParseBoolean(String s){
        return (s != null) && java.lang.Boolean.parseBoolean(s.getUnderlying());
    }

    private static int internalHashCode(boolean value) {
        return value ? 1231 : 1237;
    }

    private static int internalCompare(boolean x, boolean y) {
        return (x == y) ? 0 : (x ? 1 : -1);
    }

    //=======================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    public Boolean(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
        lazyLoad();
    }

    private Boolean(boolean b, ConstantToken constantToken){
        super(constantToken);
        this.v = b;
    }
    private boolean v;

    public boolean getUnderlying() {
        return this.v;
    }

    //========================================================
    // Methods below are excluded from shadowing
    //========================================================

    //public static boolean avm_getBoolean(java.lang.String name){}

}
