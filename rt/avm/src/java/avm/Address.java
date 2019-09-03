package avm;

/**
 * Represents an address of account in the Aion Network.
 */
public class Address {

    /**
     * The length of an address.
     */
    public static final int LENGTH = 32;

    private final byte[] raw = new byte[LENGTH];

    /**
     * Create an Address with the contents of the given raw byte array.
     *
     * @param raw a byte array
     * @throws NullPointerException when the input byte array is null.
     * @throws IllegalArgumentException when the input byte array length is invalid.
     */
    public Address(byte[] raw) throws IllegalArgumentException {
        if (raw == null) {
            throw new NullPointerException();
        }
        if (raw.length != LENGTH) {
            throw new IllegalArgumentException();
        }
        System.arraycopy(raw, 0, this.raw, 0, LENGTH);
    }

    /**
     * Converts the receiver to a new byte array.
     *
     * @return a byte array containing a copy of the receiver.
     */
    public byte[] toByteArray() {
        byte[] copy = new byte[LENGTH];
        System.arraycopy(this.raw, 0, copy, 0, LENGTH);
        return copy;
    }

    @Override
    public int hashCode() {
        // Just a really basic implementation.
        int code = 0;
        for (byte elt : this.raw) {
            code += (int) elt;
        }
        return code;
    }

    @Override
    public boolean equals(Object obj) {
        boolean isEqual = this == obj;
        if (!isEqual && (obj instanceof Address)) {
            Address other = (Address) obj;
            isEqual = true;
            for (int i = 0; isEqual && (i < LENGTH); ++i) {
                isEqual = (this.raw[i] == other.raw[i]);
            }
        }
        return isEqual;
    }

    @Override
    public java.lang.String toString() {
        return toHexStringForAPI(this.raw);
    }

    private static java.lang.String toHexStringForAPI(byte[] bytes) {
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
}
