package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.data.ScoreApi;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Env;
import foundation.icon.test.score.StepCounterScore;
import org.junit.BeforeClass;
import org.junit.Test;

import java.math.BigInteger;
import java.util.List;

import static org.hamcrest.CoreMatchers.*;
import static org.junit.Assert.*;

public class GetAPITest {
    static Env.Chain chain;
    static IconService iconService;
    static StepCounterScore score;

    @BeforeClass
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        score = StepCounterScore.mustDeploy(iconService, chain, chain.godWallet);
    }

    static final String TYPE_FUNCTION = "function";
    static final String TYPE_EVENT = "event";
    static final String TYPE_INT = "int";
    static final String TYPE_STRING = "str";
    static final String TYPE_ADDRESS = "Address";

    static final String VALUE_TRUE = "0x1";
    static final String VALUE_FALSE = "0x0";

    @Test
    public void testGetAPIForStepCounter() throws Exception {
        List<ScoreApi> apis = iconService.getScoreApi(score.getAddress()).execute();
        for ( ScoreApi api : apis ) {
            String name = api.getName().intern();
            if ( name == "getStep" ) {
                assertThat(api.getType(), is(TYPE_FUNCTION));
                assertThat(api.getInputs().size(), is(0));
                assertThat(api.getReadonly(), is(VALUE_TRUE));

                List<ScoreApi.Param> outputs = api.getOutputs();
                assertThat(outputs.size(), is(1));

                ScoreApi.Param o1 = outputs.get(0);
                assertThat(o1.getType(), is(TYPE_INT));
            } else if ( name == "setStep" || name == "resetStep" ) {
                assertThat(api.getType(), is(TYPE_FUNCTION));
                assertThat(api.getReadonly(), anyOf(is(VALUE_FALSE), nullValue()));

                List<ScoreApi.Param> inputs = api.getInputs();
                assertThat(inputs.size(), is(1));

                ScoreApi.Param p1 = inputs.get(0);
                assertThat(p1.getName(), is("step"));
                assertThat(p1.getType(), is("int"));
            } else if ( name == "increaseStep" ) {
                assertEquals(TYPE_FUNCTION, api.getType());
                assertThat(api.getReadonly(), anyOf(is(VALUE_FALSE), nullValue()));

                List<ScoreApi.Param> inputs = api.getInputs();
                assertThat(inputs.size(), is(0));
            } else if ( name == "ExternalProgress" ) {
                assertThat(api.getType(), is("event"));
                assertThat(api.getReadonly(), anyOf(is(VALUE_FALSE), nullValue()));

                List<ScoreApi.Param> inputs = api.getInputs();
                assertThat(inputs.size(), is(2));

                ScoreApi.Param p1 = inputs.get(0);
                assertThat(p1.getName(), is("addr"));
                assertThat(p1.getType(), is("Address"));
                assertThat(p1.getIndexed(), is(BigInteger.ONE));

                ScoreApi.Param p2 = inputs.get(1);
                assertThat(p2.getName(), is("step"));
                assertThat(p2.getType(), is("int"));
                assertThat(p2.getIndexed(), is(BigInteger.ONE));
            } else if ( name == "OnStep" ) {
                assertThat(api.getType(), is("event"));
                assertThat(api.getReadonly(), anyOf(is(VALUE_FALSE), nullValue()));

                List<ScoreApi.Param> inputs = api.getInputs();
                assertThat(inputs.size(), is(1));

                ScoreApi.Param p1 = inputs.get(0);
                assertThat(p1.getName(), is("step"));
                assertThat(p1.getType(), is("int"));
                assertThat(p1.getIndexed(), equalTo(BigInteger.ONE));
            } else if ( name == "trySetStepWith" || name == "setStepOf" ) {
                assertThat(api.getType(), is(TYPE_FUNCTION));
                assertThat(api.getReadonly(), anyOf(is(VALUE_FALSE), nullValue()));

                List<ScoreApi.Param> inputs = api.getInputs();
                assertThat(inputs.size(), is(2));

                ScoreApi.Param p1 = inputs.get(0);
                assertThat(p1.getName(), is("addr"));
                assertThat(p1.getType(), is(TYPE_ADDRESS));
                assertThat(p1.getIndexed(), anyOf(equalTo(BigInteger.ZERO), nullValue()));

                ScoreApi.Param p2 = inputs.get(1);
                assertThat(p2.getName(), is("step"));
                assertThat(p2.getType(), is(TYPE_INT));
                assertThat(p2.getIndexed(), anyOf(equalTo(BigInteger.ZERO), nullValue()));
            } else if ( name == "increaseStepWith") {
                assertThat(api.getType(), is(TYPE_FUNCTION));
                assertThat(api.getReadonly(), anyOf(is(VALUE_FALSE), nullValue()));

                List<ScoreApi.Param> inputs = api.getInputs();
                assertThat(inputs.size(), is(2));

                ScoreApi.Param p1 = inputs.get(0);
                assertThat(p1.getName(), is("addr"));
                assertThat(p1.getType(), is(TYPE_ADDRESS));
                assertThat(p1.getIndexed(), anyOf(equalTo(BigInteger.ZERO), nullValue()));

                ScoreApi.Param p2 = inputs.get(1);
                assertThat(p2.getName(), is("count"));
                assertThat(p2.getType(), is(TYPE_INT));
                assertThat(p2.getIndexed(), anyOf(equalTo(BigInteger.ZERO), nullValue()));
            } else {
                throw new Exception("Unexpected method:"+api.toString());
            }
        }
    }
}
