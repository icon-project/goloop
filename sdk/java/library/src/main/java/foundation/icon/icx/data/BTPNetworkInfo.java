/*
 * Copyright 2022 ICON Foundation
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

import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;

import java.math.BigInteger;

public class BTPNetworkInfo {

    private final RpcObject properties;

    public BTPNetworkInfo(RpcObject properties) {
        this.properties = properties;
    }

    public RpcObject getProperties() {
        return properties;
    }

    public BigInteger getStartHeight() {
        RpcItem item = properties.getItem("startHeight");
        return item != null ? item.asInteger() : null;
    }

    public BigInteger getNetworkTypeID() {
        RpcItem item = properties.getItem("networkTypeID");
        return item != null ? item.asInteger() : null;
    }

    public String getNetworkTypeName() {
        RpcItem item = properties.getItem("networkTypeName");
        return item != null ? item.asString() : null;
    }

    public BigInteger getNetworkID() {
        RpcItem item = properties.getItem("networkID");
        return item != null ? item.asInteger() : null;
    }

    public String getNetworkName() {
        RpcItem item = properties.getItem("networkName");
        return item != null ? item.asString() : null;
    }

    public BigInteger getOpen() {
        RpcItem item = properties.getItem("open");
        return item != null ? item.asInteger() : null;
    }

    public BigInteger getNextMessageSN() {
        RpcItem item = properties.getItem("nextMessageSN");
        return item != null ? item.asInteger() : null;
    }

    public Bytes getPrevNSHash() {
        RpcItem item = properties.getItem("prevNSHash");
        if (item == null || item.isEmpty()) {
            return null;
        }
        return item.asBytes();
    }

    public Bytes getLastNSHash() {
        RpcItem item = properties.getItem("lastNSHash");
        return item != null ? item.asBytes() : null;
    }

    @Override
    public boolean equals(Object obj) {
        if (this == obj) return true;
        if (!(obj instanceof BTPNetworkInfo)) return false;
        BTPNetworkInfo o = (BTPNetworkInfo) obj;
        return properties.equals(o.properties);
    }

    @Override
    public String toString() {
        return "BTPNetworkInfo{" +
                "properties=" + properties +
                '}';
    }
}
