/*
 * Copyright 2018 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package foundation.icon.icx.data;

import org.bouncycastle.util.encoders.Hex;

import java.math.BigInteger;
import java.util.Arrays;

/**
 * A wrapper class of byte array
 */
public class Bytes {

    public static final String HEX_PREFIX = "0x";

    private final byte[] data;

    /**
     * Creates an instance using hex string
     *
     * @param hexString hex string of bytes
     */
    public Bytes(String hexString) {
        if (!isValidHex(hexString))
            throw new IllegalArgumentException("The value is not hex string.");
        this.data = Hex.decode(cleanHexPrefix(hexString));
    }

    /**
     * Creates an instance using byte array
     *
     * @param data byte array to wrap
     */
    public Bytes(byte[] data) {
        this.data = data;
    }

    /**
     * Creates an instance using BigInteger
     * <p>
     * Set a byte array of {@link BigInteger#toByteArray()} return value.
     * The array will contain the minimum number of bytes required
     * to represent this BigInteger, including at least one sign bit.
     *
     * @param value the {@linkplain java.math.BigInteger value}
     */
    public Bytes(BigInteger value) {
        this.data = value.toByteArray();
    }

    /**
     * Gets the data as a byte array
     *
     * @return byte array
     */
    public byte[] toByteArray() {
        return data;
    }

    @Deprecated
    private static byte[] toBytesPadded(BigInteger value, int length) {
        return toBytesPadded(value.toByteArray(), length);
    }

    /**
     * add the pad bytes to the passed in block, returning the
     * number of bytes added.
     */
    public static byte[] toBytesPadded(byte[] value, int length) {
        byte[] result = new byte[length];

        int bytesLength = value.length;
        int srcOffset = 0;

        if (bytesLength > length) {
            throw new IllegalArgumentException("Input is too large to put in byte array of size " + length);
        }

        int destOffset = length - bytesLength;
        System.arraycopy(value, srcOffset, result, destOffset, bytesLength);
        return result;
    }

    public static String cleanHexPrefix(String input) {
        if (containsHexPrefix(input)) {
            return input.substring(2);
        } else {
            return input;
        }
    }

    /**
     * Gets the data as a hex string
     *
     * @param withPrefix whether 0x prefix included
     * @return hex string
     */
    public String toHexString(boolean withPrefix) {
        return toHexString(withPrefix, data.length);
    }

    public static boolean containsHexPrefix(String input) {
        return input.length() > 1 && input.charAt(0) == '0' && input.charAt(1) == 'x';
    }

    /**
     * Gets the data as a byte array given size
     *
     * @param size size of byte array
     * @return byte array given size
     */
    public byte[] toByteArray(int size) {
        return toBytesPadded(data, size);
    }

    @Override
    public String toString() {
        return toHexString(true, data.length);
    }

    @Override
    public boolean equals(Object obj) {
        if (obj == this) return true;
        if (obj instanceof Bytes) {
            return Arrays.equals(((Bytes) obj).data, data);
        }
        return false;
    }

    @Override
    public int hashCode() {
        return Arrays.hashCode(data);
    }

    /**
     * Gets the data as a hex string given size
     *
     * @param withPrefix whether 0x prefix included
     * @param size       size of byte array
     * @return hex string given size
     */
    public String toHexString(boolean withPrefix, int size) {
        String result = Hex.toHexString(data);
        int length = result.length();
        if (length < size) {
            StringBuilder sb = new StringBuilder();
            for (int i = 0; i < size - length; i++) {
                sb.append('0');
            }
            result = sb.append(result).toString();
        }

        if (withPrefix) {
            return "0x" + result;
        } else {
            return result;
        }
    }

    public int length() {
        return data == null ? 0 : data.length;
    }

    private boolean isValidHex(String value) {
        String v = cleanHexPrefix(value);
        return v.matches("^[0-9a-fA-F]*$");
    }
}
