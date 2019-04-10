package foundation.icon.test.score;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;

import java.math.BigInteger;

public class GovScore extends Score {
    private KeyWallet govWallet;
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

    public GovScore(IconService iconService, BigInteger nid, KeyWallet govWallet) {
        super(iconService, Constants.GOV_ADDRESS, nid);
        this.govWallet = govWallet;
    }

    public void setStepPrice(BigInteger price) throws Exception{
        RpcObject params = new RpcObject.Builder()
                .put("stepPrice", new RpcValue(price))
                .build();
        invokeAndWaitResult(govWallet, "setStepPrice", params, 0, 1000);
    }

    public void setStepCost(String type, BigInteger cost) throws Exception{
        RpcObject params = new RpcObject.Builder()
                .put("stepType", new RpcValue(type))
                .put("cost", new RpcValue(cost))
                .build();
        invokeAndWaitResult(govWallet, "setStepCost", params, 0, 1000);
    }

    public void setMaxStepLimit(String type, BigInteger cost) throws Exception{
        RpcObject params = new RpcObject.Builder()
                .put("contextType", new RpcValue(type))
                .put("value", new RpcValue(cost))
                .build();
        invokeAndWaitResult(govWallet, "setMaxStepLimit", params, 0, 1000);
    }
}
