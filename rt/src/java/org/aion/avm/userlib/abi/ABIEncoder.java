package org.aion.avm.userlib.abi;

import avm.Address;

import java.math.BigInteger;

/**
 * Utility class for AVM ABI encoding.
 *
 * <p>This class provides static helpers for encoding single data elements.
 * <p>It is typically more appropriate to use {@link ABIStreamingEncoder}.
 */
public final class ABIEncoder {
    /**
     * This class cannot be instantiated.
     */
    private ABIEncoder(){}

    private static final int BYTE_MASK = 0xff;

    /**
     * Encodes one byte as a serialized extent.
     * @param data one byte.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneByte(byte data) {
        byte[] result = new byte[Byte.BYTES + 1];
        result[0] = ABIToken.BYTE;
        result[1] = data;
        return result;
    }

    /**
     * Encodes one boolean as a serialized extent.
     * @param data one boolean.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneBoolean(boolean data) {
        byte[] result = new byte[Byte.BYTES + 1];
        result[0] = ABIToken.BOOLEAN;
        result[1] = (byte)(data ? 1 : 0);
        return result;
    }

    /**
     * Encodes one character as a serialized extent.
     * @param data one character.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneCharacter(char data) {
        byte[] result = new byte[Character.BYTES + 1];
        result[0] = ABIToken.CHAR;
        result[1]  = (byte) ((data >> 8) & BYTE_MASK);
        result[2] = (byte) (data & BYTE_MASK);
        return result;
    }

    /**
     * Encodes one short as a serialized extent.
     * @param data one short.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneShort(short data) {
        byte[] result = new byte[Short.BYTES + 1];
        result[0] = ABIToken.SHORT;
        result[1]  = (byte) ((data >> 8) & BYTE_MASK);
        result[2] = (byte) (data & BYTE_MASK);
        return result;
    }

    /**
     * Encodes one integer as a serialized extent.
     * @param data one integer.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneInteger(int data) {
        byte[] result = new byte[Integer.BYTES + 1];
        result[0] = ABIToken.INT;
        result[1]  = (byte) ((data >> 24) & BYTE_MASK);
        result[2]  = (byte) ((data >> 16) & BYTE_MASK);
        result[3]  = (byte) ((data >> 8) & BYTE_MASK);
        result[4] = (byte) (data & BYTE_MASK);
        return result;
    }

    /**
     * Encodes one long as a serialized extent.
     * @param data one long.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneLong(long data) {
        byte[] result = new byte[Long.BYTES + 1];
        result[0] = ABIToken.LONG;
        result[1]  = (byte) ((data >> 56) & BYTE_MASK);
        result[2]  = (byte) ((data >> 48) & BYTE_MASK);
        result[3]  = (byte) ((data >> 40) & BYTE_MASK);
        result[4]  = (byte) ((data >> 32) & BYTE_MASK);
        result[5]  = (byte) ((data >> 24) & BYTE_MASK);
        result[6]  = (byte) ((data >> 16) & BYTE_MASK);
        result[7]  = (byte) ((data >> 8) & BYTE_MASK);
        result[8] = (byte) (data & BYTE_MASK);
        return result;
    }

    /**
     * Encodes float byte as a serialized extent.
     * @param data one float.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneFloat(float data) {
        byte[] result = new byte[Float.BYTES + 1];
        int dataBits = Float.floatToIntBits(data);
        result[0] = ABIToken.FLOAT;
        result[1]  = (byte) ((dataBits >> 24) & BYTE_MASK);
        result[2]  = (byte) ((dataBits >> 16) & BYTE_MASK);
        result[3]  = (byte) ((dataBits >> 8) & BYTE_MASK);
        result[4] = (byte) (dataBits & BYTE_MASK);
        return result;
    }

    /**
     * Encodes one double as a serialized extent.
     * @param data one double.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneDouble(double data) {
        byte[] result = new byte[Double.BYTES + 1];
        long dataBits = Double.doubleToLongBits(data);
        result[0] = ABIToken.DOUBLE;
        result[1]  = (byte) ((dataBits >> 56) & BYTE_MASK);
        result[2]  = (byte) ((dataBits >> 48) & BYTE_MASK);
        result[3]  = (byte) ((dataBits >> 40) & BYTE_MASK);
        result[4]  = (byte) ((dataBits >> 32) & BYTE_MASK);
        result[5]  = (byte) ((dataBits >> 24) & BYTE_MASK);
        result[6]  = (byte) ((dataBits >> 16) & BYTE_MASK);
        result[7]  = (byte) ((dataBits >> 8) & BYTE_MASK);
        result[8] = (byte) (dataBits & BYTE_MASK);
        return result;
    }

    /**
     * Encodes one byte array as a serialized extent.
     * Null is encoded as the two identifiers: NULL, followed by A_BYTE.
     * @param data one byte array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneByteArray(byte[] data) {
        byte[] result;
        if (null == data) {
            result = new byte[2];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.A_BYTE;
        } else {
            if (data.length > Short.MAX_VALUE) {
                throw new ABIException("Array length must fit in 2 bytes");
            }
            result = new byte[data.length + Short.BYTES + 1];
            result[0] = ABIToken.A_BYTE;
            result[1] = (byte) ((data.length >> 8) & BYTE_MASK);
            result[2] = (byte) (data.length & BYTE_MASK);
            System.arraycopy(data, 0, result, 3, data.length);
        }
        return result;
    }

    /**
     * Encodes one boolean array as a serialized extent.
     * Null is encoded as the two identifiers: NULL, followed by A_BOOLEAN.
     * @param data one boolean array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneBooleanArray(boolean[] data) {
        byte[] result;
        if (null == data) {
            result = new byte[2];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.A_BOOLEAN;
        } else {
            if (data.length > Short.MAX_VALUE) {
                throw new ABIException("Array length must fit in 2 bytes");
            }
            result = new byte[data.length + Short.BYTES + 1];
            result[0] = ABIToken.A_BOOLEAN;
            result[1] = (byte) ((data.length >> 8) & BYTE_MASK);
            result[2] = (byte) (data.length & BYTE_MASK);
            for (int i = 0, j = 3; i < data.length; i++, j++) {
                result[j] = (byte)(data[i] ? 1 : 0);
            }
        }
        return result;
    }

    /**
     * Encodes one character array as a serialized extent.
     * Null is encoded as the two identifiers: NULL, followed by A_CHAR.
     * @param data one character array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneCharacterArray(char[] data) {
        byte[] result;
        if (null == data) {
            result = new byte[2];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.A_CHAR;
        } else {
            if (data.length > Short.MAX_VALUE) {
                throw new ABIException("Array length must fit in 2 bytes");
            }
            result = new byte[data.length * Character.BYTES + Short.BYTES + 1];
            result[0] = ABIToken.A_CHAR;
            result[1] = (byte) ((data.length >> 8) & BYTE_MASK);
            result[2] = (byte) (data.length & BYTE_MASK);
            for (int i = 0, j = 3; i < data.length; i++, j+=Character.BYTES) {
                result[j]  = (byte) ((data[i] >> 8) & BYTE_MASK);
                result[j+1]  = (byte) (data[i] & BYTE_MASK);
            }
        }
        return result;
    }

    /**
     * Encodes one short array as a serialized extent.
     * Null is encoded as the two identifiers: NULL, followed by A_SHORT.
     * @param data one short array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneShortArray(short[] data) {
        byte[] result;
        if (null == data) {
            result = new byte[2];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.A_SHORT;
        } else {
            if (data.length > Short.MAX_VALUE) {
                throw new ABIException("Array length must fit in 2 bytes");
            }
            result = new byte[data.length * Short.BYTES + Short.BYTES + 1];
            result[0] = ABIToken.A_SHORT;
            result[1] = (byte) ((data.length >> 8) & BYTE_MASK);
            result[2] = (byte) (data.length & BYTE_MASK);
            for (int i = 0, j = 3; i < data.length; i++, j+=Short.BYTES) {
                result[j]  = (byte) ((data[i] >> 8) & BYTE_MASK);
                result[j+1]  = (byte) (data[i] & BYTE_MASK);
            }
        }
        return result;
    }

    /**
     * Encodes one integer array as a serialized extent.
     * Null is encoded as the two identifiers: NULL, followed by A_INT.
     * @param data one integer array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneIntegerArray(int[] data) {
        byte[] result;
        if (null == data) {
            result = new byte[2];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.A_INT;
        } else {
            if (data.length > Short.MAX_VALUE) {
                throw new ABIException("Array length must fit in 2 bytes");
            }
            result = new byte[data.length * Integer.BYTES + Short.BYTES + 1];
            result[0] = ABIToken.A_INT;
            result[1] = (byte) ((data.length >> 8) & BYTE_MASK);
            result[2] = (byte) (data.length & BYTE_MASK);
            for (int i = 0, j = 3; i < data.length; i++, j+=Integer.BYTES) {
                result[j]  = (byte) ((data[i] >> 24) & BYTE_MASK);
                result[j+1]  = (byte) ((data[i] >> 16) & BYTE_MASK);
                result[j+2]  = (byte) ((data[i] >> 8) & BYTE_MASK);
                result[j+3] = (byte) (data[i] & BYTE_MASK);
            }
        }
        return result;
    }

    /**
     * Encodes one long array as a serialized extent.
     * Null is encoded as the two identifiers: NULL, followed by A_LONG.
     * @param data one long array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneLongArray(long[] data) {
        byte[] result;
        if (null == data) {
            result = new byte[2];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.A_LONG;
        } else {
            if (data.length > Short.MAX_VALUE) {
                throw new ABIException("Array length must fit in 2 bytes");
            }
            result = new byte[data.length * Long.BYTES + Short.BYTES + 1];
            result[0] = ABIToken.A_LONG;
            result[1] = (byte) ((data.length >> 8) & BYTE_MASK);
            result[2] = (byte) (data.length & BYTE_MASK);
            for (int i = 0, j = 3; i < data.length; i++, j+=Long.BYTES) {
                result[j]  = (byte) ((data[i] >> 56) & BYTE_MASK);
                result[j+1]  = (byte) ((data[i] >> 48) & BYTE_MASK);
                result[j+2]  = (byte) ((data[i] >> 40) & BYTE_MASK);
                result[j+3]  = (byte) ((data[i] >> 32) & BYTE_MASK);
                result[j+4]  = (byte) ((data[i] >> 24) & BYTE_MASK);
                result[j+5]  = (byte) ((data[i] >> 16) & BYTE_MASK);
                result[j+6]  = (byte) ((data[i] >> 8) & BYTE_MASK);
                result[j+7]  = (byte) (data[i] & BYTE_MASK);
            }
        }
        return result;
    }

    /**
     * Encodes one float array as a serialized extent.
     * Null is encoded as the two identifiers: NULL, followed by A_FLOAT.
     * @param data one float array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneFloatArray(float[] data) {
        byte[] result;
        if (null == data) {
            result = new byte[2];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.A_FLOAT;
        } else {
            if (data.length > Short.MAX_VALUE) {
                throw new ABIException("Array length must fit in 2 bytes");
            }
            result = new byte[data.length * Float.BYTES + Short.BYTES + 1];
            result[0] = ABIToken.A_FLOAT;
            result[1] = (byte) ((data.length >> 8) & BYTE_MASK);
            result[2] = (byte) (data.length & BYTE_MASK);
            for (int i = 0, j = 3; i < data.length; i++, j+=Float.BYTES) {
                int dataBits = Float.floatToIntBits(data[i]);
                result[j]  = (byte) ((dataBits >> 24) & BYTE_MASK);
                result[j+1]  = (byte) ((dataBits >> 16) & BYTE_MASK);
                result[j+2]  = (byte) ((dataBits >> 8) & BYTE_MASK);
                result[j+3]  = (byte) (dataBits & BYTE_MASK);
            }
        }
        return result;
    }

    /**
     * Encodes one double array as a serialized extent.
     * Null is encoded as the two identifiers: NULL, followed by A_DOUBLE.
     * @param data one double array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneDoubleArray(double[] data) {
        byte[] result;
        if (null == data) {
            result = new byte[2];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.A_DOUBLE;
        } else {
            if (data.length > Short.MAX_VALUE) {
                throw new ABIException("Array length must fit in 2 bytes");
            }
            result = new byte[data.length * Double.BYTES + Short.BYTES + 1];
            result[0] = ABIToken.A_DOUBLE;
            result[1] = (byte) ((data.length >> 8) & BYTE_MASK);
            result[2] = (byte) (data.length & BYTE_MASK);
            for (int i = 0, j = 3; i < data.length; i++, j+=Double.BYTES) {
                long dataBits = Double.doubleToLongBits(data[i]);
                result[j]  = (byte) ((dataBits >> 56) & BYTE_MASK);
                result[j+1]  = (byte) ((dataBits >> 48) & BYTE_MASK);
                result[j+2]  = (byte) ((dataBits >> 40) & BYTE_MASK);
                result[j+3]  = (byte) ((dataBits >> 32) & BYTE_MASK);
                result[j+4]  = (byte) ((dataBits >> 24) & BYTE_MASK);
                result[j+5]  = (byte) ((dataBits >> 16) & BYTE_MASK);
                result[j+6]  = (byte) ((dataBits >> 8) & BYTE_MASK);
                result[j+7]  = (byte) (dataBits & BYTE_MASK);
            }
        }
        return result;
    }

    /**
     * Encodes one String as a serialized extent.
     * Null is encoded as the two identifiers: NULL, followed by STRING.
     * @param data one string.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneString(String data) {
        byte[] result;
        if (null == data) {
            result = new byte[2];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.STRING;
        } else {
            byte[] stringBytes = data.getBytes();
            result = new byte[stringBytes.length + Short.BYTES + 1];
            result[0] = ABIToken.STRING;
            result[1] = (byte) ((stringBytes.length >> 8) & BYTE_MASK);
            result[2] = (byte) (stringBytes.length & BYTE_MASK);
            System.arraycopy(stringBytes, 0, result, 3, stringBytes.length);
        }
        return result;
    }

    /**
     * Encodes one Address as a serialized extent.
     * Null is encoded as the two identifiers: NULL, followed by ADDRESS.
     * @param data one address.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneAddress(Address data) {
        byte[] result;
        if (null == data) {
            result = new byte[2];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.ADDRESS;
        } else {
            byte[] addressBytes = data.toByteArray();
            if(Address.LENGTH != addressBytes.length) {
                throw new ABIException("Address was of unexpected length");
            }
            result = new byte[Address.LENGTH + 1];
            result[0] = ABIToken.ADDRESS;
            System.arraycopy(addressBytes, 0, result, 1, addressBytes.length);
        }
        return result;
    }

    /**
     * Encodes one BigInteger as a serialized extent
     * Null is encoded as the two identifiers: NULL, followed by BIGINT.
     *
     * @param data one BigInteger.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneBigInteger(BigInteger data) {
        byte[] result;
        if (null == data) {
            result = new byte[2];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.BIGINT;
        } else {
            byte[] bigIntegerBytes = data.toByteArray();
            // maximum size of a BigInteger value accepted by AVM is 32 bytes
            if (bigIntegerBytes.length > 32) {
                throw new ABIException("BigInteger value exceeds the limit of 32 bytes");
            }

            int length = bigIntegerBytes.length;
            result = new byte[length + 1 + 1];
            result[0] = ABIToken.BIGINT;
            result[1] = (byte) length;

            System.arraycopy(bigIntegerBytes, 0, result, 2, length);
        }
        return result;
    }

    /**
     * Encodes one 2D byte array as a serialized extent.
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_BYTE.
     * @param data one 2D byte array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOne2DByteArray(byte[][] data) {
        byte[] result;
        if (null == data) {
            result = new byte[3];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.ARRAY;
            result[2] = ABIToken.A_BYTE;
        } else {
            int length = Short.BYTES + 2;
            byte[][] encodedArrays = new byte[data.length][];
            for (int i = 0; i < data.length; i++) {
                encodedArrays[i] = encodeOneByteArray(data[i]);
                length += encodedArrays[i].length;
            }
            result = flatten2DEncoding(encodedArrays, length, ABIToken.A_BYTE);
        }
        return result;
    }

    /**
     * Encodes one 2D boolean array as a serialized extent.
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_BOOLEAN.
     * @param data one 2D boolean array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOne2DBooleanArray(boolean[][] data) {
        byte[] result;
        if (null == data) {
            result = new byte[3];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.ARRAY;
            result[2] = ABIToken.A_BOOLEAN;
        } else {
            int length = Short.BYTES + 2;
            byte[][] encodedArrays = new byte[data.length][];
            for (int i = 0; i < data.length; i++) {
                encodedArrays[i] = encodeOneBooleanArray(data[i]);
                length += encodedArrays[i].length;
            }
            result = flatten2DEncoding(encodedArrays, length, ABIToken.A_BOOLEAN);
        }
        return result;
    }

    /**
     * Encodes one 2D character array as a serialized extent.
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_CHAR.
     * @param data one 2D character array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOne2DCharacterArray(char[][] data) {
        byte[] result;
        if (null == data) {
            result = new byte[3];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.ARRAY;
            result[2] = ABIToken.A_CHAR;
        } else {
            int length = Short.BYTES + 2;
            byte[][] encodedArrays = new byte[data.length][];
            for (int i = 0; i < data.length; i++) {
                encodedArrays[i] = encodeOneCharacterArray(data[i]);
                length += encodedArrays[i].length;
            }
            result = flatten2DEncoding(encodedArrays, length, ABIToken.A_CHAR);
        }
        return result;
    }

    /**
     * Encodes one 2D short array as a serialized extent.
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_SHORT.
     * @param data one 2D short array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOne2DShortArray(short[][] data) {
        byte[] result;
        if (null == data) {
            result = new byte[3];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.ARRAY;
            result[2] = ABIToken.A_SHORT;
        } else {
            int length = Short.BYTES + 2;
            byte[][] encodedArrays = new byte[data.length][];
            for (int i = 0; i < data.length; i++) {
                encodedArrays[i] = encodeOneShortArray(data[i]);
                length += encodedArrays[i].length;
            }
            result = flatten2DEncoding(encodedArrays, length, ABIToken.A_SHORT);
        }
        return result;
    }

    /**
     * Encodes one 2D integer array as a serialized extent.
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_INT.
     * @param data one 2D integer array
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOne2DIntegerArray(int[][] data) {
        byte[] result;
        if (null == data) {
            result = new byte[3];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.ARRAY;
            result[2] = ABIToken.A_INT;
        } else {
            int length = Short.BYTES + 2;
            byte[][] encodedArrays = new byte[data.length][];
            for (int i = 0; i < data.length; i++) {
                encodedArrays[i] = encodeOneIntegerArray(data[i]);
                length += encodedArrays[i].length;
            }
            result = flatten2DEncoding(encodedArrays, length, ABIToken.A_INT);
        }
        return result;
    }

    /**
     * Encodes one 2D float array as a serialized extent.
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_FLOAT.
     * @param data one 2D float array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOne2DFloatArray(float[][] data) {
        byte[] result;
        if (null == data) {
            result = new byte[3];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.ARRAY;
            result[2] = ABIToken.A_FLOAT;
        } else {
            int length = Short.BYTES + 2;
            byte[][] encodedArrays = new byte[data.length][];
            for (int i = 0; i < data.length; i++) {
                encodedArrays[i] = encodeOneFloatArray(data[i]);
                length += encodedArrays[i].length;
            }
            result = flatten2DEncoding(encodedArrays, length, ABIToken.A_FLOAT);
        }
        return result;
    }

    /**
     * Encodes one 2D long array as a serialized extent.
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_LONG.
     * @param data one 2D long array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOne2DLongArray(long[][] data) {
        byte[] result;
        if (null == data) {
            result = new byte[3];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.ARRAY;
            result[2] = ABIToken.A_LONG;
        } else {
            int length = Short.BYTES + 2;
            byte[][] encodedArrays = new byte[data.length][];
            for (int i = 0; i < data.length; i++) {
                encodedArrays[i] = encodeOneLongArray(data[i]);
                length += encodedArrays[i].length;
            }
            result = flatten2DEncoding(encodedArrays, length, ABIToken.A_LONG);
        }
        return result;
    }

    /**
     * Encodes one 2D double array as a serialized extent.
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_DOUBLE.
     * @param data one 2D double array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOne2DDoubleArray(double[][] data) {
        byte[] result;
        if (null == data) {
            result = new byte[3];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.ARRAY;
            result[2] = ABIToken.A_DOUBLE;
        } else {
            int length = Short.BYTES + 2;
            byte[][] encodedArrays = new byte[data.length][];
            for (int i = 0; i < data.length; i++) {
                encodedArrays[i] = encodeOneDoubleArray(data[i]);
                length += encodedArrays[i].length;
            }
            result = flatten2DEncoding(encodedArrays, length, ABIToken.A_DOUBLE);
        }
        return result;
    }

    /**
     * Encodes one String array as a serialized extent.
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by STRING.
     * @param data one string array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneStringArray(String[] data) {
        byte[] result;
        if (null == data) {
            result = new byte[3];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.ARRAY;
            result[2] = ABIToken.STRING;
        } else {
            int length = Short.BYTES + 2;
            byte[][] encodedArrays = new byte[data.length][];
            for (int i = 0; i < data.length; i++) {
                encodedArrays[i] = encodeOneString(data[i]);
                length += encodedArrays[i].length;
            }
            result = flatten2DEncoding(encodedArrays, length, ABIToken.STRING);
        }
        return result;
    }

    /**
     * Encodes one Address array as a serialized extent.
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by ADDRESS.
     * @param data one address array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneAddressArray(Address[] data) {
        byte[] result;
        if (null == data) {
            result = new byte[3];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.ARRAY;
            result[2] = ABIToken.ADDRESS;
        } else {
            int length = Short.BYTES + 2;
            byte[][] encodedArrays = new byte[data.length][];
            for (int i = 0; i < data.length; i++) {
                encodedArrays[i] = encodeOneAddress(data[i]);
                length += encodedArrays[i].length;
            }
            result = flatten2DEncoding(encodedArrays, length, ABIToken.ADDRESS);
        }
        return result;
    }

    /**
     * Encodes one BigInteger array as a serialized extent.
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by BIGINT.
     *
     * @param data one BigInteger array.
     * @return the byte array that contains the argument descriptor and the encoded data.
     */
    public static byte[] encodeOneBigIntegerArray(BigInteger[] data) {
        byte[] result;
        if (null == data) {
            result = new byte[3];
            result[0] = ABIToken.NULL;
            result[1] = ABIToken.ARRAY;
            result[2] = ABIToken.BIGINT;
        } else {
            int length = Short.BYTES + 2;
            byte[][] encodedArrays = new byte[data.length][];
            for (int i = 0; i < data.length; i++) {
                encodedArrays[i] = encodeOneBigInteger(data[i]);
                length += encodedArrays[i].length;
            }
            result = flatten2DEncoding(encodedArrays, length, ABIToken.BIGINT);
        }
        return result;
    }

    private static byte[] flatten2DEncoding(byte[][] encodedArrays, int lengthOfEncodedResult, byte identifier) {
        byte[] result = new byte[lengthOfEncodedResult];
        result[0] = ABIToken.ARRAY;
        result[1] = identifier;
        result[2] = (byte) ((encodedArrays.length >> 8) & BYTE_MASK);
        result[3] = (byte) (encodedArrays.length & BYTE_MASK);
        int destPos = 4;
        for (int i = 0; i < encodedArrays.length; i++) {
            System.arraycopy(encodedArrays[i], 0, result, destPos, encodedArrays[i].length);
            destPos += encodedArrays[i].length;
        }
        return result;
    }
}
