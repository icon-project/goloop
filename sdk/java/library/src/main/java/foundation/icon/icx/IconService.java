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

package foundation.icon.icx;

import foundation.icon.icx.crypto.IconKeys;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Base64;
import foundation.icon.icx.data.Block;
import foundation.icon.icx.data.BlockNotification;
import foundation.icon.icx.data.BTPNetworkInfo;
import foundation.icon.icx.data.BTPNetworkTypeInfo;
import foundation.icon.icx.data.BTPSourceInfo;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.ConfirmedTransaction;
import foundation.icon.icx.data.Converters;
import foundation.icon.icx.data.EventNotification;
import foundation.icon.icx.data.ScoreApi;
import foundation.icon.icx.data.ScoreStatus;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.AnnotatedConverterFactory;
import foundation.icon.icx.transport.jsonrpc.AnnotationConverter;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcConverter;
import foundation.icon.icx.transport.jsonrpc.RpcConverter.RpcConverterFactory;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.icx.transport.monitor.BTPMonitorSpec;
import foundation.icon.icx.transport.monitor.BlockMonitorSpec;
import foundation.icon.icx.transport.monitor.EventMonitorSpec;
import foundation.icon.icx.transport.monitor.Monitor;
import foundation.icon.icx.transport.monitor.MonitorSpec;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * IconService which provides APIs of ICON network.
 */
@SuppressWarnings("WeakerAccess")
public class IconService {

    private Provider provider;
    private final List<RpcConverter.RpcConverterFactory> converterFactories = new ArrayList<>();
    private final Map<Class<?>, RpcConverter<?>> converterMap = new HashMap<>();

    /**
     * Creates an IconService instance
     *
     * @param provider the worker that transports requests
     */
    @SuppressWarnings("unchecked")
    public IconService(Provider provider) {
        this.provider = provider;
        addConverterFactory(Converters.newFactory(BigInteger.class, Converters.BIG_INTEGER));
        addConverterFactory(Converters.newFactory(Boolean.class, Converters.BOOLEAN));
        addConverterFactory(Converters.newFactory(String.class, Converters.STRING));
        addConverterFactory(Converters.newFactory(Bytes.class, Converters.BYTES));
        addConverterFactory(Converters.newFactory(byte[].class, Converters.BYTE_ARRAY));
        addConverterFactory(Converters.newFactory(Block.class, Converters.BLOCK));
        addConverterFactory(Converters.newFactory(
                ConfirmedTransaction.class, Converters.CONFIRMED_TRANSACTION));
        addConverterFactory(Converters.newFactory(
                TransactionResult.class, Converters.TRANSACTION_RESULT));
        Class<List<ScoreApi>> listClass = ((Class) List.class);
        addConverterFactory(Converters.newFactory(listClass, Converters.SCORE_API_LIST));
        addConverterFactory(Converters.newFactory(RpcItem.class, Converters.RPC_ITEM));
        addConverterFactory(Converters.newFactory(
                BlockNotification.class, Converters.BLOCK_NOTIFICATION));
        addConverterFactory(Converters.newFactory(
                EventNotification.class, Converters.EVENT_NOTIFICATION));
        addConverterFactory(Converters.newFactory(Base64[].class, Converters.BASE64_ARRAY));
        addConverterFactory(Converters.newFactory(Base64[][].class, Converters.BASE64_ARRAY_ARRAY));
        addConverterFactory(Converters.newFactory(
                Base64.class, Converters.BASE64));
        addConverterFactory(Converters.newFactory(BTPNetworkInfo.class, Converters.BTP_NETWORK_INFO));
        addConverterFactory(Converters.newFactory(BTPNetworkTypeInfo.class, Converters.BTP_NETWORK_TYPE_INFO));
        addConverterFactory(Converters.newFactory(BTPSourceInfo.class, Converters.BTP_SOURCE_INFO));
        addConverterFactory(Converters.newFactory(ScoreStatus.class, Converters.SCORE_STATUS));
    }

    public void setProvider(Provider provider) {
        this.provider = provider;
    }

    /**
     * Gets the total number of issued coins
     *
     * @return a BigInteger object of the total number of coins in loop
     */
    public Request<BigInteger> getTotalSupply() {
        return getTotalSupply(null);
    }

