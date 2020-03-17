/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package score;

/**
 * Represents an address of account in the ICON Network.
 */
public class Address {

    /**
     * The length of an address.
     */
    public static final int LENGTH = 21;

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
     * Create an Address from the hex string format
     *
     * @param str a hex string that represents an Address
     */
    public static Address fromString(String str) {
        if (str == null) {
            throw new NullPointerException();
        }
        if (str.length() != LENGTH * 2) {
            throw new IllegalArgumentException();
        }
        if (str.startsWith("hx") || str.startsWith("cx")) {
            byte[] bytes = new byte[LENGTH];
            bytes[0] = (byte) (str.startsWith("hx") ? 0x0 : 0x1);
            for (int i = 1; i < LENGTH; i++) {
                int j = i * 2;
                bytes[i] = (byte) Integer.parseInt(str.substring(j, j + 2), 16);
            }
            return new Address(bytes);
        } else {
            throw new IllegalArgumentException();
        }
    }

    public boolean isContract() {
        return this.raw[0] == 0x1;
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
    public String toString() {
        byte prefix = this.raw[0];
        byte[] body = new byte[LENGTH - 1];
        System.arraycopy(this.raw, 1, body, 0, body.length);
        return ((prefix == 0x0) ? "hx" : "cx") + toHexString(body);
    }

    private static String toHexString(byte[] bytes) {
        char[] hexChars = new char[bytes.length * 2];
        for (int i = 0; i < bytes.length; i++) {
            int v = bytes[i] & 0xFF;
            hexChars[i * 2] = hexArray[v >>> 4];
            hexChars[i * 2 + 1] = hexArray[v & 0x0F];
        }
        return new java.lang.String(hexChars);
    }

    private static final char[] hexArray = "0123456789abcdef".toCharArray();
}
