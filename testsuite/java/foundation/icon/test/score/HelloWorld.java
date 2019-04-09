package foundation.icon.test.score;

import foundation.icon.icx.IconService;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;

import java.io.IOException;
import java.math.BigInteger;

public class HelloWorld extends Score {
    private static final String PATH = Constants.SCORE_ROOT +  "helloWorld.zip";
    private static final String PATH2 = Constants.SCORE_ROOT +  "helloWorld2.zip";

    public static HelloWorld mustDeploy(IconService service, Wallet wallet, RpcObject params, BigInteger nid, long stepLimit)
            throws TransactionFailureException, ResultTimeoutException, IOException
    {
        return new HelloWorld(
                service,
                Score.mustDeploy(service, wallet, PATH, params, stepLimit),
                nid
        );
    }

    public static HelloWorld mustDeploy(IconService service, Wallet wallet, BigInteger nid, long stepLimit)
            throws TransactionFailureException, ResultTimeoutException, IOException
    {
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        return mustDeploy(service, wallet, params, nid, stepLimit);
    }

    //
    public static HelloWorld mustDeploy(IconService service, Wallet wallet, BigInteger nid)
            throws TransactionFailureException, ResultTimeoutException, IOException
    {
        return mustDeploy(service, wallet, nid, -1);
    }

    public void update(IconService service, Wallet wallet, RpcObject params, BigInteger nid)
            throws TransactionFailureException, ResultTimeoutException, IOException
    {
        super.update(service, wallet, PATH2, params, nid);
    }

    public HelloWorld(IconService iconService, Address scoreAddress, BigInteger nid) {
        super(iconService, scoreAddress, nid);
    }

    public boolean invokeHello(Wallet from) {
        try {
            TransactionResult result = invokeAndWaitResult(from, "hello", null
                    , BigInteger.valueOf(0), BigInteger.valueOf(100));
            if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
                return false;
            }
        }
        catch ( ResultTimeoutException | IOException ex) {
            return false;
        }
        return true;
    }
}
