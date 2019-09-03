package org.aion.avm.userlib;

import java.math.BigInteger;
import java.nio.BufferOverflowException;
import java.nio.BufferUnderflowException;

import avm.Address;


/**
 * A buffer, much like an NIO ByteBuffer, which allows the easy encoding/decoding of primitive values.
 */
public class AionBuffer {
    private static final int BYTE_MASK = 0xff;
    private static final int BYTE_SIZE = Byte.SIZE;
    // Note that AVM forces all BigInteger to 32-bytes, so we will serialize them as this fixed-size value.
    private static final int BIG_INTEGER_BYTES = 32;

    private final byte[] buffer;
    private int position;
    private int limit;

    private AionBuffer(byte[] array) {
        this.buffer = array;
        this.position = 0;
        this.limit = array.length;
    }

    /**
     * Creates a new AionBuffer instance with the given capacity.
     * @param capacity The size of the underlying buffer to create, in bytes.
     * @return The new AionBuffer instance.
     */
    public static AionBuffer allocate(int capacity) {
        if (capacity < 1) {
            throw new IllegalArgumentException("Illegal capacity: " + capacity);
        }
        return new AionBuffer(new byte[capacity]);
    }

    /**
     * Creates a new AionBuffer instance wrapping the given byte array.
     * @param array The array to wrap.
     * @return The new AionBuffer instance.
     */
    public static AionBuffer wrap(byte[] array) {
        if (array == null) {
            throw new NullPointerException();
        }
        if (array.length < 1) {
            throw new IllegalArgumentException("Illegal capacity: " + array.length);
        }
        return new AionBuffer(array);
    }

    // ====================
    // relative get methods
    // ====================

    /**
     * Populates the given dst buffer with the next bytes in the buffer and advances the position.
     * Note that dst MUST not be larger than the rest of the bytes in the receiver.
     * @param dst The byte array to populate with the contents of the receiver.
     * @return The receiver (for call chaining).
     */
    public AionBuffer get(byte[] dst) {
        if (dst == null) {
            throw new NullPointerException();
        }
        int remaining = this.limit - this.position;
        if (remaining < dst.length) {
            throw new BufferUnderflowException();
        }
        System.arraycopy(this.buffer, this.position, dst, 0, dst.length);
        this.position += dst.length;
        return this;
    }

    /**
     * Returns the next boolean in the buffer and advances the position.
     * Note that we store booleans as a 1-byte quantity (0x1 or 0x0).
     * @return The underlying byte, interpreted as a boolean (0x1 is true).
     */
    public boolean getBoolean() {
        byte value = internalGetByte();
        return (0x1 == value);
    }

    /**
     * Returns the next byte in the buffer and advances the position.
     * @return The byte.
     */
    public byte getByte() {
        return internalGetByte();
    }

    private byte internalGetByte() {
        int remaining = this.limit - this.position;
        if (remaining < Byte.BYTES) {
            throw new BufferUnderflowException();
        }
        byte b = this.buffer[this.position];
        this.position += Byte.BYTES;
        return b;
    }

    /**
     * Returns the next char in the buffer and advances the position.
     * @return The char.
     */
    public char getChar() {
        return (char) getShort();
    }

    /**
     * Returns the next double in the buffer and advances the position.
     * @return The double.
     */
    public double getDouble() {
        return Double.longBitsToDouble(getLong());
    }

    /**
     * Returns the next float in the buffer and advances the position.
     * @return The float.
     */
    public float getFloat() {
        return Float.intBitsToFloat(getInt());
    }

    /**
     * Returns the next int in the buffer and advances the position.
     * @return The int.
     */
    public int getInt() {
        int remaining = this.limit - this.position;
        if (remaining < Integer.BYTES) {
            throw new BufferUnderflowException();
        }
        int i = this.buffer[this.position] << BYTE_SIZE;
        i = (i | (this.buffer[this.position + 1] & BYTE_MASK)) << BYTE_SIZE;
        i = (i | (this.buffer[this.position + 2] & BYTE_MASK)) << BYTE_SIZE;
        i |= (this.buffer[this.position + 3] & BYTE_MASK);
        this.position += Integer.BYTES;
        return i;
    }

