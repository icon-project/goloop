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

public class BTPSourceInfo {

    private final RpcObject properties;

    public BTPSourceInfo(RpcObject properties) {
        this.properties = properties;
    }

    public String getSrcNetworkUID() {
        RpcItem item = properties.getItem("srcNetworkUID");
        return item != null ? item.asString() : null;
    }

    public List<BigInteger> getNetworkTypeIDs() {
        RpcItem item = properties.getItem("networkTypeIDs");
        List<BigInteger> ids = new ArrayList<>();
        if (item != null) {
            for (RpcItem rpcItem : item.asArray()) {
                ids.add(BIG_INTEGER.convertTo(rpcItem));
            }
        }
        return ids;
    }

    @Override
    public String toString() {
        return "BTPSourceInfo{" +
                "properties=" + properties +
                '}';
    }
}
