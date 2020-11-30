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

import example.SampleToken;
import foundation.icon.ee.types.Method;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.ScoreApi;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;

import java.io.IOException;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static foundation.icon.test.common.Env.LOG;

public class SampleTokenScore extends Score {
    private static final Map<String, Method> expectedMap = new HashMap<>() {{
        put("name", Method.newFunction("name", Method.Flags.READONLY | Method.Flags.EXTERNAL,
                null, Method.DataType.STRING, "str"));
        put("symbol", Method.newFunction("symbol", Method.Flags.READONLY | Method.Flags.EXTERNAL,
                null, Method.DataType.STRING, "str"));
        put("decimals", Method.newFunction("decimals", Method.Flags.READONLY | Method.Flags.EXTERNAL,
                null, Method.DataType.INTEGER, "int"));
        put("totalSupply", Method.newFunction("totalSupply", Method.Flags.READONLY | Method.Flags.EXTERNAL,
                null, Method.DataType.INTEGER, "int"));
        put("balanceOf", Method.newFunction("balanceOf", Method.Flags.READONLY | Method.Flags.EXTERNAL,
                new Method.Parameter[] {
                        new Method.Parameter("_owner", "Address", Method.DataType.ADDRESS)
                }, Method.DataType.INTEGER, "int"));
        put("transfer", Method.newFunction("transfer", Method.Flags.EXTERNAL, 1,
                new Method.Parameter[] {
                        new Method.Parameter("_to", "Address", Method.DataType.ADDRESS),
                        new Method.Parameter("_value", "int", Method.DataType.INTEGER),
                        new Method.Parameter("_data", "bytes", Method.DataType.BYTES, true)
                }, Method.DataType.NONE, null));
        put("Transfer", Method.newEvent("Transfer", 3,
                new Method.Parameter[] {
                        new Method.Parameter("_from", "Address", Method.DataType.ADDRESS),
                        new Method.Parameter("_to", "Address", Method.DataType.ADDRESS),
                        new Method.Parameter("_value", "int", Method.DataType.INTEGER),
                        new Method.Parameter("_data", "bytes", Method.DataType.BYTES)
                }));
    }};

    private static RpcObject getParams(BigInteger decimals, BigInteger initialSupply) {
        return new RpcObject.Builder()
                .put("_name", new RpcValue("MySampleToken"))
                .put("_symbol", new RpcValue("MST"))
                .put("_decimals", new RpcValue(decimals))
                .put("_initialSupply", new RpcValue(initialSupply))
                .build();
    }

    public static SampleTokenScore mustDeploy(TransactionHandler txHandler, Wallet owner,
                                              BigInteger decimals, BigInteger initialSupply, String contentType)
            throws ResultTimeoutException, TransactionFailureException, IOException {
        LOG.infoEntering("deploy", "SampleToken");
        Score score;
        if (contentType.equals(Constants.CONTENT_TYPE_PYTHON)) {
            score = txHandler.deploy(owner, getFilePath("sample_token"), getParams(decimals, initialSupply));
        } else if (contentType.equals(Constants.CONTENT_TYPE_JAVA)) {
            score = txHandler.deploy(owner, SampleToken.class, getParams(decimals, initialSupply));
        } else {
            throw new IllegalArgumentException("Unknown content type");
        }
        LOG.info("scoreAddr = " + score.getAddress());
        LOG.infoExiting();
        return new SampleTokenScore(score);
    }

    public static SampleTokenScore mustDeploy(TransactionHandler txHandler, Wallet owner,
                                              BigInteger decimals, BigInteger initialSupply, Class<?>[] classes)
            throws ResultTimeoutException, TransactionFailureException, IOException {
        LOG.infoEntering("deploy", classes[0].getName());
        Score score = txHandler.deploy(owner, classes, getParams(decimals, initialSupply));
        LOG.info("scoreAddr = " + score.getAddress());
        LOG.infoExiting();
        return new SampleTokenScore(score);
    }

    public SampleTokenScore(Score other) throws IOException {
        super(other);
        if (!checkIfConformIRC2()) {
            throw new IllegalArgumentException("Not an IRC2 token contract");
        }
    }

