package a;

import i.IInstrumentation;
import i.IObject;
import org.aion.avm.EnergyCalculator;
import s.java.lang.Cloneable;
import s.java.lang.Object;


public abstract class Array extends Object implements Cloneable, IArray {
    // Initial creation.
    public Array() {
    }

    // Deserializer support.
    public Array(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public abstract java.lang.Object getUnderlyingAsObject();

    public abstract void setUnderlyingAsObject(java.lang.Object u);

    public abstract java.lang.Object getAsObject(int idx);

    public abstract int length();

    public abstract IObject avm_clone();

    /**
     * Note that this helper exists primarily to calculate the energy cost for initArray operation.
     * Energy charged equals length * perElementFee
     *
     * @param length        length of the array.
     * @param perElementFee energy to be charged per element depending on type.
     */
    static protected void chargeEnergyInitArray(int length, int perElementFee) {
        long cost = EnergyCalculator.multiply(Math.max(length, 0), perElementFee);
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(cost);
    }

    /**
     * Note that this helper exists primarily so it can be called by generated/instrumented code
     * to calculate and charge energy for array clone operation.
     * Energy charged equals baseFee + length * RT_METHOD_FEE_FACTOR_LEVEL_2
     * Since the array code is not generally in the same class loader of the DApp, it can't call the runtime class, directly.
     *
     * @param baseFee cloning base fee
     * @param length  length of array
     */
    static protected void chargeEnergyClone(int baseFee, int length) {
        long cost = EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(baseFee, length);
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(cost);
    }
}
