package org.aion.avm;

import i.IInstrumentation;

/**
 * This class performs the linear fee calculation for JCL classes.
 */
public class EnergyCalculator {

    /**
     * @param base base cost
     * @param linearValue linear cost
     * @return base + linearValue * RT_METHOD_FEE_FACTOR_LEVEL_2
     */
    private static long multiplyLinearValueByMethodFeeLevel2AndAddBase(int base, int linearValue) {
        return add(base, multiply(Math.max(linearValue, 0), RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2));
    }

    /**
     * @param base base cost
     * @param linearValue linear cost
     * @return base + linearValue * RT_METHOD_FEE_FACTOR_LEVEL_1
     */
    private static long multiplyLinearValueByMethodFeeLevel1AndAddBase(int base, int linearValue) {
        return add(base, multiply(Math.max(linearValue, 0), RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_1));
    }

    private static long multiply(int value1, int value2) {
        return (long) value1 * (long) value2;
    }

    private static long add(int value1, long value2) {
        return (long) value1 + value2;
    }

    public static void chargeEnergy(int fixedFee) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(fixedFee);
    }

    public static void chargeEnergyLevel1(int base, int lengthForBilling) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                multiplyLinearValueByMethodFeeLevel1AndAddBase(base, lengthForBilling));
    }

    public static void chargeEnergyLevel2(int base, int lengthForBilling) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(
                multiplyLinearValueByMethodFeeLevel2AndAddBase(base, lengthForBilling));
    }

    public static void chargeEnergyLevel2(int base, int oldLen, int newLen) {
        var inst = IInstrumentation.attachedThreadInstrumentation.get();
        var es = inst.getFrameContext().getExternalState();
        int lengthForBilling = (es.fixJCLSteps()) ? newLen : oldLen;
        inst.chargeEnergy(multiplyLinearValueByMethodFeeLevel2AndAddBase(base, lengthForBilling));
    }

    public static void chargeEnergyMultiply(int base, int value1, int value2) {
        long cost = base + multiply(value1, value2);
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(cost);
    }

    public static void chargeEnergyClone(int base, int length, int perElementFee) {
        var es = IInstrumentation.getCurrentFrameContext().getExternalState();
        if (es.fixJCLSteps()) {
            chargeEnergyMultiply(base, length, perElementFee);
        } else {
            chargeEnergyLevel2(base, length);
        }
    }

    public static void chargeEnergyForIndexOf(int base, int thisLen, int tgtLen, int fromIndex) {
        var es = IInstrumentation.getCurrentFrameContext().getExternalState();
        if (es.fixJCLSteps()) {
            int sourceLen = Math.max(thisLen - fromIndex, 0);
            int valueLen = Math.max(sourceLen - tgtLen, 0);
            if (valueLen > 0) {
                chargeEnergyMultiply(base, valueLen, tgtLen);
            } else {
                chargeEnergyMultiply(base, sourceLen, 1);
            }
        } else {
            chargeEnergyLevel2(base, Math.max(thisLen - fromIndex, 0));
        }
    }

    public static void chargeEnergyForLastIndexOf(int base, int thisLen, int tgtLen) {
        var es = IInstrumentation.getCurrentFrameContext().getExternalState();
        if (es.fixJCLSteps()) {
            chargeEnergyForLastIndexOf(base, thisLen, tgtLen, thisLen);
        } else {
            chargeEnergyLevel2(base, thisLen);
        }
    }

    public static void chargeEnergyForLastIndexOf(int base, int thisLen, int tgtLen, int fromIndex) {
        var es = IInstrumentation.getCurrentFrameContext().getExternalState();
        if (es.fixJCLSteps()) {
            int rightIndex = thisLen - tgtLen;
            int valueLen = (fromIndex > rightIndex)
                    ? Math.max(rightIndex, 0)
                    : Math.max(fromIndex, 0);
            if (valueLen > 0) {
                chargeEnergyMultiply(base, valueLen, tgtLen);
            } else {
                chargeEnergyMultiply(base, thisLen, 1);
            }
        } else {
            chargeEnergyLevel2(base, Math.max(thisLen - fromIndex, 0));
        }
    }

    public static void chargeEnergyForReplace(int base, int thisLen, int tgtLen, int replLen) {
        var es = IInstrumentation.getCurrentFrameContext().getExternalState();
        if (es.fixJCLSteps()) {
            int maxCount = 1;
            long newLenHint;
            if (tgtLen == 0) {
                newLenHint = thisLen + (long) (thisLen + 1) * replLen;
            } else {
                // thisLen >= 0 && tgtLen > 0
                if (thisLen >= tgtLen && replLen > tgtLen) {
                    maxCount = thisLen / tgtLen;
                    newLenHint = replLen;
                } else {
                    newLenHint = thisLen;
                }
            }
            long cost = base + newLenHint * maxCount * RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2;
            IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(cost);
        } else {
            chargeEnergyLevel2(base, thisLen);
        }
    }
}
