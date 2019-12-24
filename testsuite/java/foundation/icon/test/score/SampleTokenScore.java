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
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.IconAmount;
import foundation.icon.icx.data.ScoreApi;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.common.Utils;

import java.io.IOException;
import java.math.BigDecimal;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static foundation.icon.test.common.Env.LOG;

public class SampleTokenScore extends Score {
    private static Map<String, Method> expectedMap = new HashMap<>() {{
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

    public static SampleTokenScore mustDeploy(TransactionHandler txHandler, Wallet owner,
                                              BigInteger decimals, BigInteger initialSupply, String contentType)
            throws ResultTimeoutException, TransactionFailureException, IOException {
        LOG.infoEntering("deploy", "SampleToken");
        RpcObject params = new RpcObject.Builder()
                .put("_name", new RpcValue("MySampleToken"))
                .put("_symbol", new RpcValue("MST"))
                .put("_decimals", new RpcValue(decimals))
                .put("_initialSupply", new RpcValue(initialSupply))
                .build();
        Score score;
        if (contentType.equals(Constants.CONTENT_TYPE_PYTHON)) {
            score = txHandler.deploy(owner, Constants.SCORE_SAMPLETOKEN_PATH, params);
        } else if (contentType.equals(Constants.CONTENT_TYPE_JAVA)) {
            score = txHandler.deploy(owner, SampleToken.class, params);
        } else {
            throw new IllegalArgumentException("Unknown content type");
        }
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
            } else if (balance.equals(BigInteger.valueOf(value).multiply(BigDecimal.TEN.pow(18).toBigInteger()))) {
                LOG.info(msg);
                break;
            } else {
                throw new IOException("Token balance mismatch!");
            }
        }
    }

    public Bytes transfer(Wallet fromWallet, Address toAddress, BigInteger value) throws IOException {
        RpcObject params = new RpcObject.Builder()
                .put("_to", new RpcValue(toAddress))
                .put("_value", new RpcValue(IconAmount.of(value, 18).toLoop()))
                .build();
        return this.invoke(fromWallet, "transfer", params, null, Constants.DEFAULT_STEPS,
                Utils.getMicroTime(), BigInteger.ONE);
    }

    public void ensureFundTransfer(TransactionResult result, Address scoreAddress,
                                   Address backer, BigInteger amount) throws IOException {
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, scoreAddress, "FundTransfer(Address,int,bool)");
        if (event != null) {
            Address _backer = event.getIndexed().get(1).asAddress();
            BigInteger _amount = event.getIndexed().get(2).asInteger();
            Boolean isContribution = event.getIndexed().get(3).asBoolean();
            if (backer.equals(_backer) && amount.equals(_amount) && !isContribution) {
                return; // ensured
            }
        }
        throw new IOException("ensureFundTransfer failed.");
    }
}
