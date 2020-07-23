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

import static foundation.icon.icx.data.Converters.CONFIRMED_TRANSACTION;

public class Block {

    private final RpcObject properties;

    Block(RpcObject properties) {
        this.properties = properties;
    }

    public RpcObject getProperties() {
        return properties;
    }

    public Bytes getPrevBlockHash() {
        RpcItem item = properties.getItem("prev_block_hash");
        return item != null ? item.asBytes() : null;
    }

    public Bytes getMerkleTreeRootHash() {
        RpcItem item = properties.getItem("merkle_tree_root_hash");
        return item != null ? item.asBytes() : null;
    }

    public BigInteger getTimestamp() {
        RpcItem item = properties.getItem("time_stamp");
        return item != null ? item.asInteger() : null;
    }

    public List<ConfirmedTransaction> getTransactions() {
        RpcItem item = properties.getItem("confirmed_transaction_list");
        List<ConfirmedTransaction> transactions = new ArrayList<>();
        if (item != null && getHeight().intValue() > 0) {
            for (RpcItem tx : item.asArray()) {
                transactions.add(CONFIRMED_TRANSACTION.convertTo(tx.asObject()));
            }
        }
        return transactions;
    }

    public Bytes getBlockHash() {
        RpcItem item = properties.getItem("block_hash");
        return item != null ? item.asBytes() : null;
    }

    public String getPeerId() {
        RpcItem item = properties.getItem("peer_id");
        return item != null ? item.asString() : null;
    }

    public BigInteger getVersion() {
        RpcItem item = properties.getItem("version");
        return item != null ? item.asInteger() : null;
    }

    public BigInteger getHeight() {
        RpcItem item = properties.getItem("height");
        return item != null ? item.asInteger() : null;
    }

    public String getSignature() {
        RpcItem item = properties.getItem("signature");
        return item != null ? item.asString() : null;
    }

    @Override
    public String toString() {
        return "Block{" +
                "properties=" + properties +
                '}';
    }
}
