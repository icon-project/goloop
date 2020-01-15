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

package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.Converters;
import foundation.icon.icx.data.ScoreApi;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcError;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.StepCounterScore;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.math.BigInteger;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.fail;

/*
test methods
    testGetAPIForStepCounter
    validateGetScoreApi
 */
@Tag(Constants.TAG_PY_SCORE)
class GetAPITest {
    private static Env.Chain chain;
    private static IconService iconService;

    @BeforeAll
    static void init() {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
    }

    private static final String TYPE_FUNCTION = "function";
    private static final String TYPE_FALLBACK = "fallback";
    private static final String TYPE_EVENTLOG = "eventlog";

    private static final String TYPE_INT = "int";
    private static final String TYPE_STRING = "str";
    private static final String TYPE_BYTES = "bytes";
    private static final String TYPE_BOOL = "bool";
    private static final String TYPE_ADDRESS = "Address";
    private static final String TYPE_LIST = "list";
    private static final String TYPE_DICT = "dict";

    private static final String VALUE_TRUE = "0x1";
    private static final String VALUE_FALSE = "0x0";

    @Test
    void testGetAPIForStepCounter() throws Exception {
        LOG.infoEntering("deployScore", "StepCounterScore");
        StepCounterScore score = StepCounterScore.mustDeploy(iconService, chain, chain.godWallet);
        LOG.infoExiting();

        LOG.infoEntering("testGetAPIForStepCounter");
        List<ScoreApi> apis = iconService.getScoreApi(score.getAddress()).execute();
        for (ScoreApi api : apis) {
            String name = api.getName();
            if (name.equals("getStep")) {
                assertEquals(api.getType(), TYPE_FUNCTION);
                assertEquals(api.getInputs().size(), 0);
                assertEquals(api.getReadonly(), VALUE_TRUE);

                List<ScoreApi.Param> outputs = api.getOutputs();
                assertEquals(outputs.size(), 1);

                ScoreApi.Param o1 = outputs.get(0);
                assertEquals(o1.getType(), TYPE_INT);
            } else if (name.equals("setStep") || name.equals("resetStep")) {
                assertEquals(api.getType(), TYPE_FUNCTION);
                assertNull(api.getReadonly());

                List<ScoreApi.Param> inputs = api.getInputs();
                assertEquals(inputs.size(), 1);

                ScoreApi.Param p1 = inputs.get(0);
                assertEquals(p1.getName(), "step");
                assertEquals(p1.getType(), "int");
            } else if (name.equals("increaseStep")) {
                assertEquals(TYPE_FUNCTION, api.getType());
                assertNull(api.getReadonly());

                List<ScoreApi.Param> inputs = api.getInputs();
                assertEquals(inputs.size(), 0);
            } else if (name.equals("ExternalProgress")) {
                assertEquals(api.getType(), "eventlog");
                assertNull(api.getReadonly());

                List<ScoreApi.Param> inputs = api.getInputs();
                assertEquals(inputs.size(), 2);

                ScoreApi.Param p1 = inputs.get(0);
                assertEquals(p1.getName(), "addr");
                assertEquals(p1.getType(), "Address");
                assertEquals(p1.getIndexed(), BigInteger.ONE);

                ScoreApi.Param p2 = inputs.get(1);
                assertEquals(p2.getName(), "step");
                assertEquals(p2.getType(), "int");
                assertEquals(p2.getIndexed(), BigInteger.ONE);
            } else if (name.equals("OnStep")) {
                assertEquals(api.getType(), "eventlog");
                assertNull(api.getReadonly());

                List<ScoreApi.Param> inputs = api.getInputs();
                assertEquals(inputs.size(), 1);

                ScoreApi.Param p1 = inputs.get(0);
                assertEquals(p1.getName(), "step");
                assertEquals(p1.getType(), "int");
                assertEquals(p1.getIndexed(), BigInteger.ONE);
            } else if (name.equals("trySetStepWith") || name.equals("setStepOf")) {
                assertEquals(api.getType(), TYPE_FUNCTION);
                assertNull(api.getReadonly());

                List<ScoreApi.Param> inputs = api.getInputs();
                assertEquals(inputs.size(), 2);

                ScoreApi.Param p1 = inputs.get(0);
                assertEquals(p1.getName(), "addr");
                assertEquals(p1.getType(), TYPE_ADDRESS);
                assertNull(p1.getIndexed());

                ScoreApi.Param p2 = inputs.get(1);
                assertEquals(p2.getName(), "step");
                assertEquals(p2.getType(), TYPE_INT);
                assertNull(p2.getIndexed());
            } else if (name.equals("increaseStepWith")) {
                assertEquals(api.getType(), TYPE_FUNCTION);
                assertNull(api.getReadonly());

                List<ScoreApi.Param> inputs = api.getInputs();
                assertEquals(inputs.size(), 2);

                ScoreApi.Param p1 = inputs.get(0);
                assertEquals(p1.getName(), "addr");
                assertEquals(p1.getType(), TYPE_ADDRESS);
                assertNull(p1.getIndexed());

                ScoreApi.Param p2 = inputs.get(1);
                assertEquals(p2.getName(), "count");
                assertEquals(p2.getType(), TYPE_INT);
                assertNull(p2.getIndexed());
            } else {
                throw new Exception("Unexpected method:"+api.toString());
            }
        }
        LOG.infoExiting();
    }