    /**
     * Returns the next long in the buffer and advances the position.
     * @return The long.
     */
    public long getLong() {
        int remaining = this.limit - this.position;
        if (remaining < Long.BYTES) {
            throw new BufferUnderflowException();
        }
        long l = this.buffer[this.position] << BYTE_SIZE;
        l = (l | (this.buffer[this.position + 1] & BYTE_MASK)) << BYTE_SIZE;
        l = (l | (this.buffer[this.position + 2] & BYTE_MASK)) << BYTE_SIZE;
        l = (l | (this.buffer[this.position + 3] & BYTE_MASK)) << BYTE_SIZE;
        l = (l | (this.buffer[this.position + 4] & BYTE_MASK)) << BYTE_SIZE;
        l = (l | (this.buffer[this.position + 5] & BYTE_MASK)) << BYTE_SIZE;
        l = (l | (this.buffer[this.position + 6] & BYTE_MASK)) << BYTE_SIZE;
        l |= this.buffer[this.position + 7] & BYTE_MASK;
        this.position += Long.BYTES;
        return l;
    }

    /**
     * Returns the next short in the buffer and advances the position.
     * @return The short.
     */
    public short getShort() {
        int remaining = this.limit - this.position;
        if (remaining < Short.BYTES) {
            throw new BufferUnderflowException();
        }
        short s = (short) (this.buffer[this.position] << BYTE_SIZE);
        s |= (this.buffer[this.position + 1] & BYTE_MASK);
        this.position += Short.BYTES;
        return s;
    }

    /**
     * Returns the next 32-byte Aion address in the buffer and advances the position.
     * @return The address.
     */
    public Address getAddress() {
        int remaining = this.limit - this.position;
        if (remaining < Address.LENGTH) {
            throw new BufferUnderflowException();
        }
        byte[] raw = new byte[Address.LENGTH];
        System.arraycopy(this.buffer, this.position, raw, 0, raw.length);
        this.position += raw.length;
        return new Address(raw);
    }

    /**
     * Returns the next 32-byte signed BigInteger in the buffer and advances the position.
     * @return The BigInteger.
     */
    public BigInteger get32ByteInt() {
        int remaining = this.limit - this.position;
        if (remaining < BIG_INTEGER_BYTES) {
            throw new BufferUnderflowException();
        }
        byte[] raw = new byte[BIG_INTEGER_BYTES];
        System.arraycopy(this.buffer, this.position, raw, 0, raw.length);
        this.position += raw.length;
        return new BigInteger(raw);
    }

    // ====================
    // relative put methods
    // ====================

    /**
     * Copies the bytes from src into the buffer and advances the position.
     * Note that src MUST not be larger than the rest of the bytes in the receiver.
     * @param src The bytes to copy.
     * @return The receiver (for call chaining).
     */
    public AionBuffer put(byte[] src) {
        if (src == null) {
            throw new NullPointerException();
        }
        int remaining = this.limit - this.position;
        if (remaining < src.length) {
            throw new BufferOverflowException();
        }
        System.arraycopy(src, 0, this.buffer, this.position, src.length);
        this.position += src.length;
        return this;
    }

    /**
     * Stores a boolean into the buffer and advances the position.
     * Note that we store booleans as a 1-byte quantity (0x1 or 0x0).
     * @param flag The boolean to store as a byte (0x1 for true, 0x0 for false).
     * @return The receiver (for call chaining).
     */
    public AionBuffer putBoolean(boolean flag) {
        byte b = (byte)(flag ? 0x1 : 0x0);
        return internalPutByte(b);
    }

    /**
     * Stores a byte into the buffer and advances the position.
     * @param b The byte to write.
     * @return The receiver (for call chaining).
     */
    public AionBuffer putByte(byte b) {
        return internalPutByte(b);
    }

