package p.avm;

import a.ByteArray;
import i.IObject;
import org.aion.avm.EnergyCalculator;
import s.java.lang.Object;
import i.IInstrumentation;
import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.lang.String;


public final class Result extends Object {

    private boolean success;

    private ByteArray returnData;

    public Result(boolean success, ByteArray returnData) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Result_avm_constructor);
        this.success = success;
        this.returnData = returnData;
    }

    public boolean avm_isSuccess() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Result_avm_isSuccess);
        return success;
    }

    public ByteArray avm_getReturnData() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Result_avm_getReturnData);
        return returnData;
    }

    @Override
    public String avm_toString() {
        int lengthForBilling = null != this.returnData
                ? this.returnData.length()
                : 0;
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.Result_avm_toString, lengthForBilling));
        lazyLoad();
        String returnDataString = (null != this.returnData)
                ? toHexString(this.returnData.getUnderlying())
                : null;
        return new String("success:" + this.success + ", returnData:" + returnDataString);
    }

    private static String toHexString(byte[] bytes) {
        int length = bytes.length;

        char[] hexChars = new char[length * 2];
        for (int i = 0; i < length; i++) {
            int v = bytes[i] & 0xFF;
            hexChars[i * 2] = hexArray[v >>> 4];
            hexChars[i * 2 + 1] = hexArray[v & 0x0F];
        }
        return new String(new java.lang.String(hexChars));
    }

    private static final char[] hexArray = "0123456789abcdef".toCharArray();

    @Override
    public boolean avm_equals(IObject obj) {
        int lengthForBilling = (obj instanceof Result && ((Result) obj).returnData != null && this.returnData != null)
                ? java.lang.Math.min(((Result) obj).returnData.length(), this.returnData.length())
                : 0;
        // Billing is done similar to Arrays.equals()
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.Result_avm_equals, lengthForBilling));
        boolean isEqual = this == obj;
        if (!isEqual && (obj instanceof Result)) {
            Result other = (Result) obj;
            lazyLoad();
            other.lazyLoad();

            if (this.returnData == null && other.returnData == null) {
                isEqual = true;
            } else if (this.returnData != null && other.returnData != null) {
                isEqual = returnData.equals(other.returnData);
            } else {
                isEqual = false;
            }

            isEqual = isEqual && (this.success == other.success);
        }
        return isEqual;
    }

    @Override
    public int avm_hashCode() {
        int lengthForBilling = this.returnData != null
                ?this.returnData.getUnderlying().length
                :0;
        // Billing is done similar to Arrays.hashcode()
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(EnergyCalculator.multiplyLinearValueByMethodFeeLevel2AndAddBase(RuntimeMethodFeeSchedule.Result_avm_hashCode, lengthForBilling));
        lazyLoad();
        // Just a really basic implementation.
        int code = 0;
        if (this.returnData != null) {
            for (byte elt : this.returnData.getUnderlying()) {
                code += (int) elt;
            }
        }

        code += this.success ? 1 : 0;

        return code;
    }
}
