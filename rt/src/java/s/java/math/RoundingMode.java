package s.java.math;

import a.ObjectArray;
import i.ConstantToken;
import i.IInstrumentation;
import i.IObjectArray;
import i.ShadowClassConstantId;
import s.java.lang.Enum;
import s.java.lang.String;
import s.java.lang.Class;

import org.aion.avm.RuntimeMethodFeeSchedule;

// Note that we want to suppress the deprecation warnings since the original RoundingMode also does:  they both depend on deprecated BigDecimal constants.
@SuppressWarnings("deprecation")
public final class RoundingMode extends Enum<RoundingMode>{
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public static final RoundingMode avm_UP;

    public static final RoundingMode avm_DOWN;

    public static final RoundingMode avm_CEILING;

    public static final RoundingMode avm_FLOOR;

    public static final RoundingMode avm_HALF_UP;

    public static final RoundingMode avm_HALF_DOWN;

    public static final RoundingMode avm_HALF_EVEN;

    public static final RoundingMode avm_UNNECESSARY;

    int avm_oldMode;

    private java.math.RoundingMode v;

    private static final ObjectArray avm_$VALUES;

    private RoundingMode(s.java.lang.String s, int a, int b, java.math.RoundingMode u, ConstantToken constantToken) {
        super(s, a, constantToken);
        avm_oldMode = b;
        v = u;
    }

    public static IObjectArray avm_values(){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.RoundingMode_avm_values);
        return (ObjectArray) avm_$VALUES.clone();
    }

    public static RoundingMode avm_valueOf(String request){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.RoundingMode_avm_valueOf);
        return internalValueOf(request);
    }

    public static RoundingMode avm_valueOf(int idx){
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.RoundingMode_avm_valueOf_1);
        if (idx > 7 || idx < 0){
            throw new IllegalArgumentException("argument out of range");
        }else{
            return (RoundingMode) avm_$VALUES.get(idx);
        }
    }

    protected java.math.RoundingMode getUnderlying(){
        return v;
    }

    // Deserializer support.
    public RoundingMode(Void ignore, int readIndex) {
        super(ignore, readIndex);
        lazyLoad();
    }

    static {
        avm_UP          = new RoundingMode(new String("UP"), 0, java.math.BigDecimal.ROUND_UP,
                            java.math.RoundingMode.UP, new ConstantToken(ShadowClassConstantId.RoundingMode_avm_UP));

        avm_DOWN        = new RoundingMode(new String("DOWN"), 1, java.math.BigDecimal.ROUND_DOWN,
                            java.math.RoundingMode.DOWN, new ConstantToken(ShadowClassConstantId.RoundingMode_avm_DOWN));

        avm_CEILING     = new RoundingMode(new String("CEILING"), 2, java.math.BigDecimal.ROUND_CEILING,
                            java.math.RoundingMode.CEILING, new ConstantToken(ShadowClassConstantId.RoundingMode_avm_CEILING));

        avm_FLOOR       = new RoundingMode(new String("FLOOR"), 3, java.math.BigDecimal.ROUND_FLOOR,
                            java.math.RoundingMode.FLOOR, new ConstantToken(ShadowClassConstantId.RoundingMode_avm_FLOOR));

        avm_HALF_UP     = new RoundingMode(new String("HALF_UP"), 4, java.math.BigDecimal.ROUND_HALF_UP,
                            java.math.RoundingMode.HALF_UP, new ConstantToken(ShadowClassConstantId.RoundingMode_avm_HALF_UP));

        avm_HALF_DOWN   = new RoundingMode(new String("HALF_DOWN"), 5, java.math.BigDecimal.ROUND_HALF_DOWN,
                            java.math.RoundingMode.HALF_DOWN, new ConstantToken(ShadowClassConstantId.RoundingMode_avm_HALF_DOWN));

        avm_HALF_EVEN   = new RoundingMode(new String("HALF_EVEN"), 6, java.math.BigDecimal.ROUND_HALF_EVEN,
                            java.math.RoundingMode.HALF_EVEN, new ConstantToken(ShadowClassConstantId.RoundingMode_avm_HALF_EVEN));

        avm_UNNECESSARY = new RoundingMode(new String("UNNECESSARY"), 7, java.math.BigDecimal.ROUND_UNNECESSARY,
                            java.math.RoundingMode.UNNECESSARY, new ConstantToken(ShadowClassConstantId.RoundingMode_avm_UNNECESSARY));

        avm_$VALUES = ObjectArray.initArray(8);
        avm_$VALUES.set(0, avm_UP);
        avm_$VALUES.set(1, avm_DOWN);
        avm_$VALUES.set(2, avm_CEILING);
        avm_$VALUES.set(3, avm_FLOOR);
        avm_$VALUES.set(4, avm_HALF_UP);
        avm_$VALUES.set(5, avm_HALF_DOWN);
        avm_$VALUES.set(6, avm_HALF_EVEN);
        avm_$VALUES.set(7, avm_UNNECESSARY);
    }

    public static RoundingMode internalValueOf(String request){
        return (RoundingMode) Enum.internalValueOf(new Class<>(RoundingMode.class), request);
    }

}