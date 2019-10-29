package org.aion.avm;

/**
 * This class performs the linear fee calculation for JCL classes.
 * If the product of values overflows, Integer.MAX_VALUE is returned.
 */
public class EnergyCalculator {

    /**
     * @param base base cost
     * @param linearValue linear cost
     * @return base + linearValue * RT_METHOD_FEE_FACTOR_LEVEL_2
     */
    public static int multiplyLinearValueByMethodFeeLevel2AndAddBase(int base, int linearValue) {
        return addAndCheckForOverflow(base, multiplyAndCheckForOverflow(linearValue, RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2));
    }

    /**
     * @param base base cost
     * @param linearValue linear cost
     * @return base + linearValue * RT_METHOD_FEE_FACTOR_LEVEL_1
     */
    public static int multiplyLinearValueByMethodFeeLevel1AndAddBase(int base, int linearValue) {
        return addAndCheckForOverflow(base, multiplyAndCheckForOverflow(linearValue, RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_1));
    }

    public static int multiply(int value1, int value2) {
        return multiplyAndCheckForOverflow(value1, value2);
    }

    private static int addAndCheckForOverflow(int value1, int value2) {
        long result = (long) value1 + (long) value2;
        if (result > Integer.MAX_VALUE) {
            return Integer.MAX_VALUE;
        } else {
            return (int) result;
        }
    }

    private static int multiplyAndCheckForOverflow(int value1, int value2) {
        long result = (long) value1 * (long) value2;
        if (result > Integer.MAX_VALUE) {
            return Integer.MAX_VALUE;
        } else {
            return (int) result;
        }
    }
}