    /**
     * Gets the total number of issued coins
     *
     * @param height the block number
     * @return a BigInteger object of the total number of coins in loop
     */
    public Request<BigInteger> getTotalSupply(BigInteger height) {
        long requestId = System.currentTimeMillis();
        RpcObject params = null;
        if (height != null) {
            params = new RpcObject.Builder()
                    .put("height", new RpcValue(height))
                    .build();
        }
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_getTotalSupply", params);
        return provider.request(request, findConverter(BigInteger.class));
    }

    /**
     * Gets the balance of an address
     *
     * @param address the address to check the balance
     * @return a BigInteger object of the current balance for the given address in loop
     */
    public Request<BigInteger> getBalance(Address address) {
        return getBalance(address, null);
    }

    /**
     * Gets the balance of an address
     *
     * @param address the address to check the balance
     * @param height the block number
     * @return a BigInteger object of the current balance for the given address in loop
     */
    public Request<BigInteger> getBalance(Address address, BigInteger height) {
        long requestId = System.currentTimeMillis();
        RpcObject.Builder builder = new RpcObject.Builder()
                .put("address", new RpcValue(address));
        if (height != null) {
            builder.put("height", new RpcValue(height));
        }
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_getBalance", builder.build());
        return provider.request(request, findConverter(BigInteger.class));
    }

    /**
     * Gets a block matching the block number
     *
     * @param height the block number
     * @return a {@code Block} object
     */
    public Request<Block> getBlock(BigInteger height) {
        long requestId = System.currentTimeMillis();
        RpcObject params = new RpcObject.Builder()
                .put("height", new RpcValue(height))
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_getBlockByHeight", params);
        return provider.request(request, findConverter(Block.class));
    }

    /**
     * Gets a block matching the block hash
     *
     * @param hash the block hash
     * @return a {@code Block} object
     */
    public Request<Block> getBlock(Bytes hash) {
        long requestId = System.currentTimeMillis();
        RpcObject params = new RpcObject.Builder()
                .put("hash", new RpcValue(hash))
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_getBlockByHash", params);
        return provider.request(request, findConverter(Block.class));
    }

    /**
     * Gets the last block
     *
     * @return a {@code Block} object
     */
    public Request<Block> getLastBlock() {
        long requestId = System.currentTimeMillis();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_getLastBlock", null);
        return provider.request(request, findConverter(Block.class));
    }

    /**
     * Gets information about the APIs in SCORE
     *
     * @param scoreAddress the address to get APIs
     * @return a {@code ScoreApi} object
     */
    public Request<List<ScoreApi>> getScoreApi(Address scoreAddress) {
        return getScoreApi(scoreAddress, null);
    }

    /**
     * Gets information about the APIs in SCORE
     *
     * @param scoreAddress the address to get APIs
     * @param height the block number
     * @return a {@code ScoreApi} object
     */
    @SuppressWarnings("unchecked")
    public Request<List<ScoreApi>> getScoreApi(Address scoreAddress, BigInteger height) {
        if (!IconKeys.isContractAddress(scoreAddress))
            throw new IllegalArgumentException("Only the contract address can be called.");
        long requestId = System.currentTimeMillis();
        RpcObject.Builder builder = new RpcObject.Builder()
                .put("address", new RpcValue(scoreAddress));
        if (height != null) {
            builder.put("height", new RpcValue(height));
        }
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_getScoreApi", builder.build());
        Class<List<ScoreApi>> listClass = ((Class) List.class);
        return provider.request(request, findConverter(listClass));
    }

    /**
     * Gets a transaction matching the given transaction hash
     *
     * @param hash a transaction hash
     * @return a {@code ConfirmedTransaction} object
     */
    public Request<ConfirmedTransaction> getTransaction(Bytes hash) {
        long requestId = System.currentTimeMillis();
        RpcObject params = new RpcObject.Builder()
                .put("txHash", new RpcValue(hash))
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_getTransactionByHash", params);
        return provider.request(request, findConverter(ConfirmedTransaction.class));
    }

    /**
     * Gets the result of a transaction specified by the transaction hash
     *
     * @param hash a transaction hash
     * @return a {@code TransactionResult} object
     */
    public Request<TransactionResult> getTransactionResult(Bytes hash) {
        long requestId = System.currentTimeMillis();
        RpcObject params = new RpcObject.Builder()
                .put("txHash", new RpcValue(hash))
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_getTransactionResult", params);
        return provider.request(request, findConverter(TransactionResult.class));
    }

