package foundation.icon.test.score;

import foundation.icon.icx.IconService;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;

import java.io.IOException;
import java.math.BigInteger;

public class StepCounterScore extends Score {
    private static final String PATH = Constants.SCORE_ROOT + "step_counter.zip";
    protected final static BigInteger STEPS = BigInteger.valueOf(3).multiply(BigInteger.TEN.pow(6));

    public static StepCounterScore mustDeploy(IconService service, Env.Chain chain, Wallet wallet)
            throws IOException, TransactionFailureException, ResultTimeoutException
    {
        return new StepCounterScore(
                service,
                chain,
                Score.install(service, chain, wallet, PATH, null)
        );
    }

    public StepCounterScore(IconService service, Env.Chain chain, Address target) {
        super(service, chain, target);
    }

    public TransactionResult increaseStep(Wallet wallet) throws ResultTimeoutException, IOException {
        return this.invokeAndWaitResult(wallet,
                "increaseStep", null, null, STEPS);
    }

    public TransactionResult setStep(Wallet wallet, BigInteger step) throws ResultTimeoutException, IOException {
        return this.invokeAndWaitResult(wallet,
                "setStep",
                (new RpcObject.Builder())
                    .put("step", new RpcValue(step))
                    .build(),
                null, STEPS);
    }

    public TransactionResult resetStep(Wallet wallet, BigInteger step) throws ResultTimeoutException, IOException {
        return this.invokeAndWaitResult(wallet,
                "resetStep",
                (new RpcObject.Builder())
                        .put("step", new RpcValue(step))
                        .build(),
                null, STEPS);
    }

    public TransactionResult setStepOf(Wallet wallet, Address target, BigInteger step) throws ResultTimeoutException, IOException {
        return this.invokeAndWaitResult(wallet,
                "setStepOf",
                (new RpcObject.Builder())
                    .put("step", new RpcValue(step))
                    .put( "addr", new RpcValue(target))
                    .build(),
                null, STEPS);
    }

    public TransactionResult trySetStepWith(Wallet wallet, Address target, BigInteger step) throws ResultTimeoutException, IOException{
        return this.invokeAndWaitResult(wallet,
                "trySetStepWith",
                (new RpcObject.Builder())
                        .put("step", new RpcValue(step))
                        .put( "addr", new RpcValue(target))
                        .build(),
                null, STEPS);
    }

    public BigInteger getStep(Address from) throws IOException {
        RpcItem res = this.call(from, "getStep", null);
        return res.asInteger();
    }
}
