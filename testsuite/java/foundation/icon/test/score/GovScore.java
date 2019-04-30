package foundation.icon.test.score;

import foundation.icon.icx.IconService;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.ResultTimeoutException;

import java.io.IOException;
import java.math.BigInteger;

public class GovScore extends Score {
    public static String []stepCostTypes = {
            "default",
            "contractCall",
            "contractCreate",
            "contractUpdate",
            "contractDestruct",
            "contractSet",
            "get",
            "set",
            "replace",
            "delete",
            "input",
            "eventLog",
            "apiCall"
    };
    final long stepLimit = 1000000;

    public GovScore(IconService iconService, Env.Chain chain) {
        super(iconService, chain, Constants.GOV_ADDRESS);
    }

    public void setStepPrice(BigInteger price) throws Exception{
        RpcObject params = new RpcObject.Builder()
                .put("price", new RpcValue(price))
                .build();
        invokeAndWaitResult(chain.governorWallet, "setStepPrice", params, 0, stepLimit);
    }

    public void setStepCost(String type, BigInteger cost) throws Exception{
        RpcObject params = new RpcObject.Builder()
                .put("type", new RpcValue(type))
                .put("cost", new RpcValue(cost))
                .build();
        invokeAndWaitResult(chain.governorWallet, "setStepCost", params, 0, stepLimit);
    }

    public void setMaxStepLimit(String type, BigInteger cost) throws Exception{
        RpcObject params = new RpcObject.Builder()
                .put("contextType", new RpcValue(type))
                .put("limit", new RpcValue(cost))
                .build();
        invokeAndWaitResult(chain.governorWallet, "setMaxStepLimit", params, 0, stepLimit);
    }

    public TransactionResult acceptScore(Bytes txHash) throws ResultTimeoutException, IOException {
        RpcObject params = new RpcObject.Builder()
                .put("txHash", new RpcValue(txHash))
                .build();
        return invokeAndWaitResult(chain.governorWallet, "acceptScore", params, 0, stepLimit);
    }

    public TransactionResult rejectScore(Bytes txHash) throws ResultTimeoutException, IOException {
        RpcObject params = new RpcObject.Builder()
                .put("txHash", new RpcValue(txHash))
                .build();
        return invokeAndWaitResult(chain.governorWallet, "rejectScore", params, 0, stepLimit);
    }
}
