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

import foundation.icon.icx.Transaction;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;

import java.math.BigInteger;


public class ConfirmedTransaction implements Transaction {

    private RpcObject properties;

    ConfirmedTransaction(RpcObject properties) {
        this.properties = properties;
    }

    @Override
    public RpcObject getProperties() {
        return properties;
    }

    @Override
    public BigInteger getVersion() {
        RpcItem item = properties.getItem("version");
        return item != null ? item.asInteger() : BigInteger.valueOf(2);
    }

    @Override
    public Address getFrom() {
        RpcItem item = properties.getItem("from");
        return item != null ? item.asAddress() : null;
    }

    @Override
    public Address getTo() {
        RpcItem item = properties.getItem("to");
        return item != null ? item.asAddress() : null;
    }

    public BigInteger getFee() {
        RpcItem item = properties.getItem("fee");
        return item != null ? convertHex(item.asValue()) : null;
    }

    @Override
    public BigInteger getValue() {
        RpcItem item = properties.getItem("value");
        if (item == null) {
            return null;
        }
        return getVersion().intValue() < 3 ? convertHex(item.asValue()) : item.asInteger();
    }

    @Override
    public BigInteger getStepLimit() {
        RpcItem item = properties.getItem("stepLimit");
        return item != null ? item.asInteger() : null;
    }

    @Override
    public BigInteger getTimestamp() {
        RpcItem item = properties.getItem("timestamp");
        if (item == null) {
            return null;
        }
        return getVersion().intValue() < 3 ? convertDecimal(item.asValue()) : item.asInteger();
    }

    @Override
    public BigInteger getNid() {
        RpcItem item = properties.getItem("nid");
        return item != null ? item.asInteger() : null;
    }

    @Override
    public BigInteger getNonce() {
        RpcItem item = properties.getItem("nonce");
        if (item == null) {
            return null;
        }
        return getVersion().intValue() < 3 ? convertDecimal(item.asValue()) : item.asInteger();
    }

    @Override
    public String getDataType() {
        RpcItem item = properties.getItem("dataType");
        return item != null ? item.asString() : null;
    }

    @Override
    public RpcItem getData() {
        return properties.getItem("data");
    }

    public Bytes getTxHash() {
        String key = getVersion().intValue() < 3 ? "tx_hash" : "txHash";
        RpcItem item = properties.getItem(key);
        return item != null ? item.asBytes() : null;
    }

    public BigInteger getTxIndex() {
        RpcItem item = properties.getItem("txIndex");
        return item != null ? item.asInteger() : null;
    }

    public BigInteger getBlockHeight() {
        RpcItem item = properties.getItem("blockHeight");
        return item != null ? item.asInteger() : null;
    }

    public Bytes getBlockHash() {
        RpcItem item = properties.getItem("blockHash");
        return item != null ? item.asBytes() : null;
    }

    public String getSignature() {
        RpcItem item = properties.getItem("signature");
        return item != null ? item.asString() : null;
    }

    @Override
    public String toString() {
        return "ConfirmedTransaction{" +
                "properties=" + properties +
                '}';
    }

    private BigInteger convertDecimal(RpcValue value) {
        // The value of timestamp and nonce in v2 specs is a decimal string.
        // But there are decimal strings, numbers and 0x included hex strings in v2 blocks.
        // e.g.) "12345", 12345, "0x12345"
        //
        // RpcValue class converts numbers and 0x included hex strings to 0x included hex string
        // and holds it
        //
        // So, stringValue is a decimal string or a 0x included hex string.("12345", "0x12345")
        // if it has 0x, the method converts it as hex otherwise decimal

        if (value.isEmpty()) {
            return null;
        }
        String stringValue = value.asString();
        if (stringValue.startsWith(Bytes.HEX_PREFIX) ||
                stringValue.startsWith("-" + Bytes.HEX_PREFIX)) {
            return convertHex(value);
        } else {
            return new BigInteger(stringValue, 10);
        }
    }

    private BigInteger convertHex(RpcValue value) {
        // The value of 'value' and nonce in v2 specs is a decimal string.
        // But there are hex strings without 0x in v2 blocks.
        //
        // This method converts the value as hex no matter it has  0x prefix or not.

        String stringValue = value.asString();
        String sign = "";
        if (stringValue.charAt(0) == '-') {
            sign = stringValue.substring(0, 1);
            stringValue = stringValue.substring(1);
        }
        return new BigInteger(sign + Bytes.cleanHexPrefix(stringValue), 16);
    }
}
