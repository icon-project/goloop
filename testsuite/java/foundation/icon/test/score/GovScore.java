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

package foundation.icon.test.score;

import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;

import java.io.IOException;
import java.math.BigInteger;
import java.util.HashMap;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;

public class GovScore extends Score {
    public static final String INSTALL_PATH = "./data/genesisStorage/" + "governance";
    public static final String UPDATE_PATH = getFilePath("governance");

    private final Wallet governorWallet;
    private final ChainScore chainScore;

    public static class Fee {
        Map<String, BigInteger> stepCosts;
        Map<String, BigInteger> stepMaxLimits;
        BigInteger stepPrice;
    }

    public static final String[] stepCostTypes = {
            "default",
            "contractCall",
            "contractCreate",
            "contractUpdate",
            "contractDestruct",
            "contractSet",
            "get",
            "set",
            "replace",
            "delete",
            "input",
            "eventLog",
            "apiCall"
    };

    public GovScore(TransactionHandler txHandler) {
        super(txHandler, Constants.GOV_ADDRESS);
        this.governorWallet = txHandler.getChain().governorWallet;
        this.chainScore = new ChainScore(txHandler);
    }

    private Wallet getWallet() {
        return this.governorWallet;
    }

    @Override
    public TransactionResult invokeAndWaitResult(Wallet wallet, String method, RpcObject params)
            throws ResultTimeoutException, IOException {
        return super.invokeAndWaitResult(wallet, method, params, BigInteger.ZERO, Constants.DEFAULT_STEPS);
    }

    public TransactionResult setRevision(int code) throws Exception {
        RpcObject params = new RpcObject.Builder()
                .put("code", new RpcValue(BigInteger.valueOf(code)))
                .build();
        return invokeAndWaitResult(getWallet(), "setRevision", params);
    }

    public void setStepPrice(BigInteger price) throws Exception {
        RpcObject params = new RpcObject.Builder()
                .put("price", new RpcValue(price))
                .build();
        invokeAndWaitResult(getWallet(), "setStepPrice", params);
    }

    public void setStepCost(String type, BigInteger cost) throws ResultTimeoutException, IOException {
        RpcObject params = new RpcObject.Builder()
                .put("type", new RpcValue(type))
                .put("cost", new RpcValue(cost))
                .build();
        invokeAndWaitResult(getWallet(), "setStepCost", params);
    }

    public TransactionResult setMaxStepLimit(String type, BigInteger cost) throws ResultTimeoutException, IOException {
        RpcObject params = new RpcObject.Builder()
                .put("contextType", new RpcValue(type))
                .put("limit", new RpcValue(cost))
                .build();
        return invokeAndWaitResult(getWallet(), "setMaxStepLimit", params);
    }

    public boolean isAuditEnabledOnly() throws IOException {
        int config = chainScore.getServiceConfig();
        return ChainScore.isAuditEnabled(config) && !ChainScore.isDeployerWhiteListEnabled(config);
    }

    public TransactionResult acceptScore(Bytes txHash) throws ResultTimeoutException, IOException {
        RpcObject params = new RpcObject.Builder()
                .put("txHash", new RpcValue(txHash))
                .build();
        return invokeAndWaitResult(getWallet(), "acceptScore", params);
    }

    public TransactionResult rejectScore(Bytes txHash) throws ResultTimeoutException, IOException {
        RpcObject params = new RpcObject.Builder()
                .put("txHash", new RpcValue(txHash))
                .build();
        return invokeAndWaitResult(getWallet(), "rejectScore", params);
    }

    public Map<String, BigInteger> getStepCosts() throws Exception {
        RpcItem rpcItem = this.chainScore.call("getStepCosts", null);
        Map<String, BigInteger> map = new HashMap<>();
        for(String type : stepCostTypes) {
            map.put(type, rpcItem.asObject().getItem(type).asInteger());
        }
        return map;
    }

