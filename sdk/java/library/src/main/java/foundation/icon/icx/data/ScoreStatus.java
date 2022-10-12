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

import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;

import java.math.BigInteger;

public class ScoreStatus {
    private final RpcObject properties;

    ScoreStatus(RpcObject properties) {
        this.properties = properties;
    }

    public RpcObject getProperties() {
        return properties;
    }

    public Address getOwner() {
        RpcItem item = properties.getItem("owner");
        return item != null ? item.asAddress() : null;
    }

    public ContractStatus getCurrent() {
        RpcItem item = properties.getItem("current");
        return item != null ? new ContractStatus(item.asObject()) : null;
    }

    public ContractStatus getNext() {
        RpcItem item = properties.getItem("next");
        return item != null ? new ContractStatus(item.asObject()) : null;
    }

    public static class ContractStatus {
        private final RpcObject properties;

        ContractStatus(RpcObject properties) {
            this.properties = properties;
        }

        public Bytes getDeployTxHash() {
            RpcItem item = properties.getItem("deployTxHash");
            return item != null ? item.asBytes() : null;
        }

        public Bytes getAuditTxHash() {
            RpcItem item = properties.getItem("auditTxHash");
            return item != null ? item.asBytes() : null;
        }

        public Bytes getCodeHash() {
            RpcItem code_hash = this.properties.getItem("codeHash");
            return code_hash.asBytes();
        }

        public String getType() {
            RpcItem item = properties.getItem("type");
            return item.asString();
        }

        public String getStatus() {
            RpcItem item = properties.getItem("status");
            return item.asString();
        }

        @Override
        public String toString() {
            return "Contract{properties="+ properties + "}";
        }
    }

    public DepositInfo getDepositInfo() {
        RpcItem item = properties.getItem("depositInfo");
        return item != null ? new DepositInfo(item.asObject()) : null;
    }

    public static class DepositInfo {
        private final RpcObject properties;
        private final RpcArray deposits;

        DepositInfo(RpcObject properties) {
            this.properties = properties;
            this.deposits = properties.getItem("deposits").asArray();
        }

        @Override
        public String toString() {
            return "DepositInfo{properties="+properties+"}";
        }

        public BigInteger getAvailableDeposit() {
            RpcItem item = properties.getItem("availableDeposit");
            return item.asInteger();
        }

        public BigInteger getAvailableVirtualStep() {
            RpcItem item = properties.getItem("availableVirtualStep");
            return item.asInteger();
        }

        public RpcObject getDeposit(int idx) {
            return deposits.get(idx).asObject();
        }

        public int getSizeOfDeposits() {
            return deposits.size();
        }
    }

    public boolean isDisabled() {
        RpcItem value = properties.getItem("disabled");
        return value != null && value.asBoolean();
    }

    public boolean isBlocked() {
        RpcItem value = properties.getItem("blocked");
        return value != null && value.asBoolean();
    }

    public boolean useSystemDeposit() {
        RpcItem value = properties.getItem("useSystemDeposit");
        return value != null && value.asBoolean();
    }

    @Override
    public String toString() {
        return "ScoreStatus{" +
                "properties=" + properties +
                '}';
    }
}
