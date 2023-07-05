/*
 * Copyright 2023 ICON Foundation
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

import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;

import java.math.BigInteger;

public class NetworkInfo {
    private final RpcObject properties;

    NetworkInfo(RpcObject properties) {
        this.properties = properties;
    }

    public RpcObject getProperties() {
        return properties;
    }

    public String getPlatform() {
        RpcItem item = properties.getItem("platform");
        return item != null ? item.asString() : null;
    }

    public BigInteger getNID() {
        RpcItem item = properties.getItem("nid");
        return item != null ? item.asInteger() : null;
    }

    public String getChannel() {
        RpcItem item = properties.getItem("channel");
        return item != null ? item.asString() : null;
    }

    public BigInteger getEarliest() {
        RpcItem item = properties.getItem("earliest");
        return item != null ? item.asInteger() : null;
    }

    public BigInteger getLatest() {
        RpcItem item = properties.getItem("latest");
        return item != null ? item.asInteger() : null;
    }

    public BigInteger getStepPrice() {
        RpcItem item = properties.getItem("stepPrice");
        return item != null ? item.asInteger() : null;
    }

    @Override
    public String toString() {
        return "NetworkInfo{" +
                "properties=" + properties +
                '}';
    }
}
