package s.java.util;

import a.ByteArray;
import i.IInstrumentation;
import org.aion.avm.EnergyCalculator;
import s.java.lang.Object;

import org.aion.avm.RuntimeMethodFeeSchedule;

// The JCL doesn't force this to be final but we might want to do that to our implementation.
public class Arrays extends Object {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    private Arrays() {}

    public static int avm_hashCode(ByteArray a) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.Arrays_avm_hashCode, ((a == null) ? 0 : a.length())));
        if (a == null) {
            return 0;
        } else {
            return java.util.Arrays.hashCode(a.getUnderlying());
        }
    }

    public static boolean avm_equals(ByteArray a, ByteArray a2) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.Arrays_avm_equals, ((a == null || a2 == null) ? 0 : Math.min(a.length(), a2.length()))));
        if (a == a2) {
            return true;
        }

        if (a == null || a2 == null) {
            return false;
        }

        return java.util.Arrays.equals(a.getUnderlying(), a2.getUnderlying());
    }

    public static ByteArray avm_copyOfRange(ByteArray a, int start, int end) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.Arrays_avm_copyOfRange, Math.max(end - start, 0)));
        return new ByteArray(java.util.Arrays.copyOfRange(a.getUnderlying(), start, end));
    }

    public static void avm_fill(ByteArray a, int fromIndex, int toIndex, byte val) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.Arrays_avm_fill, Math.max(toIndex - fromIndex, 0)));
        java.util.Arrays.fill(a.getUnderlying(), fromIndex, toIndex, val);
    }
}
