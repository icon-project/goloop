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

public class BTPNotification {
    private final RpcObject properties;

    BTPNotification(RpcObject properties) {
        this.properties = properties;
    }

    public Base64 getHeader() {
        RpcItem item = properties.getItem("header");
        return item != null ? new Base64(item.asString()) : null;
    }

    public Base64 getProof() {
        RpcItem item = properties.getItem("proof");
        return item != null ? new Base64(item.asString()) : null;
    }
}
