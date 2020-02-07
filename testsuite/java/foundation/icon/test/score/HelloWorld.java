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
import foundation.icon.test.common.TransactionHandler;

import java.io.IOException;

public class HelloWorld extends Score {
    public static final String INSTALL_PATH = Constants.SCORE_HELLOWORLD_PATH;
    public static final String UPDATE_PATH = Constants.SCORE_HELLOWORLD_UPDATE_PATH;

    public HelloWorld(TransactionHandler txHandler, Address address) {
        super(txHandler, address);
    }

    public HelloWorld(Score other) {
        super(other);
    }

    public static HelloWorld install(IconService service, Env.Chain chain, Wallet wallet)
            throws TransactionFailureException, IOException, ResultTimeoutException {
        TransactionHandler txHandler = new TransactionHandler(service, chain);
        return install(txHandler, wallet);
    }

    public static HelloWorld install(TransactionHandler txHandler, Wallet wallet)
            throws TransactionFailureException, ResultTimeoutException, IOException {
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("HelloWorld"))
                .build();
        return install(txHandler, wallet, params);
    }

    public static HelloWorld install(TransactionHandler txHandler, Wallet wallet, RpcObject params)
            throws TransactionFailureException, ResultTimeoutException, IOException {
        return new HelloWorld(txHandler.deploy(wallet, INSTALL_PATH, params));
    }

    public TransactionResult invokeHello(Wallet from) throws ResultTimeoutException, IOException {
        return invokeAndWaitResult(from, "hello", null);
    }
}
