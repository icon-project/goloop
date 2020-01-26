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

package foundation.icon.test.common;

import foundation.icon.ee.tooling.deploy.OptimizedJarBuilder;
import foundation.icon.icx.Call;
import foundation.icon.icx.IconService;
import foundation.icon.icx.SignedTransaction;
import foundation.icon.icx.Transaction;
import foundation.icon.icx.TransactionBuilder;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.ScoreApi;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.test.score.Score;
import org.aion.avm.utilities.JarBuilder;

import java.io.IOException;
import java.math.BigInteger;
import java.util.List;

public class TransactionHandler {
    private final IconService iconService;
    private final Env.Chain chain;

    public TransactionHandler(IconService iconService, Env.Chain chain) {
        this.iconService = iconService;
        this.chain = chain;
    }

    public Score deploy(Wallet owner, String scorePath, RpcObject params)
            throws IOException, ResultTimeoutException, TransactionFailureException {
        byte[] data = Utils.zipContent(scorePath);
        return doDeploy(owner, data, params, Constants.CONTENT_TYPE_PYTHON);
    }

    public Score deploy(Wallet owner, Class<?> mainClass, RpcObject params)
            throws IOException, ResultTimeoutException, TransactionFailureException {
        byte[] jar = makeJar(mainClass);
        return doDeploy(owner, jar, params, Constants.CONTENT_TYPE_JAVA);
    }

    private byte[] makeJar(Class<?> c) {
        return makeJar(c.getName(), new Class<?>[]{c});
    }

    private byte[] makeJar(String name, Class<?>[] classes) {
        byte[] jarBytes = JarBuilder.buildJarForExplicitMainAndClasses(name, classes);
        return new OptimizedJarBuilder(false, jarBytes, true)
                .withUnreachableMethodRemover()
                .withRenamer()
                .getOptimizedBytes();
    }

    private Score doDeploy(Wallet owner, byte[] content, RpcObject params, String contentType)
            throws IOException, ResultTimeoutException, TransactionFailureException {
        Transaction transaction = TransactionBuilder.newBuilder()
                .nid(getNetworkId())
                .from(owner.getAddress())
                .to(Constants.CHAINSCORE_ADDRESS)
                .stepLimit(Constants.DEFAULT_STEPS)
                .deploy(contentType, content)
                .params(params)
                .build();

        SignedTransaction signedTransaction = new SignedTransaction(transaction, owner);
        Bytes txHash = iconService.sendTransaction(signedTransaction).execute();
        TransactionResult result = Utils.getTransactionResult(iconService, txHash, Constants.DEFAULT_WAITING_TIME);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new TransactionFailureException(result.getFailure());
        }
        Utils.acceptIfAuditEnabled(iconService, chain, txHash);
        return new Score(this, new Address(result.getScoreAddress()));
    }

    public BigInteger getNetworkId() {
        return BigInteger.valueOf(chain.networkId);
    }

    public List<ScoreApi> getScoreApi(Address scoreAddress) throws IOException {
        return iconService.getScoreApi(scoreAddress).execute();
    }

    public RpcItem call(Call<RpcItem> call) throws IOException {
        return this.iconService.call(call).execute();
    }

    public Bytes invoke(Wallet wallet, Transaction tx) throws IOException {
        return this.iconService.sendTransaction(new SignedTransaction(tx, wallet)).execute();
    }

    public TransactionResult invokeAndWait(Wallet wallet, Transaction tx) throws IOException {
        return this.iconService.sendTransactionAndWait(new SignedTransaction(tx, wallet)).execute();
    }

    public TransactionResult waitResult(Bytes txHash) throws IOException {
        return this.iconService.waitTransactionResult(txHash).execute();
    }

    public TransactionResult getResult(Bytes txHash, long waiting)
            throws IOException, ResultTimeoutException {
        return Utils.getTransactionResult(this.iconService, txHash, waiting);
    }
}
