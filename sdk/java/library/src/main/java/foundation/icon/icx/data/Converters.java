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

import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcConverter;
import foundation.icon.icx.transport.jsonrpc.RpcConverter.RpcConverterFactory;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcItemCreator;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;

@SuppressWarnings("unchecked")
public final class Converters {
    private Converters() {
    }

    public static final RpcConverter<RpcItem> RPC_ITEM
            = new RpcConverter<RpcItem>() {

        @Override
        public RpcItem convertTo(RpcItem object) {
            return object;
        }

        @Override
        public RpcItem convertFrom(RpcItem object) {
            return object;
        }
    };

    public static final RpcConverter<BigInteger> BIG_INTEGER
            = new RpcConverter<BigInteger>() {

        @Override
        public BigInteger convertTo(RpcItem object) {
            return object.asInteger();
        }

        @Override
        public RpcItem convertFrom(BigInteger object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<Boolean> BOOLEAN
            = new RpcConverter<Boolean>() {

        @Override
        public Boolean convertTo(RpcItem object) {
            return object.asBoolean();
        }

        @Override
        public RpcItem convertFrom(Boolean object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<String> STRING
            = new RpcConverter<String>() {

        @Override
        public String convertTo(RpcItem object) {
            return object.asString();
        }

        @Override
        public RpcItem convertFrom(String object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<Bytes> BYTES
            = new RpcConverter<Bytes>() {

        @Override
        public Bytes convertTo(RpcItem object) {
            return object.asBytes();
        }

        @Override
        public RpcItem convertFrom(Bytes object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<byte[]> BYTE_ARRAY
            = new RpcConverter<byte[]>() {

        @Override
        public byte[] convertTo(RpcItem object) {
            return object.asByteArray();
        }

        @Override
        public RpcItem convertFrom(byte[] object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<Block> BLOCK = new RpcConverter<Block>() {

        @Override
        public Block convertTo(RpcItem object) {
            return new Block(object.asObject());
        }

        @Override
        public RpcItem convertFrom(Block object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<ConfirmedTransaction> CONFIRMED_TRANSACTION
            = new RpcConverter<ConfirmedTransaction>() {

        @Override
        public ConfirmedTransaction convertTo(RpcItem object) {
            return new ConfirmedTransaction(object.asObject());
        }

        @Override
        public RpcItem convertFrom(ConfirmedTransaction object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<TransactionResult> TRANSACTION_RESULT
            = new RpcConverter<TransactionResult>() {

        @Override
        public TransactionResult convertTo(RpcItem object) {
            return new TransactionResult(object.asObject());
        }

        @Override
        public RpcItem convertFrom(TransactionResult object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<BlockNotification> BLOCK_NOTIFICATION
            = new RpcConverter<BlockNotification>() {
        @Override
        public BlockNotification convertTo(RpcItem object) {
            return new BlockNotification(object.asObject());
        }

        @Override
        public RpcItem convertFrom(BlockNotification object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<EventNotification> EVENT_NOTIFICATION
            = new RpcConverter<EventNotification>() {
        @Override
        public EventNotification convertTo(RpcItem object) {
            return new EventNotification(object.asObject());
        }

        @Override
        public RpcItem convertFrom(EventNotification object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<Base64[][]> BASE64_ARRAY_ARRAY
            = new RpcConverter<Base64[][]>() {
        @Override
        public Base64[][] convertTo(RpcItem rpcItem) {
            RpcArray arrayArray = rpcItem.asArray();
            Base64[][] base64ArrayArray = new Base64[arrayArray.size()][];
            for (int i = 0; i < arrayArray.size(); i++) {
                RpcArray array = arrayArray.get(i).asArray();
                Base64[] base64Array = new Base64[array.size()];
                for (int j = 0; j < array.size(); j++) {
                    base64Array[j] = new Base64(array.get(i).asString());
                }
                base64ArrayArray[i] = base64Array;
            }
            return base64ArrayArray;
        }

        @Override
        public RpcItem convertFrom(Base64[][] object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<Base64[]> BASE64_ARRAY
            = new RpcConverter<Base64[]>() {
        @Override
        public Base64[] convertTo(RpcItem rpcItem) {
            RpcArray array = rpcItem.asArray();
            Base64[] base64Array = new Base64[array.size()];
            for (int i = 0; i < array.size(); i++) {
                base64Array[i] = new Base64(array.get(i).asString());
            }
            return base64Array;
        }

        @Override
        public RpcItem convertFrom(Base64[] object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<Base64> BASE64
            = new RpcConverter<Base64>() {
        @Override
        public Base64 convertTo(RpcItem rpcItem) {
            return new Base64(rpcItem.asString());
        }

        @Override
        public RpcItem convertFrom(Base64 object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<List<ScoreApi>> SCORE_API_LIST
            = new RpcConverter<List<ScoreApi>>() {

        @Override
        public List<ScoreApi> convertTo(RpcItem rpcItem) {
            RpcArray array = rpcItem.asArray();
            List<ScoreApi> scoreApis = new ArrayList<>(array.size());
            for (int i = 0; i < array.size(); i++) {
                scoreApis.add(new ScoreApi(array.get(i).asObject()));
            }
            return scoreApis;
        }

        @Override
        public RpcItem convertFrom(List<ScoreApi> object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<BTPNetworkInfo> BTP_NETWORK_INFO
            = new RpcConverter<BTPNetworkInfo>() {

        @Override
        public BTPNetworkInfo convertTo(RpcItem rpcItem) {
            return new BTPNetworkInfo(rpcItem.asObject());
        }

        @Override
        public RpcItem convertFrom(BTPNetworkInfo object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<BTPNetworkTypeInfo> BTP_NETWORK_TYPE_INFO
            = new RpcConverter<BTPNetworkTypeInfo>() {

        @Override
        public BTPNetworkTypeInfo convertTo(RpcItem rpcItem) {
            return new BTPNetworkTypeInfo(rpcItem.asObject());
        }

        @Override
        public RpcItem convertFrom(BTPNetworkTypeInfo object) {
            return RpcItemCreator.create(object);
        }
    };

    public static final RpcConverter<BTPSourceInfo> BTP_SOURCE_INFO
            = new RpcConverter<BTPSourceInfo>() {

        @Override
        public BTPSourceInfo convertTo(RpcItem rpcItem) {
            return new BTPSourceInfo(rpcItem.asObject());
        }

        @Override
        public RpcItem convertFrom(BTPSourceInfo object) {
            return RpcItemCreator.create(object);
        }
    public static final RpcConverter<ScoreStatus> SCORE_STATUS = new RpcConverter<ScoreStatus>() {
        @Override
        public ScoreStatus convertTo(RpcItem object) {
            return new ScoreStatus(object.asObject());
        }

        @Override
        public RpcItem convertFrom(ScoreStatus object) {
            return RpcItemCreator.create(object);
        }
    };

    public static <TT> RpcConverterFactory newFactory(
            final Class<TT> typeFor, final RpcConverter<TT> converter) {
        return new RpcConverterFactory() {
            @Override
            public <T> RpcConverter<T> create(Class<T> type) {
                return type.isAssignableFrom(typeFor) ? (RpcConverter<T>) converter : null;
            }
        };
    }

    public static Object fromRpcItem(RpcItem item, Class<?> type) {
        if (item == null) return null;
        if (item.getClass().isAssignableFrom(RpcArray.class)) {
            return fromRpcArray((RpcArray) item, type);
        }
        if (item.getClass().isAssignableFrom(RpcObject.class)) {
            return fromRpcObject((RpcObject) item, type);
        }
        return fromRpcValue((RpcValue) item, type);
    }

    static Object fromRpcArray(RpcArray array, Class<?> type) {
        if (type.isAssignableFrom(RpcArray.class)) return array;
        List<Object> result = new ArrayList<>();
        for (RpcItem item : array) {
            Object v = fromRpcItem(item, type);
            if (v != null) result.add(fromRpcItem(item, type));
        }
        return result;
    }

    static Object fromRpcObject(RpcObject object, Class<?> type) {
        if (type.isAssignableFrom(RpcObject.class)) return object;
        Map<String, Object> result = new HashMap<>();
        Set<String> keys = object.keySet();
        for (String key : keys) {
            Object v = fromRpcItem(object.getItem(key), type);
            if (v != null) result.put(key, v);
        }
        return result;
    }

    static Object fromRpcValue(RpcValue value, Class<?> type) {
        if (type.isAssignableFrom(Boolean.class) || type.isAssignableFrom(boolean.class)) {
            return value.asBoolean();
        } else if (type.isAssignableFrom(String.class)) {
            return value.asString();
        } else if (type.isAssignableFrom(BigInteger.class)) {
            return value.asInteger();
        } else if (type.isAssignableFrom(byte[].class)) {
            return value.asByteArray();
        } else if (type.isAssignableFrom(Bytes.class)) {
            return value.asBytes();
        } else if (type.isAssignableFrom(Address.class)) {
            return value.asAddress();
        } else if (type.isAssignableFrom(RpcItem.class)) {
            return value;
        }
        return null;
    }
}
