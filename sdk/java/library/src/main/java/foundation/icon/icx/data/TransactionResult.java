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

import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.List;

/**
 * @see <a href="https://github.com/icon-project/icon-rpc-server/blob/develop/docs/icon-json-rpc-v3.md#icx_gettransactionresult" target="_blank">ICON JSON-RPC API</a>
 */
public class TransactionResult {

    private final RpcObject properties;

    TransactionResult(RpcObject properties) {
        this.properties = properties;
    }

    public RpcObject getProperties() {
        return properties;
    }

    /**
     * @return 1 on success, 0 on failure.
     */
    public BigInteger getStatus() {
        RpcItem status = properties.getItem("status");
        if (status != null) {
            return status.asInteger();
        } else {
            // Migrates V2 block data
            // V2 Block data doesn't have a status field but have a code field
            // @see <a href="https://github.com/icon-project/icx_JSON_RPC#icx_gettransactionresult" target="_blank">ICON JSON-RPC V2 API</a>
            RpcItem code = properties.getItem("code");
            if (code != null) return new BigInteger(code.asInteger().intValue() == 0 ? "1" : "0");
            else return null;
        }
    }

    /**
     * @return Recipient address of the transaction
     */
    public String getTo() {
        RpcItem item = properties.getItem("to");
        return item != null ? item.asString() : null;
    }

    /**
     * @return Transaction hash
     */
    public Bytes getTxHash() {
        RpcItem item = properties.getItem("txHash");
        return item != null ? item.asBytes() : null;
    }

    /**
     * @return Transaction index in the block
     */
    public BigInteger getTxIndex() {
        RpcItem item = properties.getItem("txIndex");
        return item != null ? item.asInteger() : null;
    }

    /**
     * @return Height of the block that includes the transaction
     */
    public BigInteger getBlockHeight() {
        RpcItem item = properties.getItem("blockHeight");
        return item != null ? item.asInteger() : null;
    }

    /**
     * @return Hash of the block that includes the transaction
     */
    public Bytes getBlockHash() {
        RpcItem item = properties.getItem("blockHash");
        return item != null ? item.asBytes() : null;
    }

    /**
     * @return Sum of stepUsed by this transaction and all preceding transactions in the same block
     */
    public BigInteger getCumulativeStepUsed() {
        RpcItem item = properties.getItem("cumulativeStepUsed");
        return item != null ? item.asInteger() : null;
    }

    /**
     * @return The amount of step used by this transaction
     */
    public BigInteger getStepUsed() {
        RpcItem item = properties.getItem("stepUsed");
        return item != null ? item.asInteger() : null;
    }

    /**
     * @since 0.9.13
     * @return List of accounts that paid fees for this transaction
     */
    public RpcItem getStepUsedDetails() {
        return properties.getItem("stepUsedDetails");
    }

    /**
     * @return The step price used by this transaction
     */
    public BigInteger getStepPrice() {
        RpcItem item = properties.getItem("stepPrice");
        return item != null ? item.asInteger() : null;
    }

    /**
     * @return SCORE address if the transaction created a new SCORE
     */
    public String getScoreAddress() {
        RpcItem item = properties.getItem("scoreAddress");
        return item != null ? item.asString() : null;
    }

    /**
     * @return Bloom filter to quickly retrieve related eventlogs
     */
    public String getLogsBloom() {
        RpcItem item = properties.getItem("logsBloom");
        return item != null ? item.asString() : null;
    }

    /**
     * @return List of event logs that this transaction generated
     */
    public List<EventLog> getEventLogs() {
        RpcItem item = properties.getItem("eventLogs");
        List<EventLog> eventLogs = new ArrayList<>();
        if (item != null) {
            for (RpcItem rpcItem : item.asArray()) {
                eventLogs.add(new EventLog(rpcItem.asObject()));
            }
        }
        return eventLogs;
    }

    /**
     * @return This field exists when status is 0. Contains code(str) and message(str).
     */
    public Failure getFailure() {
        RpcItem failure = properties.getItem("failure");

        if (failure == null) {
            BigInteger status = getStatus();
            if (status != null && status.intValue() == 0) {
                // Migrates V2 block data
                // V2 Block data doesn't have a failure field but have a code field
                // @see <a href="https://github.com/icon-project/icx_JSON_RPC#icx_gettransactionresult" target="_blank">ICON JSON-RPC V2 API</a>
                RpcItem code = properties.getItem("code");
                if (code != null) {
                    RpcObject.Builder builder = new RpcObject.Builder();
                    builder.put("code", code);

                    RpcItem message = properties.getItem("message");
                    if (message != null) {
                        builder.put("message", message);
                    }
                    failure = builder.build();
                }
            }
        }
        return failure != null ? new Failure(failure.asObject()) : null;
    }

    @Override
    public String toString() {
        return "TransactionResult{" +
                "properties=" + properties +
                '}';
    }

    public static class EventLog {
        private final RpcObject properties;

        EventLog(RpcObject properties) {
            this.properties = properties;
        }

        public String getScoreAddress() {
            RpcItem item = properties.getItem("scoreAddress");
            return item != null ? item.asString() : null;
        }

        public List<RpcItem> getIndexed() {
            RpcItem item = properties.getItem("indexed");
            return item != null ? item.asArray().asList() : null;
        }

        public List<RpcItem> getData() {
            RpcItem field = properties.getItem("data");
            return field != null ? field.asArray().asList() : null;
        }

        @Override
        public String toString() {
            return "EventLog{" +
                    "properties=" + properties +
                    '}';
        }
    }

    public static class Failure {
        private final RpcObject properties;

        private Failure(RpcObject properties) {
            this.properties = properties;
        }

        public BigInteger getCode() {
            RpcItem item = properties.getItem("code");
            return item != null ? item.asInteger() : null;
        }

        public String getMessage() {
            RpcItem item = properties.getItem("message");
            return item != null ? item.asString() : null;
        }

        @Override
        public String toString() {
            return "Failure{" +
                    "properties=" + properties +
                    '}';
        }
    }
}
