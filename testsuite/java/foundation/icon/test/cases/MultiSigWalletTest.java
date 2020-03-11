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

package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TestBase;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.HelloWorld;
import foundation.icon.test.score.MultiSigWalletScore;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;

public class MultiSigWalletTest extends TestBase {
    private static TransactionHandler txHandler;
    private static KeyWallet[] wallets;

    @BeforeAll
    static void setUp() throws Exception {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        IconService iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);

        // init wallets
        wallets = new KeyWallet[3];
        for (int i = 0; i < wallets.length; i++) {
            wallets[i] = KeyWallet.create();
        }
    }

    @Tag(Constants.TAG_PY_SCORE)
    @Test
    public void testPython() throws Exception {
        deployAndStartTest(Constants.CONTENT_TYPE_PYTHON);
    }

    @Tag(Constants.TAG_INTER_SCORE)
    @Test
    public void testJava() throws Exception {
        deployAndStartTest(Constants.CONTENT_TYPE_JAVA);
    }

    private void deployAndStartTest(String contentType) throws Exception {
        // deploy MultiSigWallet SCORE
        Address[] walletOwners = new Address[] {
                wallets[0].getAddress(), wallets[1].getAddress(), wallets[2].getAddress()};
        MultiSigWalletScore multiSigWalletScore = MultiSigWalletScore.mustDeploy(txHandler,
                wallets[0], walletOwners, 2, contentType);
        startTest(multiSigWalletScore);
    }

    private void startTest(MultiSigWalletScore multiSigWalletScore) throws Exception {
        LOG.infoEntering("setup", "initial wallets");
        KeyWallet ownerWallet = wallets[0];
        KeyWallet aliceWallet = wallets[1];
        KeyWallet bobWallet = wallets[2];
        LOG.info("Address of owner: " + ownerWallet.getAddress());
        LOG.info("Address of Alice: " + aliceWallet.getAddress());
        LOG.info("Address of Bob:   " + bobWallet.getAddress());
        Address multiSigWalletAddress = multiSigWalletScore.getAddress();

        // send 3 icx to the multiSigWallet
        LOG.info("transfer: 5 icx to multiSigWallet");
        final BigInteger icx = BigInteger.TEN.pow(18);
        transferAndCheckResult(txHandler, multiSigWalletAddress, icx.multiply(BigInteger.valueOf(5)));
        Utils.ensureIcxBalance(txHandler, multiSigWalletAddress, 0, 5);
        LOG.infoExiting();

        // *** 1. Send 2 icx to Bob (EOA)
        LOG.infoEntering("call", "submitIcxTransaction() - send 2 icx to Bob");
        // tx is initiated by ownerWallet first
        TransactionResult result = multiSigWalletScore.submitIcxTransaction(
                ownerWallet, bobWallet.getAddress(), icx.multiply(BigInteger.TWO), "send 2 icx to Bob");
        BigInteger txId = multiSigWalletScore.getTransactionId(result);

        // Alice confirms the tx to make it executed
        LOG.info("confirmTransaction() by Alice");
        result = multiSigWalletScore.confirmTransaction(aliceWallet, txId);
        multiSigWalletScore.ensureIcxTransfer(result, multiSigWalletAddress, bobWallet.getAddress(), 2);
        multiSigWalletScore.ensureExecution(result, txId);

        // check icx balances
        Utils.ensureIcxBalance(txHandler, multiSigWalletAddress, 5, 3);
        Utils.ensureIcxBalance(txHandler, bobWallet.getAddress(), 0, 2);
        LOG.infoExiting();

        // *** 2. Send 1 icx to Contract
        LOG.infoEntering("call", "submitIcxTransaction() - send 1 icx to hello");
        // deploy another sample score to accept icx
        LOG.info("deploy: HelloWorld");
        HelloWorld helloScore = HelloWorld.install(txHandler, ownerWallet);
        // tx is initiated by ownerWallet first
        result = multiSigWalletScore.submitIcxTransaction(
                ownerWallet, helloScore.getAddress(), icx.multiply(BigInteger.ONE), "send 1 icx to hello");
        txId = multiSigWalletScore.getTransactionId(result);

        // Bob confirms the tx to make it executed
        LOG.info("confirmTransaction() by Bob");
        result = multiSigWalletScore.confirmTransaction(bobWallet, txId);
        multiSigWalletScore.ensureIcxTransfer(result, multiSigWalletAddress, helloScore.getAddress(), 1);
        multiSigWalletScore.ensureExecution(result, txId);

        // check icx balances
        Utils.ensureIcxBalance(txHandler, multiSigWalletAddress, 3, 2);
        Utils.ensureIcxBalance(txHandler, helloScore.getAddress(), 0, 1);
        LOG.infoExiting();

        // *** 3. Send a test transaction (this will not be executed intentionally)
        LOG.infoEntering("call", "submitIcxTransaction() - pending transaction");
        result = multiSigWalletScore.submitIcxTransaction(
                ownerWallet, aliceWallet.getAddress(), icx.multiply(BigInteger.TWO), "send 2 icx to Alice");
        BigInteger pendingTx = multiSigWalletScore.getTransactionId(result);
        LOG.infoExiting();

        // *** 4. Add new wallet owner (charlie)
        LOG.infoEntering("call", "addWalletOwner(Charlie)");
        KeyWallet charlieWallet = KeyWallet.create();
        LOG.info("Address of Charlie: " + charlieWallet.getAddress());
        result = multiSigWalletScore.addWalletOwner(aliceWallet, charlieWallet.getAddress(), "add new wallet owner");
        txId = multiSigWalletScore.getTransactionId(result);

        // Revocation test
        LOG.info("revokeTransaction() by Alice");
        result = multiSigWalletScore.revokeTransaction(aliceWallet, txId);
        multiSigWalletScore.ensureRevocation(result, aliceWallet.getAddress(), txId);
        multiSigWalletScore.getConfirmationsAndCheck(txId);

        // Owner and Bob confirm the tx to make it executed
        LOG.info("confirmTransaction() by Owner and Bob");
        result = multiSigWalletScore.confirmTransaction(ownerWallet, txId);
        result = multiSigWalletScore.confirmTransaction(bobWallet, txId);
        multiSigWalletScore.ensureWalletOwnerAddition(result, charlieWallet.getAddress());
        multiSigWalletScore.ensureExecution(result, txId);
        multiSigWalletScore.ensureOwners(
                ownerWallet.getAddress(), aliceWallet.getAddress(), bobWallet.getAddress(), charlieWallet.getAddress());
        LOG.infoExiting();

        // *** 5. Replace wallet owner (bob -> david)
        LOG.infoEntering("call", "replaceWalletOwner(Bob to David)");
        KeyWallet davidWallet = KeyWallet.create();
        LOG.info("Address of David: " + davidWallet.getAddress());
        result = multiSigWalletScore.replaceWalletOwner(aliceWallet, bobWallet.getAddress(),
                davidWallet.getAddress(), "replace wallet owner");
        txId = multiSigWalletScore.getTransactionId(result);

        // Charlie confirms the tx to make it executed
        LOG.info("confirmTransaction() by Charlie");
        result = multiSigWalletScore.confirmTransaction(charlieWallet, txId);
        multiSigWalletScore.ensureWalletOwnerRemoval(result, bobWallet.getAddress());
        multiSigWalletScore.ensureWalletOwnerAddition(result, davidWallet.getAddress());
        multiSigWalletScore.ensureExecution(result, txId);
        multiSigWalletScore.ensureOwners(
                ownerWallet.getAddress(), aliceWallet.getAddress(), davidWallet.getAddress(), charlieWallet.getAddress());
        LOG.infoExiting();

        // *** 6. Change requirement
        LOG.infoEntering("call", "changeRequirement(3)");
        // check the current requirement first
        RpcItem item = multiSigWalletScore.call("getRequirement", null);
        assertEquals(2, item.asInteger().intValue());
        // tx is initiated by ownerWallet first
        result = multiSigWalletScore.changeRequirement(ownerWallet, 3, "change requirement to 3");
        txId = multiSigWalletScore.getTransactionId(result);

        // Alice confirms the tx to make it executed
        LOG.info("confirmTransaction() by Alice");
        result = multiSigWalletScore.confirmTransaction(aliceWallet, txId);
        multiSigWalletScore.ensureRequirementChange(result, 3);
        multiSigWalletScore.ensureExecution(result, txId);
        // check the changed requirement
        item = multiSigWalletScore.call("getRequirement", null);
        assertEquals(3, item.asInteger().intValue());
        LOG.infoExiting();

        // *** 7. Remove wallet owner
        LOG.infoEntering("call", "removeWalletOwner(owner)");
        result = multiSigWalletScore.removeWalletOwner(aliceWallet, ownerWallet.getAddress(), "remove the owner");
        txId = multiSigWalletScore.getTransactionId(result);

        // Charlie and David confirm the tx to make it executed
        LOG.info("confirmTransaction() by Charlie");
        result = multiSigWalletScore.confirmTransaction(charlieWallet, txId);
        multiSigWalletScore.ensureConfirmationCount(txId, 2);
        multiSigWalletScore.getConfirmationsAndCheck(txId,
                aliceWallet.getAddress(), charlieWallet.getAddress());
        // check getTransactionCount before executing the tx
        multiSigWalletScore.ensureTransactionCount(2, 5);
        // check getTransactionList before executing the tx
        multiSigWalletScore.ensurePendingTransactionIds(0, 7, pendingTx, txId);

        LOG.info("confirmTransaction() by David");
        result = multiSigWalletScore.confirmTransaction(davidWallet, txId);
        multiSigWalletScore.ensureWalletOwnerRemoval(result, ownerWallet.getAddress());
        multiSigWalletScore.ensureExecution(result, txId);
        multiSigWalletScore.ensureConfirmationCount(txId, 3);
        multiSigWalletScore.getConfirmationsAndCheck(txId,
                charlieWallet.getAddress(), aliceWallet.getAddress(), davidWallet.getAddress());
        multiSigWalletScore.ensureOwners(
                charlieWallet.getAddress(), aliceWallet.getAddress(), davidWallet.getAddress());
        LOG.infoExiting();
    }
}
