package foundation.icon.test.score;

import foundation.icon.icx.IconService;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;

import java.io.IOException;
import java.math.BigInteger;

public class HelloWorld extends Score {
    private static final String INSTALL_PATH = Constants.SCORE_ROOT +  "helloWorld.zip";
    private static final String UPDATE_PATH = Constants.SCORE_ROOT +  "helloWorld2.zip";

    public HelloWorld(IconService iconService, Env.Chain chain, Address scoreAddress) {
        super(iconService, chain, scoreAddress);
    }

    // install with default stepLimit and default parameter
    public static HelloWorld install(IconService service, Env.Chain chain, Wallet wallet)
            throws TransactionFailureException, ResultTimeoutException, IOException
    {
        return install(service, chain, wallet, -1);
    }

    // install with default parameter
    public static HelloWorld install(IconService service, Env.Chain chain, Wallet wallet, long stepLimit)
            throws TransactionFailureException, ResultTimeoutException, IOException
    {
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        return install(service, chain, wallet, params, stepLimit);
    }

    // install with passed parameter
    public static HelloWorld install(IconService service, Env.Chain chain,
                                     Wallet wallet, RpcObject params, long stepLimit)
            throws TransactionFailureException, ResultTimeoutException, IOException
    {
        return new HelloWorld(
                service,
                chain,
                Score.install(service, chain, wallet, INSTALL_PATH, params, stepLimit)
        );
    }

    public void update(IconService service, Env.Chain chain, Wallet wallet, RpcObject params)
            throws TransactionFailureException, ResultTimeoutException, IOException
    {
        super.update(service, chain, wallet, UPDATE_PATH, params);
    }

    public TransactionResult invokeHello(Wallet from) throws ResultTimeoutException, IOException{
        return invokeAndWaitResult(from, "hello", null
                , BigInteger.valueOf(0), BigInteger.valueOf(100));
    }
}
