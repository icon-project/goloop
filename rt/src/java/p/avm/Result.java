package p.avm;

import a.ByteArray;
import a.CharArray;
import i.IObject;
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
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Result_avm_toString);
        lazyLoad();
        return  new String("success:" + this.success + ", returnData:" + toHexString(this.returnData.getUnderlying()));
    }

    private static String toHexString(byte[] bytes) {
        int length = bytes.length;

        char[] hexChars = new char[length * 2];
        for (int i = 0; i < length; i++) {
            int v = bytes[i] & 0xFF;
            hexChars[i * 2] = hexArray[v >>> 4];
            hexChars[i * 2 + 1] = hexArray[v & 0x0F];
        }
        return new String(new CharArray(hexChars));
    }

    private static final char[] hexArray = "0123456789abcdef".toCharArray();

    @Override
    public boolean avm_equals(IObject obj) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Result_avm_equals);

        boolean isEqual = this == obj;
        if (!isEqual && (obj instanceof Result)) {
            Result other = (Result)obj;
            lazyLoad();
            other.lazyLoad();
            if (this.returnData.length() == other.returnData.length()) {
                isEqual = true;
                byte[] us = this.returnData.getUnderlying();
                byte[] them = other.returnData.getUnderlying();
                for (int i = 0; isEqual && (i < us.length); ++i) {
                    isEqual = (us[i] == them[i]);
                }
            }

            isEqual = isEqual && (this.success == other.success);
        }
        return isEqual;
    }

    @Override
    public int avm_hashCode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Result_avm_hashCode);
        lazyLoad();
        // Just a really basic implementation.
        int code = 0;
        for (byte elt : this.returnData.getUnderlying()) {
            code += (int)elt;
        }

        code += this.success ? 1 : 0;

        return code;
    }
}
