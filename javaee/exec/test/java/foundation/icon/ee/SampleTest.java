package foundation.icon.ee;

import collection.CollectionTest;
import example.InheritedToken;
import example.SampleToken;
import example.token.IRC2;
import example.token.IRC2Basic;
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
        var app = sm.deploy(new Class<?>[]{InheritedToken.class, IRC2Basic.class, IRC2.class}, "MySampleToken", "MST", 18, 1000);
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
}