    static class FuncInfo {
        String type; // type of function
        Map<String, Input> inputsMap;
        String outputs;
        String readonly;
        String payable;

        static class Input {
            String name;
            String type; // type of data
            BigInteger indexed;
            Input(String name, String type, BigInteger indexed) {
                this.name = name;
                this.type = type;
                this.indexed = indexed;
            }
        }

        FuncInfo(String type, Input[] inputs, String outputs, String readonly, String payable) {
            this.type = type;
            this.outputs = outputs;
            this.readonly = readonly;
            this.payable = payable;
            inputsMap = new HashMap<>();
            if (inputs != null) {
                for(Input param : inputs) {
                    inputsMap.put(param.name, param);
                }
            }
        }
    }

    boolean checkApisForScoreApi(List<ScoreApi> apis) {
        LOG.infoEntering("checkApis");
        if (apis.size() == 0) {
            LOG.warning("Size of apis is 0");
            return false;
        }
        Map<String, FuncInfo> expectedFuncMap = new HashMap<String, FuncInfo>() {{
            put("externalMethod", new FuncInfo(TYPE_FUNCTION, null, null, VALUE_FALSE,  VALUE_FALSE));
            put("externalReadonlyMethod", new FuncInfo(TYPE_FUNCTION, null, null, VALUE_TRUE, VALUE_FALSE));
            put("payableExternalMethod", new FuncInfo(TYPE_FUNCTION, null, TYPE_STRING, VALUE_FALSE, VALUE_TRUE));
            put("externalPayableMethod", new FuncInfo(TYPE_FUNCTION, null, null, VALUE_FALSE, VALUE_TRUE));
            put("externalReadonlyFalseMethod", new FuncInfo(TYPE_FUNCTION, null, null, VALUE_FALSE, VALUE_FALSE));
            put("return_list", new FuncInfo(TYPE_FUNCTION, null, TYPE_LIST, VALUE_TRUE, VALUE_FALSE));
            put("return_dict", new FuncInfo(TYPE_FUNCTION, null, TYPE_DICT, VALUE_TRUE, VALUE_FALSE));
            put("fallback", new FuncInfo(TYPE_FALLBACK, null, null, VALUE_FALSE, VALUE_TRUE));
            put("param_int", new FuncInfo(TYPE_FUNCTION, new FuncInfo.Input[] {
                    new FuncInfo.Input("param1", TYPE_INT, null)
            }, "int", VALUE_TRUE, VALUE_FALSE));
            put("param_str", new FuncInfo(TYPE_FUNCTION, new FuncInfo.Input[] {
                    new FuncInfo.Input("param1", TYPE_STRING, null)
            }, "str", VALUE_TRUE, VALUE_FALSE));
            put("param_bytes", new FuncInfo(TYPE_FUNCTION, new FuncInfo.Input[] {
                    new FuncInfo.Input("param1", TYPE_BYTES, null)
            }, "bytes", VALUE_TRUE, VALUE_FALSE));
            put("param_bool", new FuncInfo(TYPE_FUNCTION, new FuncInfo.Input[] {
                    new FuncInfo.Input("param1", TYPE_BOOL, null)
            }, "bool", VALUE_TRUE, VALUE_FALSE));
            put("param_Address", new FuncInfo(TYPE_FUNCTION, new FuncInfo.Input[] {
                    new FuncInfo.Input("param1", TYPE_ADDRESS, null)
            }, "Address", VALUE_TRUE, VALUE_FALSE));
            put("eventlog_index1", new FuncInfo(TYPE_EVENTLOG, new FuncInfo.Input[] {
                    new FuncInfo.Input("param1", TYPE_INT, BigInteger.ONE),
                    new FuncInfo.Input("param2", TYPE_STRING, null)
            }, null, VALUE_FALSE, VALUE_FALSE));
        }};

        for (ScoreApi api : apis) {
            String funcName = api.getName();
            FuncInfo fInfo = expectedFuncMap.get(funcName);
            if (fInfo == null) {
                LOG.warning(funcName + " not exists function");
                return false;
            }
            if (fInfo.type.compareTo(api.getType()) != 0) {
                LOG.warning("[" + funcName + "] is " + api.getType() + " but " + fInfo.type);
                return false;
            }
            if (fInfo.readonly.equals(VALUE_TRUE)) {
                if(fInfo.readonly.compareTo(api.getReadonly()) != 0) {
                    LOG.warning("[" + funcName + "] is readonly but " + api.getReadonly());
                    return false;
                }
            }
            if (fInfo.payable.equals(VALUE_TRUE)) {
                if (fInfo.payable.compareTo(api.getProperties().getItem("payable").asString()) != 0) {
                    LOG.warning("[" + funcName + "] is payable but " + api.getProperties().getItem("payable").asString());
                    return false;
                }
            }
            for (ScoreApi.Param sParam : api.getInputs()) {
                String pName = sParam.getName();
                FuncInfo.Input fParam = fInfo.inputsMap.get(pName);
                if (fParam == null) {
                    LOG.warning("[" + funcName + "][" + pName + "] does not exist");
                    return false;
                }
                if (fParam.type.compareTo(sParam.getType()) != 0) {
                    LOG.warning("[" + funcName + "][" + pName + "] type is " + fParam.type + " but " + sParam.getType());
                    return false;
                }
                if (fParam.indexed != null) {
                    if (fParam.indexed.compareTo(sParam.getIndexed()) != 0) {
                        LOG.warning("[" + funcName + "][" + pName + "] type is indexed [" + fParam.indexed + " but " + sParam.getIndexed());
                        return false;
                    }
                }
                fInfo.inputsMap.remove(sParam.getName());
            }
            if (fInfo.inputsMap.size() != 0) {
                LOG.warning("Not received param [" + fInfo.inputsMap.keySet() + "]");
                return false;
            }
            expectedFuncMap.remove(funcName);
        }
        if (expectedFuncMap.size() != 0) {
            LOG.warning("NOT received [" + expectedFuncMap.keySet() + "]");
            return false;
        }
        LOG.infoExiting();
        return true;
    }