    public void setStepCosts(Map<String, BigInteger> map)
            throws ResultTimeoutException, TransactionFailureException, IOException {
        List<Bytes> list = new LinkedList<>();
        for(String type : map.keySet()) {
            RpcObject params = new RpcObject.Builder()
                    .put("type", new RpcValue(type))
                    .put("cost", new RpcValue(map.get(type)))
                    .build();
            Bytes txHash = invoke(getWallet(), "setStepCost", params);
            list.add(txHash);
        }
        for(Bytes txHash : list) {
            TransactionResult result = getResult(txHash);
            if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
                throw new TransactionFailureException(result.getFailure());
            }
        }
    }

    public Map<String, BigInteger> getMaxStepLimits() throws Exception {
        Map<String, BigInteger> map = new HashMap<>();
        String[] types = {"invoke", "query"};
        for(String t : types) {
            RpcObject params = new RpcObject.Builder()
                    .put("contextType", new RpcValue(t))
                    .build();
            BigInteger stepLimit = this.chainScore.call("getMaxStepLimit", params).asInteger();
            map.put(t, stepLimit);
        }
        return map;
    }

    public void setMaxStepLimits(Map<String, BigInteger> limits)
            throws ResultTimeoutException, TransactionFailureException, IOException {
        List<Bytes> list = new LinkedList<>();
        for(String type : limits.keySet()) {
            RpcObject params = new RpcObject.Builder()
                    .put("contextType", new RpcValue(type))
                    .put("limit", new RpcValue(limits.get(type)))
                    .build();
            Bytes txHash = invoke(getWallet(), "setMaxStepLimit", params);
            list.add(txHash);
        }
        for(Bytes txHash : list) {
            TransactionResult result = getResult(txHash);
            if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
                throw new TransactionFailureException(result.getFailure());
            }
        }
    }

    public Fee getFee() throws Exception {
        Fee fee = new Fee();
        fee.stepCosts = getStepCosts();
        fee.stepMaxLimits = getMaxStepLimits();
        fee.stepPrice = this.chainScore.getStepPrice();
        return fee;
    }

    public void setFee(Fee fee) throws Exception {
        setStepPrice(fee.stepPrice);
        setStepCosts(fee.stepCosts);
        setMaxStepLimits(fee.stepMaxLimits);
    }

    public TransactionResult addDeployer(Address address) throws IOException, ResultTimeoutException {
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(address))
                .build();
        return invokeAndWaitResult(getWallet(), "addDeployer", params);
    }

    public TransactionResult removeDeployer(Address address) throws IOException, ResultTimeoutException {
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(address))
                .build();
        return invokeAndWaitResult(getWallet(), "removeDeployer", params);
    }

    public TransactionResult setDeployerWhiteListEnabled(boolean yn) throws IOException, ResultTimeoutException {
        RpcObject params = new RpcObject.Builder()
                .put("yn", new RpcValue(yn))
                .build();
        return invokeAndWaitResult(getWallet(), "setDeployerWhiteListEnabled", params);
    }

    public TransactionResult setUseSystemDeposit(Address address, boolean yn) throws IOException, ResultTimeoutException {
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(address))
                .put("yn", new RpcValue(yn))
                .build();
        return invokeAndWaitResult(getWallet(), "setUseSystemDeposit", params);
    }

    public TransactionResult openBTPNetwork(String networkTypeName, String name, Address owner) throws IOException, ResultTimeoutException {
        RpcObject params = new RpcObject.Builder()
                .put("networkTypeName", new RpcValue(networkTypeName))
                .put("name", new RpcValue(name))
                .put("owner", new RpcValue(owner))
                .build();
        return invokeAndWaitResult(getWallet(), "openBTPNetwork", params);
    }

    public TransactionResult closeBTPNetwork(BigInteger id) throws IOException, ResultTimeoutException {
        RpcObject params = new RpcObject.Builder()
                .put("id", new RpcValue(id))
                .build();
        return invokeAndWaitResult(getWallet(), "closeBTPNetwork", params);
    }
}
