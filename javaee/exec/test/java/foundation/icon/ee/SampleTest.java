package foundation.icon.ee;

import collection.CollectionTest;
import example.IRC2BasicToken;
import example.IRC3BasicToken;
import example.SampleToken;
import example.token.IRC2;
import example.token.IRC2Basic;
import example.token.IRC3;
import example.token.IRC3Basic;
import example.util.Arrays;
import example.util.EnumerableIntMap;
import foundation.icon.ee.test.GoldenTest;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.nio.charset.StandardCharsets;

public class SampleTest extends GoldenTest {
    @Test
    public void testSample() {
        var owner = sm.getOrigin();
        var app = sm.deploy(SampleToken.class, "MySampleToken", "MST", 18, 1000);
        app.invoke("balanceOf", owner);
        var addr1 = sm.newExternalAddress();
        app.invoke("transfer", addr1, new BigInteger("1000000000000000000"), "Hello".getBytes(StandardCharsets.UTF_8));
        var addr2 = sm.newExternalAddress();
        app.invoke("transfer", addr2, new BigInteger("1000000000000000000"), "Hello".getBytes(StandardCharsets.UTF_8));
        app.invoke("balanceOf", addr1);
        app.invoke("balanceOf", owner);
        app.invoke("totalSupply");
        var app2 = sm.deploy(CollectionTest.class);
        app2.invoke("getInt");
        app2.invoke("totalSupply2", app.getAddress());
        app2.invoke("balanceOf2", app.getAddress(), owner);
    }

    @Test
    public void testInherited() {
        var owner = sm.getOrigin();
        var app = sm.deploy(new Class<?>[]{IRC2BasicToken.class, IRC2Basic.class, IRC2.class}, "MySampleToken", "MST", 18, 1000);
        app.invoke("balanceOf", owner);
        var addr1 = sm.newExternalAddress();
        app.invoke("transfer", addr1, new BigInteger("1000000000000000000"), "Hello".getBytes(StandardCharsets.UTF_8));
        var addr2 = sm.newExternalAddress();
        app.invoke("transfer", addr2, new BigInteger("1000000000000000000"), "Hello".getBytes(StandardCharsets.UTF_8));
        app.invoke("balanceOf", addr1);
        app.invoke("balanceOf", owner);
        app.invoke("totalSupply");
        var app2 = sm.deploy(CollectionTest.class);
        app2.invoke("getInt");
        app2.invoke("totalSupply2", app.getAddress());
        app2.invoke("balanceOf2", app.getAddress(), owner);
    }

    @Test
    public void testIRC3() {
        var owner = sm.getOrigin();
        var app = sm.deploy(new Class<?>[]{IRC3BasicToken.class, IRC3Basic.class, IRC3.class,
                EnumerableIntMap.class, Arrays.class}, "MyNFT", "NFT");
        app.invoke("balanceOf", owner);
        app.invoke("totalSupply");
        var tokenId1 = BigInteger.valueOf(100);
        var tokenId2 = BigInteger.valueOf(200);
        var tokenId3 = BigInteger.valueOf(300);
        app.invoke("mint", tokenId1);
        app.invoke("mint", tokenId2);
        app.invoke("mint", tokenId3);
        app.invoke("balanceOf", owner);
        var addr1 = sm.newExternalAddress();
        app.invoke("transfer", addr1, tokenId1);
        var addr2 = sm.newExternalAddress();
        app.invoke("transfer", addr2, tokenId3);
        app.invoke("balanceOf", owner);
        app.invoke("tokenOfOwnerByIndex", owner, 0);
        app.invoke("balanceOf", addr1);
        app.invoke("tokenOfOwnerByIndex", addr1, 0);
        app.invoke("balanceOf", addr2);
        app.invoke("tokenOfOwnerByIndex", addr2, 0);
        app.invoke("totalSupply");
    }
}
