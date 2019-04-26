package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.data.ScoreApi;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Env;
import foundation.icon.test.score.StepCounterScore;
import org.hamcrest.CoreMatchers;
import org.junit.BeforeClass;
import org.junit.Test;

import java.math.BigInteger;
import java.util.List;

import static org.junit.Assert.assertThat;

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

    @Test
    public void testGetAPIForStepCounter() throws Exception {
        List<ScoreApi> apis = iconService.getScoreApi(score.getAddress()).execute();
        for ( ScoreApi api : apis ) {
            String name = api.getName().intern();
            if ( name == "getStep" ) {
                assertThat(api.getType(), CoreMatchers.is("function"));
                assertThat(api.getInputs().size(), CoreMatchers.is(0));
                assertThat(api.getReadonly(), CoreMatchers.is("0x1"));

                List<ScoreApi.Param> outputs = api.getOutputs();
                assertThat(outputs.size(), CoreMatchers.is(1));

                ScoreApi.Param o1 = outputs.get(0);
                assertThat(o1.getType(), CoreMatchers.is("int"));
            } else if ( name == "setStep" || name == "resetStep" ) {
                assertThat(api.getType(), CoreMatchers.is("function"));
                assertThat(api.getReadonly(), CoreMatchers.not("0x1"));

                List<ScoreApi.Param> inputs = api.getInputs();
                assertThat(inputs.size(), CoreMatchers.is(1));

                ScoreApi.Param p1 = inputs.get(0);
                assertThat(p1.getName(), CoreMatchers.is("step"));
                assertThat(p1.getType(), CoreMatchers.is("int"));
            } else if ( name == "increaseStep" ) {
                assertThat(api.getType(), CoreMatchers.is("function"));
                assertThat(api.getReadonly(), CoreMatchers.not("0x1"));

                List<ScoreApi.Param> inputs = api.getInputs();
                assertThat(inputs.size(), CoreMatchers.is(0));
            } else if ( name == "ExternalProgress" ) {
                assertThat(api.getType(), CoreMatchers.is("event"));
                assertThat(api.getReadonly(), CoreMatchers.not("0x1"));

                List<ScoreApi.Param> inputs = api.getInputs();
                assertThat(inputs.size(), CoreMatchers.is(2));

                ScoreApi.Param p1 = inputs.get(0);
                assertThat(p1.getName(), CoreMatchers.is("addr"));
                assertThat(p1.getType(), CoreMatchers.is("Address"));
                assertThat(p1.getIndexed(), CoreMatchers.is(BigInteger.ONE));

                ScoreApi.Param p2 = inputs.get(1);
                assertThat(p2.getName(), CoreMatchers.is("step"));
                assertThat(p2.getType(), CoreMatchers.is("int"));
                assertThat(p2.getIndexed(), CoreMatchers.is(BigInteger.ONE));
            } else if ( name == "OnStep" ) {
                assertThat(api.getType(), CoreMatchers.is("event"));
                assertThat(api.getReadonly(), CoreMatchers.not("0x1"));

                List<ScoreApi.Param> inputs = api.getInputs();
                assertThat(inputs.size(), CoreMatchers.is(1));

                ScoreApi.Param p1 = inputs.get(0);
                assertThat(p1.getName(), CoreMatchers.is("step"));
                assertThat(p1.getType(), CoreMatchers.is("int"));
                assertThat(p1.getIndexed(), CoreMatchers.is(CoreMatchers.equalTo(BigInteger.ONE)));
            } else if ( name == "trySetStepWith" || name == "setStepOf" ) {
                assertThat(api.getType(), CoreMatchers.is("function"));
                assertThat(api.getReadonly(), CoreMatchers.not("0x1"));

                List<ScoreApi.Param> inputs = api.getInputs();
                assertThat(inputs.size(), CoreMatchers.is(2));

                ScoreApi.Param p1 = inputs.get(0);
                assertThat(p1.getName(), CoreMatchers.is("addr"));
                assertThat(p1.getType(), CoreMatchers.is("Address"));
                assertThat(p1.getIndexed(), CoreMatchers.not(BigInteger.ONE));

                ScoreApi.Param p2 = inputs.get(1);
                assertThat(p2.getName(), CoreMatchers.is("step"));
                assertThat(p2.getType(), CoreMatchers.is("int"));
                assertThat(p2.getIndexed(), CoreMatchers.not(BigInteger.ONE));
            } else if ( name == "increaseStepWith") {
                assertThat(api.getType(), CoreMatchers.is("function"));
                assertThat(api.getReadonly(), CoreMatchers.not("0x1"));

                List<ScoreApi.Param> inputs = api.getInputs();
                assertThat(inputs.size(), CoreMatchers.is(2));

                ScoreApi.Param p1 = inputs.get(0);
                assertThat(p1.getName(), CoreMatchers.is("addr"));
                assertThat(p1.getType(), CoreMatchers.is("Address"));
                assertThat(p1.getIndexed(), CoreMatchers.not(BigInteger.ONE));

                ScoreApi.Param p2 = inputs.get(1);
                assertThat(p2.getName(), CoreMatchers.is("count"));
                assertThat(p2.getType(), CoreMatchers.is("int"));
                assertThat(p2.getIndexed(), CoreMatchers.not(BigInteger.ONE));
            } else {
                throw new Exception("Unexpected method:"+api.toString());
            }
        }
    }
}