    @Test
    void validateGetScoreApi() throws Exception {
        String scorePath = Constants.SCORE_API_PATH;
        LOG.infoEntering("deployScore", "ScoreApi");
        Bytes txHash = Utils.deployScore(iconService, chain.networkId,
                KeyWallet.create(), Constants.CHAINSCORE_ADDRESS, scorePath, null);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());
        Utils.acceptIfAuditEnabled(iconService, chain, txHash);
        LOG.infoExiting();

        LOG.infoEntering("validateGetScoreApi");
        Address scoreAddr = new Address(result.getScoreAddress());
        List<ScoreApi> apis = iconService.getScoreApi(scoreAddr).execute();
        assertTrue(checkApisForScoreApi(apis));
        LOG.infoExiting();

        LOG.infoEntering("notExistsScoreAddress");
        String newAddr = KeyWallet.create().getAddress().toString();
        Address noScoreAddr = new Address("cx" + newAddr.substring(2));
        try {
            iconService.getScoreApi(noScoreAddr).execute();
            fail();
        }
        catch (RpcError ex) {
            LOG.info("Expected exception: " + ex);
        }
        LOG.infoExiting();

        LOG.infoEntering("getApiWithEOA");
        try {
            // we use custom rpc requester to get the server response directly
            getScoreApi(KeyWallet.create().getAddress().toString());
            fail();
        }
        catch (RpcError ex) {
            LOG.info("Expected exception: " + ex);
        }
        LOG.infoExiting();
    }

    private List<ScoreApi> getScoreApi(String addr) throws IOException {
        long requestId = System.currentTimeMillis();
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(addr))
                .build();
        foundation.icon.icx.transport.jsonrpc.Request request = new foundation.icon.icx.transport.jsonrpc.Request(requestId, "icx_getScoreApi", params);
        return new HttpProvider(Env.nodes[0].channels[0].getAPIUrl(Env.testApiVer)).request(request, Converters.SCORE_API_LIST).execute();
    }
}