    /**
     * Calls a SCORE read-only API
     *
     * @param call an instance of Call
     * @param <T> the response type
     * @return a {@code Request} object that can execute the request
     */
    public <T> Request<T> call(Call<T> call) {
        long requestId = System.currentTimeMillis();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_call", call.getProperties());
        return provider.request(request, findConverter(call.responseType()));
    }

    /**
     * Sends a transaction that changes the states of the account
     *
     * @param signedTransaction a transaction that was signed with the sender's wallet
     * @return a {@code Request} object that can execute the request (return type is txHash)
     */
    public Request<Bytes> sendTransaction(SignedTransaction signedTransaction) {
        long requestId = System.currentTimeMillis();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_sendTransaction", signedTransaction.getProperties());
        return provider.request(request, findConverter(Bytes.class));
    }

    /**
     * Gets an estimated step of how much step is necessary to allow the transaction to complete
     *
     * @param transaction a raw transaction without stepLimit and signature information
     * @return a {@code Request} object that can execute the request (return type is BigInteger)
     *
     * @since 0.9.12
     */
    public Request<BigInteger> estimateStep(Transaction transaction) {
        long requestId = System.currentTimeMillis();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "debug_estimateStep", transaction.getProperties());
        return provider.request(request, findConverter(BigInteger.class));
    }

    // Below APIs are additional features for core2

    /**
     * Sends a transaction like {@code sendTransaction}, then waits for some time to get the result.
     * The user may set a specific timeout in the HTTP header, but it cannot exceed the node's max timeout limit.
     * If a timeout did not set by the user, the node uses its {@code defaultWaitTimeout} setting.
     *
     * @param signedTransaction a transaction that was signed with the sender's wallet
     * @return a {@code TransactionResult} object
     */
    public Request<TransactionResult> sendTransactionAndWait(SignedTransaction signedTransaction) {
        long requestId = System.currentTimeMillis();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_sendTransactionAndWait", signedTransaction.getProperties());
        return provider.request(request, findConverter(TransactionResult.class));
    }

    /**
     * Gets the result of a transaction specified by the transaction hash like {@code getTransactionResult},
     * but waits for some time to get the transaction result instead of returning immediately
     * if there is no finalized result.
     * The user may set a specific timeout in the HTTP header, but it cannot exceed the node's max timeout limit.
     * If a timeout did not set by the user, the node uses its {@code defaultWaitTimeout} setting.
     *
     * @param hash a transaction hash
     * @return a {@code TransactionResult} object
     */
    public Request<TransactionResult> waitTransactionResult(Bytes hash) {
        long requestId = System.currentTimeMillis();
        RpcObject params = new RpcObject.Builder()
                .put("txHash", new RpcValue(hash))
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_waitTransactionResult", params);
        return provider.request(request, findConverter(TransactionResult.class));
    }

    /**
     * Retrieves data based on the hash algorithm (SHA3-256)
     * Following data can be retrieved by a hash.
     *  - BlockHeader, Validators, Votes ...
     *
     * @param hash the hash value of the data to retrieve
     * @return a {@code Request} object that can execute the request
     */
    public Request<Base64> getDataByHash(Bytes hash) {
        long requestId = System.currentTimeMillis();
        RpcObject params = new RpcObject.Builder()
                .put("hash", new RpcValue(hash))
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request =
                new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_getDataByHash", params);
        return provider.request(request, findConverter(Base64.class));
    }

    /**
     * Gets block header for specified height
     *
     * @param height the height of the block
     * @return a {@code Request} object that can execute the request
     */
    public Request<Base64> getBlockHeaderByHeight(BigInteger height) {
        long requestId = System.currentTimeMillis();
        RpcObject params = new RpcObject.Builder()
                .put("height", new RpcValue(height))
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_getBlockHeaderByHeight", params);
        return provider.request(request, findConverter(Base64.class));
    }

    /**
     * Gets votes for the block specified by height
     *
     * @param height the height of the block
     * @return a {@code Request} object that can execute the request
     */
    public Request<Base64> getVotesByHeight(BigInteger height) {
        long requestId = System.currentTimeMillis();
        RpcObject params = new RpcObject.Builder()
                .put("height", new RpcValue(height))
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_getVotesByHeight", params);
        return provider.request(request, findConverter(Base64.class));
    }

    /**
     * Gets proof for the receipt
     *
     * @param hash the hash value of the block including the result
     * @param index index of the receipt in the block
     * @return a {@code Request} object that can execute the request
     */
    public Request<Base64[]> getProofForResult(Bytes hash, BigInteger index) {
        long requestId = System.currentTimeMillis();
        RpcObject params = new RpcObject.Builder()
                .put("hash", new RpcValue(hash))
                .put("index", new RpcValue(index))
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_getProofForResult", params);
        return provider.request(request, findConverter(Base64[].class));
    }

    /**
     * Get proof for the receipt and the events in it
     *
     * @param hash the hash value of the block including the result
     * @param index index of the receipt in the block
     * @param events list of indexes of the events in the receipt.
     * @return a {@code Request} object that can execute the request
     */
    public Request<Base64[][]> getProofForEvents(Bytes hash, BigInteger index, BigInteger[] events) {
        long requestId = System.currentTimeMillis();
        RpcArray.Builder arrayBuilder = new RpcArray.Builder();
        for(BigInteger d : events) {
            arrayBuilder.add(new RpcValue(d));
        }
        RpcObject params = new RpcObject.Builder()
                .put("hash", new RpcValue(hash))
                .put("index", new RpcValue(index))
                .put("events", arrayBuilder.build())
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_getProofForEvents", params);
        return provider.request(request, findConverter(Base64[][].class));
    }

    /**
     * Get status of the contract
     *
     * @param contract the address of the contract
     * @return a {@code Request} object that can execute the request
     */
    public Request<ScoreStatus> getScoreStatus(Address contract) {
        long requestId = System.currentTimeMillis();
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(contract))
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "icx_getScoreStatus", params);
        return provider.request(request, findConverter(ScoreStatus.class));
    }

    /**
     * Gets a monitor for block notification
     *
     * @param height the start height
     * @return a {@code Monitor} object
     */
    public Monitor<BlockNotification> monitorBlocks(BigInteger height) {
        MonitorSpec ms = new BlockMonitorSpec(height, null);
        return provider.monitor(ms, findConverter(BlockNotification.class));
    }

    // Below APIs are additional features for BTP 2.0

    /**
     * Gets BTP network information
     *
     * @param id BTP network ID
     * @return a {@code Request} object that can execute the request
     */
    public Request<BTPNetworkInfo> btpGetNetworkInfo(BigInteger id) {
        return btpGetNetworkInfo(null, id);
    }

    /**
     * Gets BTP network information
     *
     * @param height Main block height
     * @param id BTP network ID
     * @return a {@code Request} object that can execute the request
     */
    public Request<BTPNetworkInfo> btpGetNetworkInfo(BigInteger height, BigInteger id) {
        long requestId = System.currentTimeMillis();
        RpcObject.Builder params = new RpcObject.Builder()
                .put("id", new RpcValue(id));
        if (height != null) {
            params.put("height", new RpcValue(height));
        }
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "btp_getNetworkInfo", params.build());
        return provider.request(request, findConverter(BTPNetworkInfo.class));
    }

    /**
     * Gets BTP network type information
     *
     * @param id BTP network type ID
     * @return a {@code Request} object that can execute the request
     */
    public Request<BTPNetworkTypeInfo> btpGetNetworkTypeInfo(BigInteger id) {
        return btpGetNetworkTypeInfo(null, id);
    }
    /**
     * Gets BTP network type information
     *
     * @param height the block number
     * @param id BTP network type ID
     * @return a {@code Request} object that can execute the request
     */
    public Request<BTPNetworkTypeInfo> btpGetNetworkTypeInfo(BigInteger height, BigInteger id) {
        long requestId = System.currentTimeMillis();
        RpcObject.Builder params = new RpcObject.Builder()
                .put("id", new RpcValue(id));
        if (height != null) {
                params.put("height", new RpcValue(height));
        }
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "btp_getNetworkTypeInfo", params.build());
        return provider.request(request, findConverter(BTPNetworkTypeInfo.class));
    }

    /**
     * Gets BTP messages
     *
     * @param height the block number
     * @param networkID BTP network ID
     * @return a {@code Request} object that can execute the request
     */
    public Request<Base64[]> btpGetMessages(BigInteger height, BigInteger networkID) {
        long requestId = System.currentTimeMillis();
        RpcObject params = new RpcObject.Builder()
                .put("height", new RpcValue(height))
                .put("networkID", new RpcValue(networkID))
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "btp_getMessages", params);
        return provider.request(request, findConverter(Base64[].class));
    }

    /**
     * Gets BTP block header
     *
     * @param height the block number
     * @param networkID BTP network ID
     * @return a {@code Request} object that can execute the request
     */
    public Request<Base64> btpGetHeader(BigInteger height, BigInteger networkID) {
        long requestId = System.currentTimeMillis();
        RpcObject params = new RpcObject.Builder()
                .put("height", new RpcValue(height))
                .put("networkID", new RpcValue(networkID))
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "btp_getHeader", params);
        return provider.request(request, findConverter(Base64.class));
    }

    /**
     * Gets BTP block proof
     *
     * @param height the block number
     * @param networkID BTP network ID
     * @return a {@code Request} object that can execute the request
     */
    public Request<Base64> btpGetProof(BigInteger height, BigInteger networkID) {
        long requestId = System.currentTimeMillis();
        RpcObject params = new RpcObject.Builder()
                .put("height", new RpcValue(height))
                .put("networkID", new RpcValue(networkID))
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "btp_getProof", params);
        return provider.request(request, findConverter(Base64.class));
    }

    /**
     * Gets the last block
     *
     * @return a {@code Block} object
     */
    public Request<BTPSourceInfo> btpGetSourceInformation() {
        long requestId = System.currentTimeMillis();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(
                requestId, "btp_getSourceInformation", null);
        return provider.request(request, findConverter(BTPSourceInfo.class));
    }

    /**
     * Gets a monitor for block notification
     *
     * @param height the start height
     * @param eventFilters array of eventFilter
     * @return a {@code Monitor} object
     */
    public Monitor<BlockNotification> monitorBlocks(BigInteger height, EventMonitorSpec.EventFilter[] eventFilters) {
        MonitorSpec ms = new BlockMonitorSpec(height, eventFilters);
        return provider.monitor(ms, findConverter(BlockNotification.class));
    }

    /**
     * Gets a monitor for event notification
     *
     * @param height the start height
     * @param event the event signature
     * @param addr the address of SCORE
     * @param indexed the array of arguments to match with indexed parameters of event
     * @param data the array of arguments to match with non-indexed parameters of event
     * @return a {@code Monitor} object
     */
    public Monitor<EventNotification> monitorEvents(BigInteger height, String event, Address addr, String[] indexed, String[] data) {
        MonitorSpec ms = new EventMonitorSpec(height, event, addr, indexed, data);
        return provider.monitor(ms, findConverter(EventNotification.class));
    }

    @SuppressWarnings("unchecked")
    private <T> RpcConverter<T> findConverter(Class<T> type) {
        RpcConverter<T> converter = (RpcConverter<T>) converterMap.get(type);
        if (converter != null) return converter;

        for (RpcConverterFactory factory : converterFactories) {
            converter = factory.create(type);
            if (converter != null) {
                converterMap.put(type, converter);
                return converter;
            }
        }

        if (type.isAnnotationPresent(AnnotationConverter.class)) {
            if (type.getAnnotation(AnnotationConverter.class).use()) {
                return new AnnotatedConverterFactory().create(type);
            }
        }

        throw new IllegalArgumentException("Could not locate response converter for:'" + type + "'");
    }

    /**
     * Gets a monitor for BTP notification
     *
     * @param height the start height
     * @param networkId the btp network id
     * @param proofFlag Proof Included for BTP Header
     * @return a {@code Monitor} object
     *
     */
    public Monitor<BlockNotification> monitorBTP(BigInteger height, BigInteger networkId, BigInteger proofFlag) {
        MonitorSpec ms = new BTPMonitorSpec(height, networkId, proofFlag);
        return provider.monitor(ms, findConverter(BlockNotification.class));
    }

    /**
     * Adds Converter factory.
     * It has a create function that creates the converter of the specific type.
     *
     * @param factory a converter factory
     */
    public void addConverterFactory(RpcConverterFactory factory) {
        converterFactories.add(factory);
    }
}
