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


public class ScoreApi {

    private RpcObject properties;

    ScoreApi(RpcObject properties) {
        this.properties = properties;
    }

    public RpcObject getProperties() {
        return properties;
    }

    public String getType() {
        RpcItem item = properties.getItem("type");
        return item != null ? item.asString() : null;
    }

    public String getName() {
        RpcItem item = properties.getItem("name");
        return item != null ? item.asString() : null;
    }

    public List<Param> getInputs() {
        return getParams(properties.getItem("inputs"));
    }

    public List<Param> getOutputs() {
        return getParams(properties.getItem("outputs"));
    }

    List<Param> getParams(RpcItem item) {
        List<Param> params = new ArrayList<>();
        if (item != null) {
            for (RpcItem rpcItem : item.asArray()) {
                RpcObject object = (RpcObject) rpcItem;
                params.add(new Param(object));
            }
        }
        return params;
    }

    public String getReadonly() {
        RpcItem item = properties.getItem("readonly");
        return item != null ? item.asString() : null;
    }

    @Override
    public String toString() {
        return "ScoreApi{" +
                "properties=" + properties +
                '}';
    }

    public class Param {
        private RpcObject properties;

        Param(RpcObject properties) {
            this.properties = properties;
        }

        public String getType() {
            RpcItem item = properties.getItem("type");
            return item != null ? item.asString() : null;
        }

        public String getName() {
            RpcItem item = properties.getItem("name");
            return item != null ? item.asString() : null;
        }

        public BigInteger getIndexed() {
            RpcItem item = properties.getItem("indexed");
            return item != null ? item.asInteger() : null;
        }

        @Override
        public String toString() {
            return "Param{" +
                    "properties=" + properties +
                    '}';
        }
    }
}
