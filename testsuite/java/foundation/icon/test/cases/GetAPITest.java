package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.ScoreApi;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.StepCounterScore;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.*;


@Tag(Constants.TAG_PARALLEL)
public class GetAPITest {
    static Env.Chain chain;
    static IconService iconService;
    static StepCounterScore score;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        score = StepCounterScore.mustDeploy(iconService, chain, chain.godWallet);
    }

    static final String TYPE_FUNCTION = "function";
    static final String TYPE_FALLBACK = "fallback";
    static final String TYPE_EVENTLOG = "eventlog";

    static final String TYPE_INT = "int";
    static final String TYPE_STRING = "str";
    static final String TYPE_BYTES = "bytes";
    static final String TYPE_BOOL = "bool";
    static final String TYPE_ADDRESS = "Address";
    static final String TYPE_LIST = "list";
    static final String TYPE_DICT = "dict";

    static final String VALUE_TRUE = "0x1";
    static final String VALUE_FALSE = "0x0";

    @Test
    public void testGetAPIForStepCounter() throws Exception {
        List<ScoreApi> apis = iconService.getScoreApi(score.getAddress()).execute();
        for ( ScoreApi api : apis ) {
            String name = api.getName().intern();
            if ( name == "getStep" ) {
                assertEquals(api.getType(), TYPE_FUNCTION);
                assertEquals(api.getInputs().size(), 0);
                assertEquals(api.getReadonly(), VALUE_TRUE);

                List<ScoreApi.Param> outputs = api.getOutputs();
                assertEquals(outputs.size(), 1);

                ScoreApi.Param o1 = outputs.get(0);
                assertEquals(o1.getType(), TYPE_INT);
            } else if ( name == "setStep" || name == "resetStep" ) {
                assertEquals(api.getType(), TYPE_FUNCTION);
                assertNull(api.getReadonly());

                List<ScoreApi.Param> inputs = api.getInputs();
                assertEquals(inputs.size(), 1);

                ScoreApi.Param p1 = inputs.get(0);
                assertEquals(p1.getName(), "step");
                assertEquals(p1.getType(), "int");
            } else if ( name == "increaseStep" ) {
                assertEquals(TYPE_FUNCTION, api.getType());
                assertNull(api.getReadonly());

                List<ScoreApi.Param> inputs = api.getInputs();
                assertEquals(inputs.size(), 0);
            } else if ( name == "ExternalProgress" ) {
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
            } else if ( name == "OnStep" ) {
                assertEquals(api.getType(), "eventlog");
                assertNull(api.getReadonly());

                List<ScoreApi.Param> inputs = api.getInputs();
                assertEquals(inputs.size(), 1);

                ScoreApi.Param p1 = inputs.get(0);
                assertEquals(p1.getName(), "step");
                assertEquals(p1.getType(), "int");
                assertEquals(p1.getIndexed(), BigInteger.ONE);
            } else if ( name == "trySetStepWith" || name == "setStepOf" ) {
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
            } else if ( name == "increaseStepWith") {
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
            if(inputs != null) {
                for(Input param : inputs) {
                    inputsMap.put(param.name, param);
                }
            }
        }
    }

    @Test
    public void checkScoreApi() throws Exception {
        // expected type(function, eventlog, fallback), name, inputs(name, type, indexed), outputs(type), readonly, payable
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

        LOG.infoEntering("checkScoreApi");
        String scorePath = Constants.SCORE_API_PATH;
        Bytes txHash = Utils.deployScore(iconService, chain.networkId,
                KeyWallet.create(), Constants.CHAINSCORE_ADDRESS, scorePath, null);
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        assertEquals(Constants.STATUS_SUCCESS, result.getStatus());

        Utils.acceptIfAuditEnabled(iconService, chain, txHash);
        Address scoreAddr = new Address(result.getScoreAddress());
        List<ScoreApi> apis = iconService.getScoreApi(scoreAddr).execute();
        for ( ScoreApi api : apis ) {
            String funcName = api.getName();
            FuncInfo fInfo = expectedFuncMap.get(funcName);
            assertNotNull(fInfo);
            assertEquals(fInfo.type, api.getType());
            if(fInfo.readonly.equals(VALUE_TRUE)) {
                assertEquals(fInfo.readonly, api.getReadonly());
            }
            if(fInfo.payable.equals(VALUE_TRUE)) {
                assertEquals(fInfo.payable, api.getProperties().getItem("payable").asString());
            }
            for(ScoreApi.Param sParam : api.getInputs()) {
                String pName = sParam.getName();
                FuncInfo.Input fParam = fInfo.inputsMap.get(pName);
                assertNotNull(fParam);
                assertEquals(fParam.type, sParam.getType());
                if(fParam.indexed != null) {
                    assertEquals(fParam.indexed, sParam.getIndexed());
                }
                fInfo.inputsMap.remove(sParam.getName());
            }
            if(fInfo.inputsMap.size() != 0) {
                LOG.warning("Not received param [" + fInfo.inputsMap.keySet() + "]");
                fail();
            }
            expectedFuncMap.remove(funcName);
        }
        if(expectedFuncMap.size() != 0) {
            LOG.warning("NOT received [" + expectedFuncMap.keySet() + "]");
            fail();
        }
        LOG.infoExiting();
    }
}