    private AionBuffer internalPutByte(byte b) {
        int remaining = this.limit - this.position;
        if (remaining < Byte.BYTES) {
            throw new BufferOverflowException();
        }
        this.buffer[this.position] = b;
        this.position += Byte.BYTES;
        return this;
    }

    /**
     * Stores a char into the buffer and advances the position.
     * @param value The char to write.
     * @return The receiver (for call chaining).
     */
    public AionBuffer putChar(char value) {
        return putShort((short) value);
    }

    /**
     * Stores a double into the buffer and advances the position.
     * @param value The double to write.
     * @return The receiver (for call chaining).
     */
    public AionBuffer putDouble(double value) {
        return putLong(Double.doubleToLongBits(value));
    }

    /**
     * Stores a float into the buffer and advances the position.
     * @param value The float to write.
     * @return The receiver (for call chaining).
     */
    public AionBuffer putFloat(float value) {
        return putInt(Float.floatToIntBits(value));
    }

    /**
     * Stores a int into the buffer and advances the position.
     * @param value The int to write.
     * @return The receiver (for call chaining).
     */
    public AionBuffer putInt(int value) {
        int remaining = this.limit - this.position;
        if (remaining < Integer.BYTES) {
            throw new BufferOverflowException();
        }
        this.buffer[this.position] = (byte) ((value >> 24) & BYTE_MASK);
        this.buffer[this.position + 1] = (byte) ((value >> 16) & BYTE_MASK);
        this.buffer[this.position + 2] = (byte) ((value >> 8) & BYTE_MASK);
        this.buffer[this.position + 3] = (byte) (value & BYTE_MASK);
        this.position += Integer.BYTES;
        return this;
    }

    /**
     * Stores a long into the buffer and advances the position.
     * @param value The long to write.
     * @return The receiver (for call chaining).
     */
    public AionBuffer putLong(long value) {
        int remaining = this.limit - this.position;
        if (remaining < Long.BYTES) {
            throw new BufferOverflowException();
        }
        this.buffer[this.position] = (byte) ((value >> 56) & BYTE_MASK);
        this.buffer[this.position + 1] = (byte) ((value >> 48) & BYTE_MASK);
        this.buffer[this.position + 2] = (byte) ((value >> 40) & BYTE_MASK);
        this.buffer[this.position + 3] = (byte) ((value >> 32) & BYTE_MASK);
        this.buffer[this.position + 4] = (byte) ((value >> 24) & BYTE_MASK);
        this.buffer[this.position + 5] = (byte) ((value >> 16) & BYTE_MASK);
        this.buffer[this.position + 6] = (byte) ((value >> 8) & BYTE_MASK);
        this.buffer[this.position + 7] = (byte) (value & BYTE_MASK);
        this.position += Long.BYTES;
        return this;
    }

    /**
     * Stores a short into the buffer and advances the position.
     * @param value The short to write.
     * @return The receiver (for call chaining).
     */
    public AionBuffer putShort(short value) {
        int remaining = this.limit - this.position;
        if (remaining < Short.BYTES) {
            throw new BufferOverflowException();
        }
        this.buffer[this.position] = (byte) ((value >> 8) & BYTE_MASK);
        this.buffer[this.position + 1] = (byte) (value & BYTE_MASK);
        this.position += Short.BYTES;
        return this;
    }

    /**
     * Stores an Aion address into the buffer and advances the position.
     * @param value The address to write.
     * @return The receiver (for call chaining).
     */
    public AionBuffer putAddress(Address value) {
        if (value == null) {
            throw new NullPointerException();
        }
        int remaining = this.limit - this.position;
        if (remaining < Address.LENGTH) {
            throw new BufferOverflowException();
        }
        byte[] raw = value.toByteArray();
        System.arraycopy(raw, 0, this.buffer, this.position, raw.length);
        this.position += raw.length;
        return this;
    }

