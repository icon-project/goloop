package org.aion.avm.userlib.abi;

import avm.Address;
import org.aion.avm.userlib.AionBuffer;

import java.math.BigInteger;

/**
 * Utility class for AVM ABI encoding.
 *
 * <p>Instances of this class are stateful, allowing several pieces of data to be serialized into the same buffer.
 */
public final class ABIStreamingEncoder {

    private AionBuffer buffer;

    // NULL followed by array type (e.g. A_BYTE)
    private final static int NULL_ARRAY_ONE_DIMENSION = 2;

    // NULL followed by ARRAY followed by array type (e.g. A_BYTE)
    private final static int NULL_ARRAY_TWO_DIMENSION = 3;

    /**
     * Creates a new encoder, and sets the buffer size to 64 KiB.
     */
    public ABIStreamingEncoder(){
        buffer = AionBuffer.allocate(64 * 1024);
    }

    /**
     * Creates a new encoder, which writes into the provided array. This array must be the expected
     * size of the final encoding.
     * @param array the array into which encoded bytes will be written.
     */
    public ABIStreamingEncoder(byte[] array){
        buffer = AionBuffer.wrap(array);
    }

    /**
     * Creates and returns a byte array representing everything that has been encoded so far.
     * Resets the buffer and its underlying array to an empty state.
     * @return The byte array representing everything encoded so far.
     */
    public byte[] toBytes() {
        int length = buffer.getPosition();
        byte[] encoding = new byte[length];
        System.arraycopy(buffer.getArray(), 0, encoding, 0, encoding.length);
        buffer.clear();
        return encoding;
    }

    /**
     * Encode one byte.
     * @param data one byte
     * @return the encoder with this element written into its buffer
     */
    public ABIStreamingEncoder encodeOneByte(byte data) {
        buffer.putByte(ABIToken.BYTE);
        buffer.putByte(data);
        return this;
    }

    /**
     * Encode one boolean.
     * @param data one boolean
     * @return the encoder with this element written into its buffer
     */
    public ABIStreamingEncoder encodeOneBoolean(boolean data) {
        buffer.putByte(ABIToken.BOOLEAN);
        buffer.putBoolean(data);
        return this;
    }

    /**
     * Encode one char.
     * @param data one char
     * @return the encoder with this element written into its buffer
     */
    public ABIStreamingEncoder encodeOneCharacter(char data) {
        buffer.putByte(ABIToken.CHAR);
        buffer.putChar(data);
        return this;
    }

    /**
     * Encode one short.
     * @param data one short
     * @return the encoder with this element written into its buffer
     */
    public ABIStreamingEncoder encodeOneShort(short data) {
        buffer.putByte(ABIToken.SHORT);
        buffer.putShort(data);
        return this;
    }

    /**
     * Encode one int.
     * @param data one int
     * @return the encoder with this element written into its buffer
     */
    public ABIStreamingEncoder encodeOneInteger(int data) {
        buffer.putByte(ABIToken.INT);
        buffer.putInt(data);
        return this;
    }

    /**
     * Encode one long.
     * @param data one long
     * @return the encoder with this element written into its buffer
     */
    public ABIStreamingEncoder encodeOneLong(long data) {
        buffer.putByte(ABIToken.LONG);
        buffer.putLong(data);
        return this;
    }

    /**
     * Encode one float.
     * @param data one float
     * @return the encoder with this element written into its buffer
     */
    public ABIStreamingEncoder encodeOneFloat(float data) {
        buffer.putByte(ABIToken.FLOAT);
        buffer.putFloat(data);
        return this;
    }

    /**
     * Encode one double.
     * @param data one double
     * @return the encoder with this element written into its buffer
     */
    public ABIStreamingEncoder encodeOneDouble(double data) {
        buffer.putByte(ABIToken.DOUBLE);
        buffer.putDouble(data);
        return this;
    }

