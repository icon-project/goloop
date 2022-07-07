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

public interface RpcItem {

    boolean isEmpty();
    boolean equals(Object o);
    default boolean isNull() {
        return false;
    }

    default RpcObject asObject() {
        if (this instanceof RpcObject) return (RpcObject) this;
        throw new RpcValueException("This item can not be converted to RpcObject");
    }

    default RpcArray asArray() {
        if (this instanceof RpcArray) return (RpcArray) this;
        throw new RpcValueException("This item can not be converted to RpcValue");
    }

    default RpcValue asValue() {
        if (this instanceof RpcValue) return (RpcValue) this;
        throw new RpcValueException("This item can not be converted to RpcValue");
    }

    default String asString() {
        return asValue().asString();
    }

    default BigInteger asInteger() {
        return asValue().asInteger();
    }

    default byte[] asByteArray() {
        return asValue().asByteArray();
    }

    default Boolean asBoolean() {
        return asValue().asBoolean();
    }

    default Address asAddress() {
        return asValue().asAddress();
    }

    default Bytes asBytes() {
        return asValue().asBytes();
    }

    class RpcValueException extends IllegalArgumentException {
        RpcValueException(String message) {
            super(message);
        }
    }
}
