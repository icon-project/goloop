package foundation.icon.test.score;

import foundation.icon.icx.IconService;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.IconAmount;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.Utils;

import java.io.IOException;
import java.math.BigInteger;

public class MultiSigWalletScore extends Score {
    private static final BigInteger STEPS = BigInteger.valueOf(10000000);
    private static final String PATH = Constants.SCORE_ROOT + "multiSigWallet.zip";

    public static MultiSigWalletScore mustDeploy(IconService service, Wallet wallet, BigInteger nid,
                                                 Address[] walletOwners, int required)
            throws IOException, TransactionFailureException, ResultTimeoutException {
        StringBuffer buf = new StringBuffer();
        for (int i = 0; i < walletOwners.length; i++) {
            buf.append(walletOwners[i].toString()).append(",");
        }
        String str = buf.substring(0, buf.length() - 1);
        RpcObject params = new RpcObject.Builder()
                .put("_walletOwners", new RpcValue(str))
                .put("_required", new RpcValue(BigInteger.valueOf(required)))
                .build();
        return new MultiSigWalletScore(
                service,
                Score.mustDeploy(service, wallet, PATH, params),
                nid
        );
    }

    public MultiSigWalletScore(IconService iconService, Address scoreAddress, BigInteger nid) {
        super(iconService, scoreAddress, nid);
    }

    public TransactionResult submitIcxTransaction(Wallet fromWallet, Address dest, long value, String description) throws IOException, ResultTimeoutException {
        BigInteger icx = IconAmount.of(BigInteger.valueOf(value), IconAmount.Unit.ICX).toLoop();
        RpcObject params = new RpcObject.Builder()
                .put("_destination", new RpcValue(dest))
                .put("_value", new RpcValue(icx))
                .put("_description", new RpcValue(description))
                .build();
        return invokeAndWaitResult(fromWallet, "submitTransaction", params, null, STEPS);
    }

    public TransactionResult confirmTransaction(Wallet fromWallet, BigInteger txId) throws IOException, ResultTimeoutException {
        RpcObject params = new RpcObject.Builder()
                .put("_transactionId", new RpcValue(txId))
                .build();
        return invokeAndWaitResult(fromWallet, "confirmTransaction", params, null, STEPS);
    }

    public TransactionResult addWalletOwner(Wallet fromWallet, Address newOwner, String description) throws IOException, ResultTimeoutException {
        String methodParams = String.format("[{\"name\": \"_walletOwner\", \"type\": \"Address\", \"value\": \"%s\"}]", newOwner);
        RpcObject params = new RpcObject.Builder()
                .put("_destination", new RpcValue(scoreAddress))
                .put("_method", new RpcValue("addWalletOwner"))
                .put("_params", new RpcValue(methodParams))
                .put("_description", new RpcValue(description))
                .build();
        return invokeAndWaitResult(fromWallet, "submitTransaction", params, null, STEPS);
    }

    public TransactionResult replaceWalletOwner(Wallet fromWallet, Address oldOwner, Address newOwner, String description) throws IOException, ResultTimeoutException {
        String methodParams = String.format(
                "[{\"name\": \"_walletOwner\", \"type\": \"Address\", \"value\": \"%s\"},"
                        + "{\"name\": \"_newWalletOwner\", \"type\": \"Address\", \"value\": \"%s\"}]", oldOwner, newOwner);
        RpcObject params = new RpcObject.Builder()
                .put("_destination", new RpcValue(scoreAddress))
                .put("_method", new RpcValue("replaceWalletOwner"))
                .put("_params", new RpcValue(methodParams))
                .put("_description", new RpcValue(description))
                .build();
        return invokeAndWaitResult(fromWallet, "submitTransaction", params, null, STEPS);
    }

    public TransactionResult changeRequirement(Wallet fromWallet, int required, String description) throws IOException, ResultTimeoutException {
        String methodParams = String.format("[{\"name\": \"_required\", \"type\": \"int\", \"value\": \"%d\"}]", required);
        RpcObject params = new RpcObject.Builder()
                .put("_destination", new RpcValue(scoreAddress))
                .put("_method", new RpcValue("changeRequirement"))
                .put("_params", new RpcValue(methodParams))
                .put("_description", new RpcValue(description))
                .build();
        return invokeAndWaitResult(fromWallet, "submitTransaction", params, null, STEPS);
    }

    public BigInteger getTransactionId(TransactionResult result) throws IOException {
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, scoreAddress, "Submission(int)");
        if (event != null) {
            return event.getIndexed().get(1).asInteger();
        }
        throw new IOException("Failed to get transactionId.");
    }

    public void ensureConfirmation(TransactionResult result, Address sender, BigInteger txId) throws IOException {
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, scoreAddress, "Confirmation(Address,int)");
        if (event != null) {
            Address _sender = event.getIndexed().get(1).asAddress();
            BigInteger _txId = event.getIndexed().get(2).asInteger();
            if (sender.equals(_sender) && txId.equals(_txId)) {
                return; // ensured
            }
        }
        throw new IOException("Failed to get Confirmation.");
    }

    public void ensureIcxTransfer(TransactionResult result, Address from, Address to, long value) throws IOException {
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, scoreAddress, "ICXTransfer(Address,Address,int)");
        if (event != null) {
            BigInteger icxValue = IconAmount.of(BigInteger.valueOf(value), IconAmount.Unit.ICX).toLoop();
            Address _from = event.getIndexed().get(1).asAddress();
            Address _to = event.getIndexed().get(2).asAddress();
            BigInteger _value = event.getIndexed().get(3).asInteger();
            if (from.equals(_from) && to.equals(_to) && icxValue.equals(_value)) {
                return; // ensured
            }
        }
        throw new IOException("Failed to get ICXTransfer.");
    }

    public void ensureExecution(TransactionResult result, BigInteger txId) throws IOException {
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, scoreAddress, "Execution(int)");
        if (event != null) {
            BigInteger _txId = event.getIndexed().get(1).asInteger();
            if (txId.equals(_txId)) {
                return; // ensured
            }
        }
        throw new IOException("Failed to get Execution.");
    }

    public void ensureWalletOwnerAddition(TransactionResult result, Address address) throws IOException {
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, scoreAddress, "WalletOwnerAddition(Address)");
        if (event != null) {
            Address _address = event.getIndexed().get(1).asAddress();
            if (address.equals(_address)) {
                return; // ensured
            }
        }
        throw new IOException("Failed to get WalletOwnerAddition.");
    }

    public void ensureWalletOwnerRemoval(TransactionResult result, Address address) throws IOException {
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, scoreAddress, "WalletOwnerRemoval(Address)");
        if (event != null) {
            Address _address = event.getIndexed().get(1).asAddress();
            if (address.equals(_address)) {
                return; // ensured
            }
        }
        throw new IOException("Failed to get WalletOwnerRemoval.");
    }

    public void ensureRequirementChange(TransactionResult result, Integer required) throws IOException {
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, scoreAddress, "RequirementChange(int)");
        if (event != null) {
            BigInteger _required = event.getData().get(0).asInteger();
            if (required.equals(_required.intValue())) {
                return; // ensured
            }
        }
        throw new IOException("Failed to get RequirementChange.");
    }
}
