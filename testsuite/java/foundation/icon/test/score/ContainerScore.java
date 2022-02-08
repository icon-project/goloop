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

package foundation.icon.test.score;

import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.transport.jsonrpc.RpcArray;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;
import testcases.ContainerDB;

import java.io.IOException;
import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;

public class ContainerScore extends Score {
    public static final String T_INT = "int";
    public static final String T_STRING = "str";
    public static final String T_BYTES = "bytes";
    public static final String T_BOOL = "bool";
    public static final String T_ADDRESS = "addr";
    private static final Class<?>[] javaClasses = new Class<?>[]{ContainerDB.class};
    private final Wallet owner;

    public ContainerScore(Score other, Wallet owner) {
        super(other);
        this.owner = owner;
    }

    public static ContainerScore mustDeploy(TransactionHandler txHandler, Wallet owner)
            throws TransactionFailureException, IOException, ResultTimeoutException {
        LOG.infoEntering("deploy", "ContainerScore");
        Score score;
        score = txHandler.deploy(owner, getFilePath("container_db"), null);
        LOG.info("scoreAddr = " + score.getAddress());
        LOG.infoExiting();
        return new ContainerScore(score, owner);
    }

    public Bytes updateToJavaScore() throws IOException {
        return updateScore(owner, javaClasses, null);
    }

    public Bytes setVar(Object value) throws IOException {
        if (value instanceof BigInteger) {
            return invoke(owner, "setVar",
                    (new RpcObject.Builder())
                            .put("v_int", new RpcValue((BigInteger) value))
                            .build());
        } else if (value instanceof String) {
            return invoke(owner, "setVar",
                    (new RpcObject.Builder())
                            .put("v_str", new RpcValue((String) value))
                            .build());
        } else if (value instanceof byte[]) {
            return invoke(owner, "setVar",
                    (new RpcObject.Builder())
                            .put("v_bytes", new RpcValue((byte[]) value))
                            .build());
        } else if (value instanceof Boolean) {
            return invoke(owner, "setVar",
                    (new RpcObject.Builder())
                            .put("v_bool", new RpcValue((Boolean) value))
                            .build());
        } else if (value instanceof Address) {
            return invoke(owner, "setVar",
                    (new RpcObject.Builder())
                            .put("v_addr", new RpcValue((Address) value))
                            .build());
        }
        throw new IllegalArgumentException("Not supported");
    }

    public Bytes setDict(String key, Object value) throws IOException {
        if (value instanceof BigInteger) {
            return invoke(owner, "setDict",
                    (new RpcObject.Builder())
                            .put("key", new RpcValue(key))
                            .put("v_int", new RpcValue((BigInteger) value))
                            .build());
        } else if (value instanceof String) {
            return invoke(owner, "setDict",
                    (new RpcObject.Builder())
                            .put("key", new RpcValue(key))
                            .put("v_str", new RpcValue((String) value))
                            .build());
        } else if (value instanceof byte[]) {
            return invoke(owner, "setDict",
                    (new RpcObject.Builder())
                            .put("key", new RpcValue(key))
                            .put("v_bytes", new RpcValue((byte[]) value))
                            .build());
        } else if (value instanceof Boolean) {
            return invoke(owner, "setDict",
                    (new RpcObject.Builder())
                            .put("key", new RpcValue(key))
                            .put("v_bool", new RpcValue((Boolean) value))
                            .build());
        } else if (value instanceof Address) {
            return invoke(owner, "setDict",
                    (new RpcObject.Builder())
                            .put("key", new RpcValue(key))
                            .put("v_addr", new RpcValue((Address) value))
                            .build());
        }
        throw new IllegalArgumentException("Not supported");
    }

    public Bytes setArray(Object value) throws IOException {
        if (value instanceof BigInteger) {
            return invoke(owner, "setArray",
                    (new RpcObject.Builder())
                            .put("v_int", new RpcValue((BigInteger) value))
                            .build());
        } else if (value instanceof String) {
            return invoke(owner, "setArray",
                    (new RpcObject.Builder())
                            .put("v_str", new RpcValue((String) value))
                            .build());
        } else if (value instanceof byte[]) {
            return invoke(owner, "setArray",
                    (new RpcObject.Builder())
                            .put("v_bytes", new RpcValue((byte[]) value))
                            .build());
        } else if (value instanceof Boolean) {
            return invoke(owner, "setArray",
                    (new RpcObject.Builder())
                            .put("v_bool", new RpcValue((Boolean) value))
                            .build());
        } else if (value instanceof Address) {
            return invoke(owner, "setArray",
                    (new RpcObject.Builder())
                            .put("v_addr", new RpcValue((Address) value))
                            .build());
        }
        throw new IllegalArgumentException("Not supported");
    }

    public Object getVar(String type) throws IOException {
        var result = this.call("getVar",
                (new RpcObject.Builder())
                        .put("type", new RpcValue(type))
                        .build());
        if (T_INT.equals(type)) {
            return result.asObject().getItem(T_INT).asInteger();
        } else if (T_STRING.equals(type)) {
            return result.asObject().getItem(T_STRING).asString();
        } else if (T_BYTES.equals(type)) {
            return result.asObject().getItem(T_BYTES).asByteArray();
        } else if (T_BOOL.equals(type)) {
            return result.asObject().getItem(T_BOOL).asBoolean();
        } else if (T_ADDRESS.equals(type)) {
            return result.asObject().getItem(T_ADDRESS).asAddress();
        }
        throw new IllegalArgumentException("Not supported");
    }

    public Object getDict(String key, String type) throws IOException {
        var result = this.call("getDict",
                (new RpcObject.Builder())
                        .put("key", new RpcValue(key))
                        .put("type", new RpcValue(type))
                        .build());
        if (T_INT.equals(type)) {
            return result.asObject().getItem(key).asInteger();
        } else if (T_STRING.equals(type)) {
            return result.asObject().getItem(key).asString();
        } else if (T_BYTES.equals(type)) {
            return result.asObject().getItem(key).asByteArray();
        } else if (T_BOOL.equals(type)) {
            return result.asObject().getItem(key).asBoolean();
        } else if (T_ADDRESS.equals(type)) {
            return result.asObject().getItem(key).asAddress();
        }
        throw new IllegalArgumentException("Not supported");
    }

    public RpcArray getArray(String type) throws IOException {
        var result = this.call("getArray",
                (new RpcObject.Builder())
                        .put("type", new RpcValue(type))
                        .build());
        return result.asArray();
    }
}
