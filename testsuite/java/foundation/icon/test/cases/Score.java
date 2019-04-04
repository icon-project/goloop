package foundation.icon.test.cases;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.test.common.Utils;

import java.math.BigInteger;

/*
sendTransaction with call
icx_call
stepUsed is bigger than specified stepLimit
 */
public class Score {
    public static Bytes sendTransaction(
            IconService iconService, BigInteger nid, Wallet fromWallet, Address scoreAddr, String function,
            RpcObject params, BigInteger value) throws Exception {

        long timestamp = System.currentTimeMillis() * 1000L;
        Transaction transaction = TransactionBuilder.of(nid)
                .from(fromWallet.getAddress())
                .to(scoreAddr)
                .stepLimit(new BigInteger("2000000"))
                .timestamp(new BigInteger(Long.toString(timestamp)))
                .nonce(new BigInteger("1"))
                .call(function)
                .params(params)
                .build();

        SignedTransaction signedTransaction = new SignedTransaction(transaction, fromWallet);
        Bytes txHash = iconService.sendTransaction(signedTransaction).execute();
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, 3);
        if (result != null) {
            if (!Utils.STATUS_SUCCESS.equals(result.getStatus())) {
                // TODO define Deploy fail exception
                throw new Exception("Failed to call.");
            }
            return null;
        }
        throw new Exception("Failed to call.");
    }

    public static RpcItem icxCall(
            IconService iconService, BigInteger nid, Wallet fromWallet, Address scoreAddr, String function,
            RpcObject params) throws Exception {
        // TODO NID
        Call.Builder builder = new Call.Builder()
                .from(fromWallet.getAddress())
                .to(scoreAddr)
                .method(function);
        if(params != null) {
            builder.params(params);
        }
        Call<RpcItem> call = builder.build();
        return iconService.call(call).execute();
    }
}