    /**
     * Stores a 32-byte signed BigInteger into the buffer and advances the position.
     * Note that this BigInteger is always stored as 32 bytes, even if its internal representation may be smaller.
     * @param value The BigInteger to write.
     * @return The receiver (for call chaining).
     */
    public AionBuffer put32ByteInt(BigInteger value) {
        if (value == null) {
            throw new NullPointerException();
        }
        int remaining = this.limit - this.position;
        if (remaining < BIG_INTEGER_BYTES) {
            throw new BufferOverflowException();
        }
        byte prefixByte = (-1 == value.signum())
                ? (byte)0xff
                : (byte)0x0;
        byte[] raw = value.toByteArray();
        // BigInteger instances can't be larger than 32-bytes, in AVM.
        assert (raw.length <= BIG_INTEGER_BYTES);
        // Add additional 0-bytes for any not expressed in the BigInteger (this is big-endian, so they preceed the value).
        for (int i = raw.length; i < BIG_INTEGER_BYTES; ++i) {
            internalPutByte(prefixByte);
        }
        System.arraycopy(raw, 0, this.buffer, this.position, raw.length);
        this.position += raw.length;
        return this;
    }

    // =====================
    // query & misc. methods
    // =====================

    /**
     * Allows access to the byte array under the buffer.  Note that this will be a shared instance so changes to one will be observable in the other.
     * @return The byte array underneath the receiver.
     */
    public byte[] getArray() {
        return this.buffer;
    }

    /**
     * @return The total capacity of the receiver.
     */
    public int getCapacity() {
        return this.buffer.length;
    }

    /**
     * @return The offset into the underlying byte array which the receiver will read/write its next byte.
     */
    public int getPosition() {
        return this.position;
    }

    /**
     * @return The end of the buffer which will currently be used by a read/write operation.
     */
    public int getLimit() {
        return this.limit;
    }

    /**
     * Resets the position to 0 and the limit to the full capacity of the buffer.
     * Used when discarding state associated with a previous use of the buffer.
     * 
     * @return The receiver (for call chaining).
     */
    public AionBuffer clear() {
        this.position = 0;
        this.limit = this.buffer.length;
        return this;
    }

    /**
     * Sets the limit to the current position and resets the position to 0.
     * Primarily used when switching between writing and reading modes:
     *  write(X), write(Y), write(Z), flip(), read(X), read(Y), read(Z).
     * 
     * @return The receiver (for call chaining).
     */
    public AionBuffer flip() {
        this.limit = this.position;
        this.position = 0;
        return this;
    }

    /**
     * Sets the position back to 0.
     * Useful for cases where the previously processed contents want to be reprocessed.
     * 
     * @return The receiver (for call chaining).
     */
    public AionBuffer rewind() {
        this.position = 0;
        return this;
    }

    @Override
    public boolean equals(Object ob) {
        // The standard JCL ByteBuffer derives its equality from its state and internal data so do the same, here.
        if (this == ob) {
            return true;
        }
        if (!(ob instanceof AionBuffer)) {
            return false;
        }
        AionBuffer other = (AionBuffer) ob;
        if (this.buffer.length != other.buffer.length) {
            return false;
        }
        if (this.position != other.position) {
            return false;
        }
        if (this.limit != other.limit) {
            return false;
        }
        // The comparison is not the full buffer, only up to the limit.
        for (int i = 0; i < this.limit; i++) {
            if (this.buffer[i] != other.buffer[i]) {
                return false;
            }
        }
        return true;
    }

    @Override
    public int hashCode() {
        // The standard JCL ByteBuffer derives its hash code from its internal data so do the same, here.
        int h = 1;
        // The comparison is not the full buffer, only up to the limit.
        for (int i = this.limit - 1; i >= 0; i--) {
            h = 31 * h + (int) this.buffer[i];
        }
        return h;
    }

    @Override
    public String toString() {
        return "AionBuffer [capacity = " + this.buffer.length + ", position = " + this.position + ", limit = " + this.limit + " ]";
    }
}
