/*
 * Copyright 2020 ICON Foundation
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

package foundation.icon.test.common;

import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;

public class RpcItems {
    public static boolean equals(RpcItem l, RpcItem r) {
        if (l instanceof RpcObject && r instanceof RpcObject) {
            var lo = (RpcObject) l;
            var ro = (RpcObject) r;
            for (var k : lo.keySet()) {
                if (!equals(lo.getItem(k), ro.getItem(k))) {
                    return false;
                }
            }
            return true;
        }
        return l.toString().equals(r.toString());
    }
}
