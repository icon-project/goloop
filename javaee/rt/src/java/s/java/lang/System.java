package s.java.lang;

import a.Array;
import i.IInstrumentation;
import i.IObject;
import org.aion.avm.EnergyCalculator;
import org.aion.avm.RuntimeMethodFeeSchedule;


public final class System extends Object{
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    private System() {
    }

    public static void avm_arraycopy(IObject src,  int  srcPos,
                                     IObject dest, int destPos,
                                     int length)
    {
        EnergyCalculator.chargeEnergyLevel1(RuntimeMethodFeeSchedule.System_avm_arraycopy, java.lang.Math.max(length, 0));
        if(src == null || dest == null){
            throw new NullPointerException();
        } else if (!((src instanceof Array) && (dest instanceof Array))){
            throw new ArrayStoreException();
        }else{
            java.lang.Object asrc = ((Array) src).getUnderlyingAsObject();
            java.lang.Object adst = ((Array) dest).getUnderlyingAsObject();
            java.lang.System.arraycopy(asrc, srcPos, adst, destPos, length);
            ((Array) dest).setUnderlyingAsObject(adst);
        }
    }
}
