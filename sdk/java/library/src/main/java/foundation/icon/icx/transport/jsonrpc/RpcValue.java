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

package foundation.icon.icx.transport.jsonrpc;

import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;

import java.math.BigInteger;

import static foundation.icon.icx.data.Bytes.HEX_PREFIX;

/**
 * RpcValue contains a leaf value such as string, bytes, integer, boolean
 */
public class RpcValue implements RpcItem {

    private String value;

    public RpcValue(RpcValue value) {
        this.value = value.asString();
    }

    public RpcValue(Address value) {
        if (value.isMalformed()) throw new IllegalArgumentException("Invalid address");
        this.value = value.toString();
    }

    private RpcValue() {
        value = null;
    }

    public final static RpcValue NULL = new RpcValue();

    public RpcValue(String value) {
        this.value = value;
    }

    public RpcValue(byte[] value) {
        this.value = new Bytes(value).toHexString(true);
    }

    public RpcValue(BigInteger value) {
        String sign = (value.signum() == -1) ? "-" : "";
        this.value = sign + HEX_PREFIX + value.abs().toString(16);
    }

    public RpcValue(boolean value) {
        this.value = value ? "0x1" : "0x0";
    }

    public RpcValue(Boolean value) {
        this(value.booleanValue());
    }

    public RpcValue(Bytes value) {
        this.value = value.toString();
    }

    @Override
    public boolean isEmpty() {
        return value == null || value.isEmpty();
    }

    @Override
    public boolean isNull() {
        return value == null;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (!(o instanceof  RpcValue)) return false;
        RpcValue obj = (RpcValue) o;
        return value.equals(obj.value);
    }

    /**
     * Returns the value as string
     *
     * @return the value as string
     */
    @Override
    public String asString() {
        return value;
    }

    /**
     * Returns the value as bytes
     *
     * @return the value as bytes
     */
    @Override
    public byte[] asByteArray() {
        if (!value.startsWith(HEX_PREFIX)) {
            throw new RpcValueException("The value is not hex string.");
        }

        // bytes should be even length of hex string
        if (value.length() % 2 != 0) {
            throw new RpcValueException(
                    "The hex value is not bytes format.");
        }
        if (value.length()==2) {
            return new byte[]{};
        }
        return new Bytes(value).toByteArray();
    }

    @Override
    public Address asAddress() {
        try {
            if (isEmpty()) {
                return null;
            }
            return new Address(value);
        } catch (IllegalArgumentException e) {
            return Address.createMalformedAddress(value);
        }
    }

    @Override
    public Bytes asBytes() {
        return new Bytes(value);
    }

    /**
     * Returns the value as integer
     *
     * @return the value as integer
     */
    @Override
    public BigInteger asInteger() {
        if (!(value.startsWith(HEX_PREFIX) || value.startsWith('-' + HEX_PREFIX))) {
            throw new RpcValueException("The value is not hex string.");
        }

        try {
            String sign = "";
            if (value.charAt(0) == '-') {
                sign = value.substring(0, 1);
                value = value.substring(1);
            }
            String result = sign + Bytes.cleanHexPrefix(value);
            return new BigInteger(result, 16);
        } catch (NumberFormatException e) {
            throw new RpcValueException("The value is not hex string.");
        }
    }

    /**
     * Returns the value as boolean
     *
     * @return the value as boolean
     */
    @Override
    public Boolean asBoolean() {
        switch (value) {
            case "0x0":
                return false;
            case "0x1":
                return true;
            default:
                throw new RpcValueException("The value is not boolean format.");
        }
    }

    @Override
    public String toString() {
        return value;
    }
}
