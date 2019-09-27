package s.java.util.concurrent;

import org.aion.avm.RuntimeMethodFeeSchedule;
import a.ObjectArray;
import i.ConstantToken;
import i.IInstrumentation;
import i.IObjectArray;
import i.ShadowClassConstantId;
import s.java.lang.Class;
import s.java.lang.Enum;
import s.java.lang.String;

public final class TimeUnit extends Enum<TimeUnit> {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    public static final TimeUnit avm_DAYS;
    public static final TimeUnit avm_HOURS;
    public static final TimeUnit avm_MINUTES;
    public static final TimeUnit avm_SECONDS;
    public static final TimeUnit avm_MILLISECONDS;
    public static final TimeUnit avm_MICROSECONDS;
    public static final TimeUnit avm_NANOSECONDS;

    private java.util.concurrent.TimeUnit v;

    private static final ObjectArray avm_$VALUES;

    private TimeUnit(s.java.lang.String name, int ordinal, java.util.concurrent.TimeUnit u, ConstantToken constantToken) {
        super(name, ordinal, constantToken);
        v = u;
    }

    public long avm_convert(long sourceDuration, TimeUnit sourceUnit) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.TimeUnit_avm_convert);
        return this.v.convert(sourceDuration, sourceUnit.v);
    }

    public long avm_toDays(long duration) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.TimeUnit_avm_toDays);
        return this.v.toDays(duration);
    }

    public long avm_toHours(long duration) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.TimeUnit_avm_toHours);
        return this.v.toHours(duration);
    }

    public long avm_toMinutes(long duration) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.TimeUnit_avm_toMinutes);
        return this.v.toMinutes(duration);
    }

    public long avm_toSeconds(long duration) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.TimeUnit_avm_toSeconds);
        return this.v.toSeconds(duration);
    }

    public long avm_toMillis(long duration) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.TimeUnit_avm_toMillis);
        return this.v.toMillis(duration);
    }

    public long avm_toMicros(long duration) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.TimeUnit_avm_toMicros);
        return this.v.toMicros(duration);
    }

    public long avm_toNanos(long duration) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.TimeUnit_avm_toNanos);
        return this.v.toNanos(duration);
    }

    public static IObjectArray avm_values() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.TimeUnit_avm_values);
        return (ObjectArray) avm_$VALUES.clone();
    }

    public static TimeUnit avm_valueOf(String request) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.TimeUnit_avm_valueOf);
        return Enum.internalValueOf(new Class<>(TimeUnit.class), request);
    }

    protected java.util.concurrent.TimeUnit getUnderlying() {
        return v;
    }

    // Deserializer support.
    public TimeUnit(Void ignore, int readIndex) {
        super(ignore, readIndex);
        lazyLoad();
    }

    static {
        avm_DAYS = new TimeUnit(new String("DAYS"), 0, java.util.concurrent.TimeUnit.DAYS,
                new ConstantToken(ShadowClassConstantId.TimeUnit_avm_DAYS));
        avm_HOURS = new TimeUnit(new String("HOURS"), 1, java.util.concurrent.TimeUnit.HOURS,
                new ConstantToken(ShadowClassConstantId.TimeUnit_avm_HOURS));
        avm_MINUTES = new TimeUnit(new String("MINUTES"), 2, java.util.concurrent.TimeUnit.MINUTES,
                new ConstantToken(ShadowClassConstantId.TimeUnit_avm_MINUTES));
        avm_SECONDS = new TimeUnit(new String("SECONDS"), 3, java.util.concurrent.TimeUnit.SECONDS,
                new ConstantToken(ShadowClassConstantId.TimeUnit_avm_SECONDS));
        avm_MILLISECONDS = new TimeUnit(new String("MILLISECONDS"), 4, java.util.concurrent.TimeUnit.MILLISECONDS,
                new ConstantToken(ShadowClassConstantId.TimeUnit_avm_MILLISECONDS));
        avm_MICROSECONDS = new TimeUnit(new String("MICROSECONDS"), 5, java.util.concurrent.TimeUnit.MICROSECONDS,
                new ConstantToken(ShadowClassConstantId.TimeUnit_avm_MICROSECONDS));
        avm_NANOSECONDS = new TimeUnit(new String("NANOSECONDS"), 6, java.util.concurrent.TimeUnit.NANOSECONDS,
                new ConstantToken(ShadowClassConstantId.TimeUnit_avm_NANOSECONDS));

        avm_$VALUES = ObjectArray.initArray(7);
        avm_$VALUES.set(0, avm_DAYS);
        avm_$VALUES.set(1, avm_HOURS);
        avm_$VALUES.set(2, avm_MINUTES);
        avm_$VALUES.set(3, avm_SECONDS);
        avm_$VALUES.set(4, avm_MILLISECONDS);
        avm_$VALUES.set(5, avm_MICROSECONDS);
        avm_$VALUES.set(6, avm_NANOSECONDS);
    }
}
