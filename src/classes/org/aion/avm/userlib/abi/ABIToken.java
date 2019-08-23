package org.aion.avm.userlib.abi;

/**
 * Identifiers of the tokens the ABI uses to describe extents of data in a serialized stream.
 */
public final  class ABIToken {
    public static final byte BYTE = (byte) 0x01;
    public static final byte BOOLEAN = (byte) 0x02;
    public static final byte CHAR = (byte) 0x03;
    public static final byte SHORT = (byte) 0x04;
    public static final byte INT = (byte) 0x05;
    public static final byte LONG = (byte) 0x06;
    public static final byte FLOAT = (byte) 0x07;
    public static final byte DOUBLE = (byte) 0x08;

    public static final byte A_BYTE = (byte) 0x11;
    public static final byte A_BOOLEAN = (byte) 0x12;
    public static final byte A_CHAR = (byte) 0x13;
    public static final byte A_SHORT = (byte) 0x14;
    public static final byte A_INT = (byte) 0x15;
    public static final byte A_LONG = (byte) 0x16;
    public static final byte A_FLOAT = (byte) 0x17;
    public static final byte A_DOUBLE = (byte) 0x18;

    public static final byte STRING = (byte) 0x21;
    public static final byte ADDRESS = (byte) 0x22;
    public static final byte BIGINT = (byte) 0x23;
    public static final byte ARRAY = (byte) 0x31;
    public static final byte NULL = (byte) 0x32;
}
