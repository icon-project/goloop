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

    /**
     * Creates an address with the contents of the given raw byte array.
     *
     * @param raw a byte array
     * @throws NullPointerException if the input byte array is null
     * @throws IllegalArgumentException if the input byte array length is invalid
     */
    public Address(byte[] raw) throws IllegalArgumentException {
    }

    /**
     * Creates an address from the hex string format.
     *
     * @param str a hex string that represents an Address
     * @return the resulting address
     * @throws NullPointerException if the input string is null
     * @throws IllegalArgumentException if the input string format or length is invalid
     */
    public static Address fromString(String str) {
        return null;
    }

    /**
     * Returns true if and only if this address represents a contract address.
     *
     * @return true if this address represents a contract address, false otherwise
     */
    public boolean isContract() {
        return false;
    }

    /**
     * Converts this address to a new byte array.
     *
     * @return a newly allocated byte array that represents this address
     */
    public byte[] toByteArray() {
        return null;
    }

    /**
     * Returns a hash code for this address.
     *
     * @return a hash code value for this object
     */
    @Override
    public int hashCode() {
        return 0;
    }

    /**
     * Compares this address to the specified object.
     *
     * @param obj the object to compare this address against
     * @return true if the given object represents an {@code Address} equivalent to this address, false otherwise
     */
    @Override
    public boolean equals(Object obj) {
        return false;
    }

    /**
     * Returns a string representation of this address.
     *
     * @return a string representation of this object
     */
    @Override
    public String toString() {
        return null;
    }
}
