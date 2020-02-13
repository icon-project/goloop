package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import static org.junit.jupiter.api.Assertions.assertEquals;

@Tag(Constants.TAG_PY_SCORE)
public class GetTotalSupplyTest {
    private static Env.Chain chain;
    private static IconService iconService;

    @BeforeAll
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
    }

    @Test
    public void testGetTotalSupply() throws Exception {
        BigInteger supply = iconService.getTotalSupply().execute();
        String exp = chain.getProperty("totalSupply");
        System.out.println("Total supply = "+supply);
        if ( exp != null ) {
            RpcValue ro = new RpcValue(supply);
            assertEquals(ro.toString(), exp);
        } else {
            System.err.println("We can't sure that it's correct 'totalSupply' should be set in env.properties");
        }
    }
}
