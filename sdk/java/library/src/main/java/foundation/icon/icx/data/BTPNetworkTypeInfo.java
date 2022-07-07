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
import java.util.ArrayList;
import java.util.List;

import static foundation.icon.icx.data.Converters.BIG_INTEGER;

public class BTPNetworkTypeInfo {

    private final RpcObject properties;

    BTPNetworkTypeInfo(RpcObject properties) {
        this.properties = properties;
    }

    public RpcObject getProperties() {
        return properties;
    }

    public BigInteger getNetworkTypeID() {
        RpcItem item = properties.getItem("networkTypeID");
        return item != null ? item.asInteger() : null;
    }

    public String getNetworkTypeName() {
        RpcItem item = properties.getItem("networkTypeName");
        return item != null ? item.asString() : null;
    }

    public List<BigInteger> getOpenNetworkIDs() {
        RpcItem item = properties.getItem("openNetworkIDs");
        List<BigInteger> ids = new ArrayList<>();
        if (item != null) {
            for (RpcItem rpcItem : item.asArray()) {
                ids.add(BIG_INTEGER.convertTo(rpcItem));
            }
        }
        return ids;
    }

    public String getNextProofContext() {
        RpcItem item = properties.getItem("nextProofContext");
        return item != null ? item.asString() : null;
    }

    @Override
    public boolean equals(Object obj) {
        if (this == obj) {
            return true;
        }
        if (!(obj instanceof BTPNetworkTypeInfo)) {
            return false;
        }
        BTPNetworkTypeInfo ntInfo = (BTPNetworkTypeInfo) obj;
        return properties.equals(ntInfo.properties);
    }

    @Override
    public String toString() {
        return "BTPNetworkTypeInfo{" +
                "properties=" + properties +
                '}';
    }
}
