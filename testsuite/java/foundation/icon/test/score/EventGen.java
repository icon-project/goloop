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

public class EventGen extends Score {
    private static final String INSTALL_PATH = Constants.SCORE_ROOT +  "event_gen";

    public EventGen(IconService iconService, Env.Chain chain, Address scoreAddress) {
        super(iconService, chain, scoreAddress);
    }

    // install with default stepLimit and default parameter
    public static EventGen install(IconService service, Env.Chain chain, Wallet wallet)
            throws TransactionFailureException, ResultTimeoutException, IOException
    {
        return install(service, chain, wallet, Constants.DEFAULT_STEP_LIMIT);
    }

    // install with default parameter
    public static EventGen install(IconService service, Env.Chain chain, Wallet wallet, long stepLimit)
            throws TransactionFailureException, ResultTimeoutException, IOException
    {
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("EventGen"))
                .build();
        return install(service, chain, wallet, params, stepLimit);
    }

    // install with passed parameter
    public static EventGen install(IconService service, Env.Chain chain,
                                     Wallet wallet, RpcObject params, long stepLimit)
            throws TransactionFailureException, ResultTimeoutException, IOException
    {
        return new EventGen(
                service,
                chain,
                Score.install(service, chain, wallet, INSTALL_PATH, params, stepLimit)
        );
    }

    public TransactionResult invokeGenerate(Wallet from, Address addr, BigInteger i, byte[] bytes) throws ResultTimeoutException, IOException{
        RpcObject params = new RpcObject.Builder()
                .put("_addr", new RpcValue(addr))
                .put("_int", new RpcValue(i))
                .put("_bytes", new RpcValue(bytes))
                .build();
        return invokeAndWaitResult(from, "generate", params,
                BigInteger.valueOf(0), BigInteger.valueOf(100));
    }
}