    /**
     * Encode one byte array.
     * @param data one byte array
     * @return the encoder with this element written into its buffer
     */
    public ABIStreamingEncoder encodeOneByteArray(byte[] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.A_BYTE);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.A_BYTE);
            buffer.putShort((short) data.length);
            buffer.put(data);
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this byte array.
     * @param data one byte array
     * @return the length of the ABI encoding of this byte array.
     */
    public static int getLengthOfOneByteArray(byte[] data) {
        if (null == data) {
            return NULL_ARRAY_ONE_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            return 1 + Short.BYTES + data.length;
        }
    }

    /**
     * Encode one boolean array.
     * @param data one boolean array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the two identifiers: NULL, followed by A_BOOLEAN
     */
    public ABIStreamingEncoder encodeOneBooleanArray(boolean[] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.A_BOOLEAN);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.A_BOOLEAN);
            buffer.putShort((short) data.length);
            for (boolean bit : data) {
                buffer.putBoolean(bit);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this boolean array.
     * @param data one boolean array
     * @return the length of the ABI encoding of this boolean array.
     */
    public static int getLengthOfOneBooleanArray(boolean[] data) {
        if (null == data) {
            return NULL_ARRAY_ONE_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            return 1 + Short.BYTES + data.length;
        }
    }

    /**
     * Encode one char array.
     * @param data one character array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the two identifiers: NULL, followed by A_CHAR
     */
    public ABIStreamingEncoder encodeOneCharacterArray(char[] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.A_CHAR);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.A_CHAR);
            buffer.putShort((short) data.length);
            for (char c : data) {
                buffer.putChar(c);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this character array.
     * @param data one character array
     * @return the length of the ABI encoding of this character array.
     */
    public static int getLengthOfOneCharacterArray(char[] data) {
        if (null == data) {
            return NULL_ARRAY_ONE_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            return 1 + Short.BYTES + (data.length * Character.BYTES);
        }
    }

    /**
     * Encode one short array.
     * @param data one short array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the two identifiers: NULL, followed by A_SHORT
     */
    public ABIStreamingEncoder encodeOneShortArray(short[] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.A_SHORT);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.A_SHORT);
            buffer.putShort((short) data.length);
            for (short s : data) {
                buffer.putShort(s);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this short array.
     * @param data one short array
     * @return the length of the ABI encoding of this short array.
     */
    public static int getLengthOfOneShortArray(short[] data) {
        if (null == data) {
            return NULL_ARRAY_ONE_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            return 1 + Short.BYTES + (data.length * Short.BYTES);
        }
    }

    /**
     * Encode one int array.
     * @param data one integer array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the two identifiers: NULL, followed by A_INT
     */
    public ABIStreamingEncoder encodeOneIntegerArray(int[] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.A_INT);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.A_INT);
            buffer.putShort((short) data.length);
            for (int i : data) {
                buffer.putInt(i);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this integer array.
     * @param data one integer array
     * @return the length of the ABI encoding of this integer array.
     */
    public static int getLengthOfOneIntegerArray(int[] data) {
        if (null == data) {
            return NULL_ARRAY_ONE_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            return 1 + Short.BYTES + (data.length * Integer.BYTES);
        }
    }

    /**
     * Encode one long array.
     * @param data one long array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the two identifiers: NULL, followed by A_LONG
     */
    public ABIStreamingEncoder encodeOneLongArray(long[] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.A_LONG);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.A_LONG);
            buffer.putShort((short) data.length);
            for (long l : data) {
                buffer.putLong(l);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this long array.
     * @param data one long array
     * @return the length of the ABI encoding of this long array.
     */
    public static int getLengthOfOneLongArray(long[] data) {
        if (null == data) {
            return NULL_ARRAY_ONE_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            return 1 + Short.BYTES + (data.length * Long.BYTES);
        }
    }

    /**
     * Encode one float array.
     * @param data one float array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the two identifiers: NULL, followed by A_FLOAT
     */
    public ABIStreamingEncoder encodeOneFloatArray(float[] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.A_FLOAT);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.A_FLOAT);
            buffer.putShort((short) data.length);
            for (float f : data) {
                buffer.putFloat(f);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this float array.
     * @param data one float array
     * @return the length of the ABI encoding of this float array.
     */
    public static int getLengthOfOneFloatArray(float[] data) {
        if (null == data) {
            return NULL_ARRAY_ONE_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            return 1 + Short.BYTES + (data.length * Float.BYTES);
        }
    }

    /**
     * Encode one double array.
     * @param data one double array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the two identifiers: NULL, followed by A_DOUBLE
     */
    public ABIStreamingEncoder encodeOneDoubleArray(double[] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.A_DOUBLE);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.A_DOUBLE);
            buffer.putShort((short) data.length);
            for (double d : data) {
                buffer.putDouble(d);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this double array.
     * @param data one double array
     * @return the length of the ABI encoding of this double array.
     */
    public static int getLengthOfOneDoubleArray(double[] data) {
        if (null == data) {
            return NULL_ARRAY_ONE_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            return 1 + Short.BYTES + (data.length * Double.BYTES);
        }
    }

    /**
     * Encode one string.
     * @param data one string
     * @return the encoder with this element written into its buffer
     * Null is encoded as the two identifiers: NULL, followed by STRING
     */
    public ABIStreamingEncoder encodeOneString(String data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.STRING);
        } else {
            byte[] stringBytes = data.getBytes();
            checkLengthIsAShort(stringBytes.length);
            buffer.putByte(ABIToken.STRING);
            buffer.putShort((short) stringBytes.length);
            buffer.put(stringBytes);
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this String.
     * @param data one String
     * @return the length of the ABI encoding of this String.
     */
    public static int getLengthOfOneString(String data) {
        if (null == data) {
            return NULL_ARRAY_ONE_DIMENSION;
        } else {
            checkLengthIsAShort(data.length());
            return 1 + Short.BYTES + data.getBytes().length;
        }
    }

    /**
     * Encode one address.
     * @param data one address
     * @return the encoder with this element written into its buffer
     * Null is encoded as the two identifiers: NULL, followed by ADDRESS
     */
    public ABIStreamingEncoder encodeOneAddress(Address data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.ADDRESS);
        } else {
            byte[] addressBytes = data.toByteArray();
            if(Address.LENGTH != addressBytes.length) {
                throw new ABIException("Address was of unexpected length");
            }
            buffer.putByte(ABIToken.ADDRESS);
            buffer.put(addressBytes);

        }
        return this;
    }

    /**
     * Encode one BigInteger.
     *
     * @param data one BigInteger
     * @return the encoder with this element written into its buffer
     * Null is encoded as the two identifiers: NULL, followed by BIGINT
     */
    public ABIStreamingEncoder encodeOneBigInteger(BigInteger data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.BIGINT);
        } else {
            byte[] bigIntegerBytes = data.toByteArray();
            // maximum size of a BigInteger value accepted by AVM is 32 bytes
            if (bigIntegerBytes.length > 32) {
                throw new ABIException("BigInteger value exceeds the limit of 32 bytes");
            }
            buffer.putByte(ABIToken.BIGINT);
            buffer.putByte((byte) bigIntegerBytes.length);
            buffer.put(bigIntegerBytes);
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this Address.
     * @param data one Address
     * @return the length of the ABI encoding of this Address.
     */
    public static int getLengthOfOneAddress(Address data) {
        if (null == data) {
            return NULL_ARRAY_ONE_DIMENSION;
        } else {
            return 1 + Address.LENGTH;
        }
    }

    /**
     * Encode one 2D byte array.
     * @param data one 2D byte array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_BYTE
     */
    public ABIStreamingEncoder encodeOne2DByteArray(byte[][] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_BYTE);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_BYTE);
            buffer.putShort((short) data.length);

            for (byte[] array : data) {
                encodeOneByteArray(array);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this 2D byte array.
     * @param data one 2D byte array
     * @return the length of the ABI encoding of this 2D byte array.
     */
    public static int getLengthOfOne2DByteArray(byte[][] data) {
        if (null == data) {
            return NULL_ARRAY_TWO_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            int length = 2 + Short.BYTES;
            for (byte[] array : data) {
                length += getLengthOfOneByteArray(array);
            }
            return length;
        }
    }

    /**
     * Encode one 2D boolean array.
     * @param data one 2D boolean array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_BOOLEAN
     */
    public ABIStreamingEncoder encodeOne2DBooleanArray(boolean[][] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_BOOLEAN);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_BOOLEAN);
            buffer.putShort((short) data.length);

            for (boolean[] array : data) {
                encodeOneBooleanArray(array);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this 2D boolean array.
     * @param data one 2D boolean array
     * @return the length of the ABI encoding of this 2D boolean array.
     */
    public static int getLengthOfOne2DBooleanArray(boolean[][] data) {
        if (null == data) {
            return NULL_ARRAY_TWO_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            int length = 2 + Short.BYTES;
            for (boolean[] array : data) {
                length += getLengthOfOneBooleanArray(array);
            }
            return length;
        }
    }

    /**
     * Encode one 2D character array.
     * @param data one 2D character array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_CHAR
     */
    public ABIStreamingEncoder encodeOne2DCharacterArray(char[][] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_CHAR);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_CHAR);
            buffer.putShort((short) data.length);

            for (char[] array : data) {
                encodeOneCharacterArray(array);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this 2D character array.
     * @param data one 2D character array
     * @return the length of the ABI encoding of this 2D character array.
     */
    public static int getLengthOfOne2DCharacterArray(char[][] data) {
        if (null == data) {
            return NULL_ARRAY_TWO_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            int length = 2 + Short.BYTES;
            for (char[] array : data) {
                length += getLengthOfOneCharacterArray(array);
            }
            return length;
        }
    }

    /**
     * Encode one 2D short array.
     * @param data one 2D short array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_SHORT
     */
    public ABIStreamingEncoder encodeOne2DShortArray(short[][] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_SHORT);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_SHORT);
            buffer.putShort((short) data.length);

            for (short[] array : data) {
                encodeOneShortArray(array);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this 2D short array.
     * @param data one 2D short array
     * @return the length of the ABI encoding of this 2D short array.
     */
    public static int getLengthOfOne2DShortArray(short[][] data) {
        if (null == data) {
            return NULL_ARRAY_TWO_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            int length = 2 + Short.BYTES;
            for (short[] array : data) {
                length += getLengthOfOneShortArray(array);
            }
            return length;
        }
    }

    /**
     * Encode one 2D integer array.
     * @param data one 2D integer array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_INT
     */
    public ABIStreamingEncoder encodeOne2DIntegerArray(int[][] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_INT);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_INT);
            buffer.putShort((short) data.length);

            for (int[] array : data) {
                encodeOneIntegerArray(array);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this 2D integer array.
     * @param data one 2D integer array
     * @return the length of the ABI encoding of this 2D integer array.
     */
    public static int getLengthOfOne2DIntegerArray(int[][] data) {
        if (null == data) {
            return NULL_ARRAY_TWO_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            int length = 2 + Short.BYTES;
            for (int[] array : data) {
                length += getLengthOfOneIntegerArray(array);
            }
            return length;
        }
    }

    /**
     * Encode one 2D float array.
     * @param data one 2D float array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_FLOAT
     */
    public ABIStreamingEncoder encodeOne2DFloatArray(float[][] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_FLOAT);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_FLOAT);
            buffer.putShort((short) data.length);

            for (float[] array : data) {
                encodeOneFloatArray(array);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this 2D float array.
     * @param data one 2D float array
     * @return the length of the ABI encoding of this 2D float array.
     */
    public static int getLengthOfOne2DFloatArray(float[][] data) {
        if (null == data) {
            return NULL_ARRAY_TWO_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            int length = 2 + Short.BYTES;
            for (float[] array : data) {
                length += getLengthOfOneFloatArray(array);
            }
            return length;
        }
    }

    /**
     * Encode one 2D long array.
     * @param data one 2D long array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_LONG
     */
    public ABIStreamingEncoder encodeOne2DLongArray(long[][] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_LONG);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_LONG);
            buffer.putShort((short) data.length);

            for (long[] array : data) {
                encodeOneLongArray(array);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this 2D long array.
     * @param data one 2D long array
     * @return the length of the ABI encoding of this 2D long array.
     */
    public static int getLengthOfOne2DLongArray(long[][] data) {
        if (null == data) {
            return NULL_ARRAY_TWO_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            int length = 2 + Short.BYTES;
            for (long[] array : data) {
                length += getLengthOfOneLongArray(array);
            }
            return length;
        }
    }

    /**
     * Encode one 2D double array.
     * @param data one 2D double array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by A_DOUBLE
     */
    public ABIStreamingEncoder encodeOne2DDoubleArray(double[][] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_DOUBLE);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.A_DOUBLE);
            buffer.putShort((short) data.length);

            for (double[] array : data) {
                encodeOneDoubleArray(array);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this 2D double array.
     * @param data one 2D double array
     * @return the length of the ABI encoding of this 2D double array.
     */
    public static int getLengthOfOne2DDoubleArray(double[][] data) {
        if (null == data) {
            return NULL_ARRAY_TWO_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            int length = 2 + Short.BYTES;
            for (double[] array : data) {
                length += getLengthOfOneDoubleArray(array);
            }
            return length;
        }
    }

    /**
     * Encode one string array.
     * @param data one string array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by STRING
     */
    public ABIStreamingEncoder encodeOneStringArray(String[] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.STRING);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.STRING);
            buffer.putShort((short) data.length);

            for (String str : data) {
                encodeOneString(str);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this string array.
     * @param data one string array
     * @return the length of the ABI encoding of this string array.
     */
    public static int getLengthOfOneStringArray(String[] data) {
        if (null == data) {
            return NULL_ARRAY_TWO_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            int length = 2 + Short.BYTES;
            for (String str : data) {
                length += getLengthOfOneString(str);
            }
            return length;
        }
    }


    /**
     * Encode one address array.
     * @param data one address array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by ADDRESS
     */
    public ABIStreamingEncoder encodeOneAddressArray(Address[] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.ADDRESS);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.ADDRESS);
            buffer.putShort((short) data.length);

            for (Address addr : data) {
                encodeOneAddress(addr);
            }
        }
        return this;
    }

    /**
     * Encode one BigInteger array.
     *
     * @param data one BigInteger array
     * @return the encoder with this element written into its buffer
     * Null is encoded as the three identifiers: NULL, followed by ARRAY, followed by BIGINT
     */
    public ABIStreamingEncoder encodeOneBigIntegerArray(BigInteger[] data) {
        if (null == data) {
            buffer.putByte(ABIToken.NULL);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.BIGINT);
        } else {
            checkLengthIsAShort(data.length);
            buffer.putByte(ABIToken.ARRAY);
            buffer.putByte(ABIToken.BIGINT);
            buffer.putShort((short) data.length);

            for (BigInteger bigInt : data) {
                encodeOneBigInteger(bigInt);
            }
        }
        return this;
    }

    /**
     * Returns the length of the ABI encoding of this address array.
     * @param data one address array
     * @return the length of the ABI encoding of this address array.
     */
    public static int getLengthOfOneStringArray(Address[] data) {
        if (null == data) {
            return NULL_ARRAY_TWO_DIMENSION;
        } else {
            checkLengthIsAShort(data.length);
            int length = 2 + Short.BYTES;
            for (Address addr : data) {
                length += getLengthOfOneAddress(addr);
            }
            return length;
        }
    }

    private static void checkLengthIsAShort(int size) {
        if (size > Short.MAX_VALUE) {
            throw new ABIException("Array length must fit in 2 bytes");
        }
    }
}
