package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.data.IconAmount;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Env;
import org.hamcrest.CoreMatchers;
import org.junit.BeforeClass;
import org.junit.Test;

import java.math.BigInteger;

import static org.junit.Assert.assertThat;

public class GetTotalSupplyTest {
    static Env.Chain chain;
    static IconService iconService;

    @BeforeClass
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
            assertThat(ro.toString(), CoreMatchers.is(exp));
        } else {
            System.err.println("We can't sure that it's correct 'totalSupply' should be set in env.properties");
        }
    }
}
