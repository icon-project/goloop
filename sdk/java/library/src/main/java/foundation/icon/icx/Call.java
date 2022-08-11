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
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcItemCreator;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;

import java.math.BigInteger;

import static foundation.icon.icx.TransactionBuilder.checkArgument;

/**
 * Call contains parameters for querying request.
 *
 * @param <T> Response type
 */
public final class Call<T> {

    private final RpcObject properties;
    private final Class<T> responseType;

    private Call(RpcObject properties, Class<T> responseType) {
        this.properties = properties;
        this.responseType = responseType;
    }

    RpcObject getProperties() {
        return properties;
    }

    Class<T> responseType() {
        return responseType;
    }

    /**
     * Builder for creating immutable object of Call.<br>
     * It has following properties<br>
     * - {@link #from(Address)} the request account<br>
     * - {@link #to(Address)} the SCORE address to call<br>
     * - {@link #method(String)}  the method name to call<br>
     * - {@link #params(Object)}  the parameter of call<br>
     */
    @SuppressWarnings("WeakerAccess")
    public static class Builder {
        private Address from;
        private Address to;
        private String method;
        private BigInteger height;
        private RpcItem params;

        public Builder() {
        }

        public Builder from(Address from) {
            this.from = from;
            return this;
        }

        public Builder to(Address to) {
            if (!IconKeys.isContractAddress(to))
                throw new IllegalArgumentException("Only the contract address can be called.");
            this.to = to;
            return this;
        }

        public Builder method(String method) {
            this.method = method;
            return this;
        }

        public Builder height(BigInteger height) {
            this.height = height;
            return this;
        }

        public <T> Builder params(T params) {
            this.params = RpcItemCreator.create(params);
            return this;
        }

        public Builder params(RpcItem params) {
            this.params = params;
            return this;
        }

        /**
         * Builds with RpcItem. that means the return type is RpcItem
         *
         * @return Call
         */
        public Call<RpcItem> build() {
            checkArgument(to, "to not found");
            checkArgument(method, "method not found");
            return buildWith(RpcItem.class);
        }

        /**
         * Builds with User defined class. an object of the class would be returned
         *
         * @param responseType Response type
         * @param <T> responseType
         * @return Call
         */
        public <T> Call<T> buildWith(Class<T> responseType) {
            RpcObject data = new RpcObject.Builder()
                    .put("method", new RpcValue(method))
                    .put("params", params)
                    .build();

            RpcObject.Builder propertiesBuilder = new RpcObject.Builder()
                    .put("to", new RpcValue(to))
                    .put("data", data)
                    .put("dataType", new RpcValue("call"));

            // optional
            if (from != null) {
                propertiesBuilder.put("from", new RpcValue(from));
            }
            if (height != null) {
                propertiesBuilder.put("height", new RpcValue(height));
            }

            return new Call<>(propertiesBuilder.build(), responseType);
        }
    }
}
