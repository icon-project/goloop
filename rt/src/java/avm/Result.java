package avm;

import java.util.Arrays;

/**
 * Represents an cross-call invocation result.
 */
public class Result {

    private boolean success;

    private byte[] returnData;

    /**
     * Creates an instance.
     *
     * @param success    whether the invocation is success or not.
     * @param returnData the return data
     */
    public Result(boolean success, byte[] returnData) {
        this.success = success;
        this.returnData = returnData;
    }

    /**
     * Returns whether the invocation is success or not.
     *
     * @return true if success
     */
    public boolean isSuccess() {
        return success;
    }

    /**
     * Returns the data returned by the invoked dapp.
     *
     * @return a byte array, may be NULL
     */
    public byte[] getReturnData() {
        return returnData;
    }

    @Override
    public String toString() {
        String returnDataString = (null != this.returnData)
                ? toHexString(this.returnData)
                : null;
        return "success:" + this.success + ", returnData:" + returnDataString;
    }

    private static String toHexString(byte[] bytes) {
        int length = bytes.length;

        char[] hexChars = new char[length * 2];
        for (int i = 0; i < length; i++) {
            int v = bytes[i] & 0xFF;
            hexChars[i * 2] = hexArray[v >>> 4];
            hexChars[i * 2 + 1] = hexArray[v & 0x0F];
        }
        return new java.lang.String(hexChars);
    }

    private static final char[] hexArray = "0123456789abcdef".toCharArray();

    @Override
    public boolean equals(Object obj) {
        boolean isEqual = this == obj;
        if (!isEqual && (obj instanceof Result)) {
            Result other = (Result) obj;
            isEqual = (this.success == other.success)
                    && Arrays.equals(this.returnData, other.returnData);
        }
        return isEqual;
    }

    @Override
    public int hashCode() {
        // Just a really basic implementation.
        int code = 0;
        for (byte elt : this.returnData) {
            code += (int)elt;
        }

        code += this.success ? 1 : 0;

        return code;
    }
}
