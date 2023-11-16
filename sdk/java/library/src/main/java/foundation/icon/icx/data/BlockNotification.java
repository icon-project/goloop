/*
 * Copyright 2019 ICON Foundation
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
import java.util.ArrayList;
import java.util.List;

public class BlockNotification {
    private final RpcObject properties;

    BlockNotification(RpcObject properties) {
        this.properties = properties;
    }

    public Bytes getHash() {
        RpcItem item = properties.getItem("hash");
        return item != null ? item.asBytes() : null;
    }

    public BigInteger getHeight() {
        return asInteger(properties.getItem("height"));
    }

    public BigInteger[][] getIndexes() {
        return asIntegerArrayArray(properties.getItem("indexes"));
    }

    public BigInteger[][][] getEvents() {
        RpcItem item = properties.getItem("events");
        BigInteger[][][] events = null;
        if (item != null) {
            RpcArray rpcArray = item.asArray();
            int size = rpcArray.size();
            events = new BigInteger[size][][];
            for (int i = 0; i < size; i++) {
                events[i] = asIntegerArrayArray(rpcArray.get(i));
            }
        }
        return events;
    }

    public static BigInteger[][] asIntegerArrayArray(RpcItem item) {
        BigInteger[][] arr = null;
        if (item != null) {
            RpcArray rpcArray = item.asArray();
            int size = rpcArray.size();
            arr = new BigInteger[size][];
            for (int i = 0; i < size; i++) {
                arr[i] = asIntegerArray(rpcArray.get(i));
            }
        }
        return arr;
    }

    public static BigInteger[] asIntegerArray(RpcItem item) {
        BigInteger[] arr = null;
        if (item != null) {
            RpcArray rpcArray = item.asArray();
            int size = rpcArray.size();
            arr = new BigInteger[size];
            for (int i = 0; i < size; i++) {
                arr[i] = asInteger(rpcArray.get(i));
            }
        }
        return arr;
    }

    public static BigInteger asInteger(RpcItem item) {
        return item != null ? item.asInteger() : null;
    }

    public List<TransactionResult.EventLog> getLogs() {
        RpcItem item = properties.getItem("logs");
        List<TransactionResult.EventLog> eventLogs = new ArrayList<>();
        if (item != null) {
            for (RpcItem rpcItem : item.asArray()) {
                eventLogs.add(new TransactionResult.EventLog(rpcItem.asObject()));
            }
        }
        return eventLogs;
    }

    @Override
    public String toString() {
        return "BlockNotification{Properties="+properties+"}";
    }
}
