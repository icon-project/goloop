/*
 * Copyright 2019 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package foundation.icon.test.score;

import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.IconAmount;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.common.Utils;

import java.io.IOException;
import java.math.BigInteger;

public class MultiSigWalletScore extends Score {
    private static final String SCORE_MULTISIG_PATH = Constants.SCORE_ROOT + "multisig_wallet";

    public MultiSigWalletScore(Score other) {
        super(other);
    }

    public static MultiSigWalletScore mustDeploy(TransactionHandler txHandler, Wallet wallet,
                                                 Address[] walletOwners, int required)
            throws IOException, TransactionFailureException, ResultTimeoutException {
        StringBuilder buf = new StringBuilder();
        for (Address walletOwner : walletOwners) {
            buf.append(walletOwner.toString()).append(",");
        }
        String str = buf.substring(0, buf.length() - 1);
        RpcObject params = new RpcObject.Builder()
                .put("_walletOwners", new RpcValue(str))
                .put("_required", new RpcValue(BigInteger.valueOf(required)))
                .build();
        return new MultiSigWalletScore(
                txHandler.deploy(wallet, SCORE_MULTISIG_PATH, params)
        );
    }

    public TransactionResult submitIcxTransaction(Wallet fromWallet, Address dest, long value, String description)
            throws IOException, ResultTimeoutException {
        BigInteger icx = IconAmount.of(BigInteger.valueOf(value), IconAmount.Unit.ICX).toLoop();
        RpcObject params = new RpcObject.Builder()
                .put("_destination", new RpcValue(dest))
                .put("_value", new RpcValue(icx))
                .put("_description", new RpcValue(description))
                .build();
        return invokeAndWaitResult(fromWallet, "submitTransaction", params);
    }

    public TransactionResult confirmTransaction(Wallet fromWallet, BigInteger txId)
            throws IOException, ResultTimeoutException {
        RpcObject params = new RpcObject.Builder()
                .put("_transactionId", new RpcValue(txId))
                .build();
        return invokeAndWaitResult(fromWallet, "confirmTransaction", params);
    }

    public TransactionResult addWalletOwner(Wallet fromWallet, Address newOwner, String description)
            throws IOException, ResultTimeoutException {
        String methodParams = String.format("[{\"name\": \"_walletOwner\", \"type\": \"Address\", \"value\": \"%s\"}]", newOwner);
        RpcObject params = new RpcObject.Builder()
                .put("_destination", new RpcValue(getAddress()))
                .put("_method", new RpcValue("addWalletOwner"))
                .put("_params", new RpcValue(methodParams))
                .put("_description", new RpcValue(description))
                .build();
        return invokeAndWaitResult(fromWallet, "submitTransaction", params);
    }

    public TransactionResult replaceWalletOwner(Wallet fromWallet, Address oldOwner, Address newOwner, String description)
            throws IOException, ResultTimeoutException {
        String methodParams = String.format(
                "[{\"name\": \"_walletOwner\", \"type\": \"Address\", \"value\": \"%s\"},"
                        + "{\"name\": \"_newWalletOwner\", \"type\": \"Address\", \"value\": \"%s\"}]", oldOwner, newOwner);
        RpcObject params = new RpcObject.Builder()
                .put("_destination", new RpcValue(getAddress()))
                .put("_method", new RpcValue("replaceWalletOwner"))
                .put("_params", new RpcValue(methodParams))
                .put("_description", new RpcValue(description))
                .build();
        return invokeAndWaitResult(fromWallet, "submitTransaction", params);
    }

    public TransactionResult changeRequirement(Wallet fromWallet, int required, String description)
            throws IOException, ResultTimeoutException {
        String methodParams = String.format("[{\"name\": \"_required\", \"type\": \"int\", \"value\": \"%d\"}]", required);
        RpcObject params = new RpcObject.Builder()
                .put("_destination", new RpcValue(getAddress()))
                .put("_method", new RpcValue("changeRequirement"))
                .put("_params", new RpcValue(methodParams))
                .put("_description", new RpcValue(description))
                .build();
        return invokeAndWaitResult(fromWallet, "submitTransaction", params);
    }

    public BigInteger getTransactionId(TransactionResult result) throws IOException {
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, getAddress(), "Submission(int)");
        if (event != null) {
            return event.getIndexed().get(1).asInteger();
        }
        throw new IOException("Failed to get transactionId.");
    }

    public void ensureConfirmation(TransactionResult result, Address sender, BigInteger txId) throws IOException {
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, getAddress(), "Confirmation(Address,int)");
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
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, getAddress(), "ICXTransfer(Address,Address,int)");
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
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, getAddress(), "Execution(int)");
        if (event != null) {
            BigInteger _txId = event.getIndexed().get(1).asInteger();
            if (txId.equals(_txId)) {
                return; // ensured
            }
        }
        throw new IOException("Failed to get Execution.");
    }

    public void ensureWalletOwnerAddition(TransactionResult result, Address address) throws IOException {
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, getAddress(), "WalletOwnerAddition(Address)");
        if (event != null) {
            Address _address = event.getIndexed().get(1).asAddress();
            if (address.equals(_address)) {
                return; // ensured
            }
        }
        throw new IOException("Failed to get WalletOwnerAddition.");
    }

    public void ensureWalletOwnerRemoval(TransactionResult result, Address address) throws IOException {
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, getAddress(), "WalletOwnerRemoval(Address)");
        if (event != null) {
            Address _address = event.getIndexed().get(1).asAddress();
            if (address.equals(_address)) {
                return; // ensured
            }
        }
        throw new IOException("Failed to get WalletOwnerRemoval.");
    }

    public void ensureRequirementChange(TransactionResult result, Integer required) throws IOException {
        TransactionResult.EventLog event = Utils.findEventLogWithFuncSig(result, getAddress(), "RequirementChange(int)");
        if (event != null) {
            BigInteger _required = event.getData().get(0).asInteger();
            if (required.equals(_required.intValue())) {
                return; // ensured
            }
        }
        throw new IOException("Failed to get RequirementChange.");
    }
}