    private boolean checkIfConformIRC2() throws IOException {
        Map<String, Method> copyMap = new HashMap<>(expectedMap);
        for (ScoreApi api : getScoreApi()) {
            Method m = expectedMap.get(api.getName());
            if (m != null) {
                if (("function".equals(api.getType()) && m.getType() != Method.MethodType.FUNCTION)
                        || ("eventlog".equals(api.getType()) && m.getType() != Method.MethodType.EVENT)
                        || ("fallback".equals(api.getType()))) {
                    return false;
                }
                if ((api.getReadonly() == null && (m.getFlags() & Method.Flags.READONLY) != 0)
                        || (api.getReadonly() != null) && (m.getFlags() & Method.Flags.READONLY) == 0) {
                    return false;
                }
                List<ScoreApi.Param> inputs = api.getInputs();
                if (inputs.size() != (m.getInputs() != null ? m.getInputs().length : 0)) {
                    return false;
                }
                List<ScoreApi.Param> matched = new ArrayList<>();
                for (ScoreApi.Param sp : inputs) {
                    for (Method.Parameter mp : m.getInputs()) {
                        if (sp.getName().equals(mp.getName()) && sp.getType().equals(mp.getDescriptor())) {
                            matched.add(sp);
                            if (mp.isOptional() && (sp.getDefault() == null || !sp.getDefault().isNull())) {
                                return false;
                            }
                        }
                    }
                }
                inputs.removeAll(matched);
                if (!inputs.isEmpty()) {
                    return false;
                }
                List<ScoreApi.Param> outputs = api.getOutputs();
                if (m.getOutput() == 0 && outputs.size() != 0) {
                    return false;
                }
                if (m.getOutput() != 0 && !m.getOutputDescriptor().equals(outputs.get(0).getType())) {
                    return false;
                }
                copyMap.remove(m.getName());
            }
        }
        return copyMap.isEmpty();
    }

    public BigInteger balanceOf(Address owner) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_owner", new RpcValue(owner))
                .build();
        return call("balanceOf", params).asInteger();
    }

    public TransactionResult transfer(Wallet wallet, Address to, BigInteger value)
            throws IOException, ResultTimeoutException {
        return this.transfer(wallet, to, value, null);
    }

    public TransactionResult transfer(Wallet wallet, Address to, BigInteger value, byte[] data)
            throws IOException, ResultTimeoutException {
        RpcObject.Builder builder = new RpcObject.Builder()
                .put("_to", new RpcValue(to))
                .put("_value", new RpcValue(value));
        if (data != null) {
            builder.put("_data", new RpcValue(data));
        }
        return this.invokeAndWaitResult(wallet, "transfer", builder.build());
    }

    public void ensureTransfer(TransactionResult result, Address from, Address to, BigInteger value, byte[] data)
            throws IOException {
        TransactionResult.EventLog event = findEventLog(result, "Transfer(Address,Address,int,bytes)");
        if (event != null) {
            if (data == null) {
                data = new byte[0];
            }
            Address _from = event.getIndexed().get(1).asAddress();
            Address _to = event.getIndexed().get(2).asAddress();
            BigInteger _value = event.getIndexed().get(3).asInteger();
            byte[] _data = event.getData().get(0).asByteArray();
            if (from.equals(_from) && to.equals(_to) && value.equals(_value) && Arrays.equals(data, _data)) {
                return; // ensured
            }
        }
        throw new IOException("ensureTransfer failed.");
    }

    public void ensureTokenBalance(Address owner, long value) throws ResultTimeoutException, IOException {
        long limitTime = System.currentTimeMillis() + Constants.DEFAULT_WAITING_TIME;
        while (true) {
            BigInteger balance = balanceOf(owner);
            String msg = "Token balance of " + owner + ": " + balance;
            if (balance.equals(BigInteger.valueOf(0))) {
                try {
                    if (limitTime < System.currentTimeMillis()) {
                        throw new ResultTimeoutException();
                    }
                    // wait until block confirmation
                    LOG.info(msg + "; Retry in 1 sec.");
                    Thread.sleep(1000);
                } catch (InterruptedException e) {
                    e.printStackTrace();
                }
            } else if (balance.equals(BigInteger.valueOf(value).multiply(BigInteger.TEN.pow(18)))) {
                LOG.info(msg);
                break;
            } else {
                throw new IOException("Token balance mismatch!");
            }
        }
    }
}
