package foundation.icon.test.cases;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import org.junit.BeforeClass;
import org.junit.Test;

import java.io.IOException;
import java.math.BigInteger;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;

/*
test cases
1. audit
2. not enough balance for deploy.
 - setStepPrice
3. not enough stepLimit for deploy.
4. content
 - no root file.
 - not zip
 - too large
 - takes too long time for uncompress
5. sendTransaction with invalid/valid params
6. sendTransaction for update with invalid score address
7. change destination url.
8. sendTransaction with invalid signature
 */
public class Deploy {
    private String contentPath;
    private KeyWallet governor;
    private KeyWallet owner;
    private boolean audit;
    private final IconService iconService;
    private final Env.Chain chain;

    public static final Address ZERO_ADDRESS = new Address("cx0000000000000000000000000000000000000000");
    public Deploy() {
        Env.Node node = Env.nodes[0];
        chain = Env.nodes[0].chains[0];
        iconService = new IconService(new HttpProvider(node.endpointUrl));
    }

    @BeforeClass
    public static void initDeploy() {

    }

    public void notEnoughBalance() {
    }

    public void notEnoughStepLimit() {
    }

    public void invalidContentNoRootFile() {
    }

    public void invalidContentNotZip() {
    }

    public void invalidContentTooBig() {
    }

    public void installWithInvalidParams() {
    }

    public void updateWithInvalidParams() {
    }

    public void updateWithInvalidScoreAddress() {
    }
}
