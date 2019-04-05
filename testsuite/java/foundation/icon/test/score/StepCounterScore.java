package foundation.icon.test.score;

import foundation.icon.icx.IconService;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;

import java.io.IOException;
import java.math.BigInteger;
import java.util.concurrent.TimeoutException;

public class StepCounterScore extends Score {
    private static final String PATH = Constants.SCORE_ROOT + "step_counter.zip";
    protected final static BigInteger STEPS = BigInteger.valueOf(3).multiply(BigInteger.TEN.pow(6));

    public static StepCounterScore mustDeploy(IconService service, Wallet wallet, BigInteger nid)
            throws IOException, TransactionFailureException, TimeoutException
    {
        return new StepCounterScore(
                service,
                Score.mustDeploy(service, wallet, PATH, null),
                nid
        );
    }

    public StepCounterScore(IconService service, Address target, BigInteger nid) {
        super(service, target, nid);
    }

    public TransactionResult increaseStep(Wallet wallet) throws IOException, TimeoutException {
        return this.invokeAndWaitResult(wallet,
                "increaseStep", null, null, STEPS);
    }

    public TransactionResult setStep(Wallet wallet, BigInteger step) throws IOException, TimeoutException {
        return this.invokeAndWaitResult(wallet,
                "setStep",
                (new RpcObject.Builder())
                    .put("step", new RpcValue(step))
                    .build(),
                null, STEPS);
    }

    public TransactionResult resetStep(Wallet wallet, BigInteger step) throws IOException, TimeoutException {
        return this.invokeAndWaitResult(wallet,
                "resetStep",
                (new RpcObject.Builder())
                        .put("step", new RpcValue(step))
                        .build(),
                null, STEPS);
    }

    public TransactionResult setStepOf(Wallet wallet, Address target, BigInteger step) throws IOException, TimeoutException {
        return this.invokeAndWaitResult(wallet,
                "setStepOf",
                (new RpcObject.Builder())
                    .put("step", new RpcValue(step))
                    .put( "addr", new RpcValue(target))
                    .build(),
                null, STEPS);
    }

    public TransactionResult trySetStepWith(Wallet wallet, Address target, BigInteger step) throws IOException, TimeoutException {
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
