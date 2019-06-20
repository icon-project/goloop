package foundation.icon.test.score;

import foundation.icon.icx.IconService;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.*;

import java.io.IOException;
import java.math.BigInteger;
import java.util.HashMap;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;

public class GovScore extends Score {
    public class Fee {
        Map<String, BigInteger> stepCosts;
        Map<String, BigInteger> stepMaxLimits;
        BigInteger stepPrice;
    }

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

    public void setStepCost(String type, BigInteger cost) throws ResultTimeoutException, IOException{
        RpcObject params = new RpcObject.Builder()
                .put("type", new RpcValue(type))
                .put("cost", new RpcValue(cost))
                .build();
        invokeAndWaitResult(chain.governorWallet, "setStepCost", params, 0, stepLimit);
    }

    public void setMaxStepLimit(String type, BigInteger cost) throws ResultTimeoutException, IOException{
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

    public Map<String, BigInteger> getStepCosts() throws Exception {
        RpcItem rpcItem = Utils.icxCall(service,Constants.CHAINSCORE_ADDRESS, "getStepCosts", null);
        Map<String, BigInteger> map = new HashMap<>();
        for(String type : stepCostTypes) {
            map.put(type, rpcItem.asObject().getItem(type).asInteger());
        }
        return map;
    }

    public void setStepCosts(Map<String, BigInteger> map)
            throws ResultTimeoutException, TransactionFailureException, IOException {
        List<Bytes> list = new LinkedList<>();
        for(String type : map.keySet()) {
            RpcObject params = new RpcObject.Builder()
                    .put("type", new RpcValue(type))
                    .put("cost", new RpcValue(map.get(type)))
                    .build();
            Bytes txHash = invoke(chain.governorWallet, "setStepCost", params, 0, stepLimit);
            list.add(txHash);
        }
        for(Bytes txHash : list) {
            TransactionResult result = waitResult(txHash);
            if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
                throw new TransactionFailureException(result.getFailure());
            }
        }
    }

    public Map<String, BigInteger> getMaxStepLimits() throws Exception {
        Map<String, BigInteger> map = new HashMap<>();
        String types[] = {"invoke", "query"};
        for(String t : types) {
            RpcObject params = new RpcObject.Builder()
                    .put("contextType", new RpcValue(t))
                    .build();
            BigInteger stepLimit = Utils.icxCall(service,
                    Constants.CHAINSCORE_ADDRESS,"getMaxStepLimit", params).asInteger();
            map.put(t, stepLimit);
        }
        return map;
    }

    public void setMaxStepLimits(Map<String, BigInteger> limits)
            throws ResultTimeoutException, TransactionFailureException, IOException {
        List<Bytes> list = new LinkedList<>();
        for(String type : limits.keySet()) {
            RpcObject params = new RpcObject.Builder()
                    .put("contextType", new RpcValue(type))
                    .put("limit", new RpcValue(limits.get(type)))
                    .build();
            Bytes txHash = invoke(chain.governorWallet, "setMaxStepLimit", params, 0, stepLimit);
            list.add(txHash);
        }
        for(Bytes txHash : list) {
            TransactionResult result = waitResult(txHash);
            if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
                throw new TransactionFailureException(result.getFailure());
            }
        }
    }

    public Fee getFee() throws Exception {
        Fee fee = new Fee();
        fee.stepCosts = getStepCosts();
        fee.stepMaxLimits = getMaxStepLimits();
        fee.stepPrice = Utils.icxCall(service,
                Constants.CHAINSCORE_ADDRESS,"getStepPrice", null)
                .asInteger();
        return fee;
    }

    public void setFee(Fee fee) throws Exception {
        setStepPrice(fee.stepPrice);
        setStepCosts(fee.stepCosts);
        setMaxStepLimits(fee.stepMaxLimits);
    }
}
