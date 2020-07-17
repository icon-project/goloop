package foundation.icon.test.score;

import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import test.TBCProtocol;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;
import test.TBCInterpreter;

import java.io.IOException;
import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;

public class TBCInterpreterScore extends Score {
    private static final BigInteger STEP = BigInteger.valueOf(10_000_000);

    public TBCInterpreterScore(Score other) throws IOException {
        super(other);
    }

    public static TBCInterpreterScore mustDeploy(TransactionHandler txHandler,
            Wallet owner, String name, String contentType)
            throws ResultTimeoutException, TransactionFailureException,
            IOException {
        LOG.infoEntering("deploy", "TBCInterpreter");
        RpcObject params = new RpcObject.Builder()
                .put("_name", new RpcValue(name))
                .build();
        Score score;
        if (contentType.equals(Constants.CONTENT_TYPE_PYTHON)) {
            score = txHandler.deploy(owner, getFilePath("tbc_interpreter"), params);
        } else if (contentType.equals(Constants.CONTENT_TYPE_JAVA)) {
            score = txHandler.deploy(
                    owner,
                    new Class<?>[]{TBCInterpreter.class, TBCProtocol.class},
                    params
            );
        } else {
            throw new IllegalArgumentException("Unknown content type");
        }
        LOG.info("scoreAddr = " + score.getAddress());
        LOG.infoExiting();
        return new TBCInterpreterScore(score);
    }

    public TransactionResult runAndLogEvent(Wallet wallet, byte[] code) throws IOException,
            ResultTimeoutException {
        RpcObject.Builder builder = new RpcObject.Builder()
                .put("_code", new RpcValue(code));
        return this.invokeAndWaitResult(wallet, "runAndLogResult",
                builder.build(), null, STEP);
    }
}
