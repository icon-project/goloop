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

package foundation.icon.test.score;

import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionHandler;

import java.io.IOException;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.List;

public class ChainScore extends Score {
    private static final int CONFIG_AUDIT = 0x2;
    private static final int CONFIG_DEPLOYER_WHITE_LIST = 0x4;

    public ChainScore(TransactionHandler txHandler) {
        super(txHandler, Constants.CHAINSCORE_ADDRESS);
    }

    public int getRevision() throws IOException {
        return call("getRevision", null).asInteger().intValue();
    }

    public BigInteger getStepPrice() throws IOException {
        return call("getStepPrice", null).asInteger();
    }

    public int getServiceConfig() throws IOException {
        return call("getServiceConfig", null).asInteger().intValue();
    }

    public static boolean isAuditEnabled(int config) {
        return (config & CONFIG_AUDIT) != 0;
    }

    public boolean isAuditEnabled() throws IOException {
        return isAuditEnabled(this.getServiceConfig());
    }

    public static boolean isDeployerWhiteListEnabled(int config) {
        return (config & CONFIG_DEPLOYER_WHITE_LIST) != 0;
    }

    public boolean isDeployerWhiteListEnabled() throws IOException {
        return isDeployerWhiteListEnabled(this.getServiceConfig());
    }

    public boolean isDeployer(Address address) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(address))
                .build();
        return call("isDeployer", params).asBoolean();
    }

    public List<Address> getDeployers() throws IOException {
        List<Address> list = new ArrayList<>();
        RpcArray items = call("getDeployers", null).asArray();
        for (RpcItem item : items) {
            list.add(item.asAddress());
        }
        return list;
    }

    public RpcObject getScoreStatus(Address address) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(address))
                .build();
        return call("getScoreStatus", params).asObject();
    }

    public BigInteger getSystemDepositUsage() throws IOException {
        return call("getSystemDepositUsage", null).asInteger();
    }

    public TransactionResult disableScore(Wallet wallet, Address address)
            throws IOException, ResultTimeoutException {
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(address))
                .build();
        return invokeAndWaitResult(wallet, "disableScore", params, null, Constants.DEFAULT_STEPS);
    }

    public TransactionResult enableScore(Wallet wallet, Address address)
            throws IOException, ResultTimeoutException {
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(address))
                .build();
        return invokeAndWaitResult(wallet, "enableScore", params, null, Constants.DEFAULT_STEPS);
    }

    public TransactionResult setBTPPublicKey(Wallet wallet, String name, byte[] pubKey)
            throws IOException, ResultTimeoutException {
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue(name))
                .put("pubKey", new RpcValue(pubKey))
                .build();
        return invokeAndWaitResult(wallet, "setBTPPublicKey", params, null, Constants.DEFAULT_STEPS);
    }

    public byte[] getBTPPublicKey(Address address, String name) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(address))
                .put("name", new RpcValue(name))
                .build();
        RpcItem ret = call("getBTPPublicKey", params);
        return ret == null ? null : ret.asByteArray();
    }

    public BigInteger getBTPNetworkTypeID(String name) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue(name))
                .build();
        RpcItem ret = call("getBTPNetworkTypeID", params);
        return ret.asInteger();
    }

    public TransactionResult sendBTPMessage(Wallet wallet, BigInteger networkId, byte[] message) throws IOException, ResultTimeoutException {
        RpcObject params = new RpcObject.Builder()
                .put("networkId", new RpcValue(networkId))
                .put("message", new RpcValue(message))
                .build();
        return invokeAndWaitResult(wallet, "sendBTPMessage", params, null, Constants.DEFAULT_STEPS);
    }
}
