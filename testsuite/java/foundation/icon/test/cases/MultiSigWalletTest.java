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

    @Tag(Constants.TAG_JAVA_SCORE)
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
        KeyWallet ownerWallet = wallets[0];
        KeyWallet aliceWallet = wallets[1];
        KeyWallet bobWallet = wallets[2];
        LOG.info("Address of owner: " + ownerWallet.getAddress());
        LOG.info("Address of Alice: " + aliceWallet.getAddress());
        LOG.info("Address of Bob:   " + bobWallet.getAddress());
        Address multiSigWalletAddress = multiSigWalletScore.getAddress();

        // send 3 icx to the multiSigWallet
        LOG.info("transfer: 3 icx to multiSigWallet");
        final BigInteger icx = BigInteger.TEN.pow(18);
        transferAndCheckResult(txHandler, multiSigWalletAddress, icx.multiply(BigInteger.valueOf(3)));
        Utils.ensureIcxBalance(txHandler, multiSigWalletAddress, 0, 3);

        // *** Send 2 icx to Bob (EOA)
        // 1. tx is initiated by ownerWallet first
        LOG.infoEntering("call", "submitIcxTransaction() - send 2 icx to Bob");
        TransactionResult result =
                multiSigWalletScore.submitIcxTransaction(ownerWallet, bobWallet.getAddress(), 2, "send 2 icx to Bob");
        BigInteger txId = multiSigWalletScore.getTransactionId(result);
        LOG.infoExiting();

        // 2. Alice confirms the tx to make it executed
        LOG.infoEntering("call", "confirmTransaction() by Alice");
        result = multiSigWalletScore.confirmTransaction(aliceWallet, txId);

        multiSigWalletScore.ensureConfirmation(result, aliceWallet.getAddress(), txId);
        multiSigWalletScore.ensureIcxTransfer(result, multiSigWalletAddress, bobWallet.getAddress(), 2);
        multiSigWalletScore.ensureExecution(result, txId);

        // check icx balances
        Utils.ensureIcxBalance(txHandler, multiSigWalletAddress, 3, 1);
        Utils.ensureIcxBalance(txHandler, bobWallet.getAddress(), 0, 2);
        LOG.infoExiting();

        // *** Send 1 icx to Contract
        // deploy another sample score to accept icx
        LOG.info("deploy: HelloWorld");
        HelloWorld helloScore = HelloWorld.install(txHandler, ownerWallet);

        // 3. tx is initiated by ownerWallet first
        LOG.infoEntering("call", "submitIcxTransaction() - send 1 icx to hello");
        result = multiSigWalletScore.submitIcxTransaction(ownerWallet, helloScore.getAddress(), 1, "send 1 icx to hello");
        txId = multiSigWalletScore.getTransactionId(result);
        LOG.infoExiting();

        // 4. Bob confirms the tx to make it executed
        LOG.infoEntering("call", "confirmTransaction() by Bob");
        result = multiSigWalletScore.confirmTransaction(bobWallet, txId);

        multiSigWalletScore.ensureConfirmation(result, bobWallet.getAddress(), txId);
        multiSigWalletScore.ensureIcxTransfer(result, multiSigWalletAddress, helloScore.getAddress(), 1);
        multiSigWalletScore.ensureExecution(result, txId);

        // check icx balances
        Utils.ensureIcxBalance(txHandler, multiSigWalletAddress, 1, 0);
        Utils.ensureIcxBalance(txHandler, helloScore.getAddress(), 0, 1);
        LOG.infoExiting();

        // *** Add new wallet owner (charlie)
        KeyWallet charlieWallet = KeyWallet.create();
        // 5. tx is initiated by ownerWallet first
        LOG.infoEntering("call", "addWalletOwner(Charlie)");
        result = multiSigWalletScore.addWalletOwner(ownerWallet, charlieWallet.getAddress(), "add new wallet owner");
        txId = multiSigWalletScore.getTransactionId(result);
        LOG.infoExiting();

        // 6. Alice confirms the tx to make it executed
        LOG.infoEntering("call", "confirmTransaction() by Alice");
        result = multiSigWalletScore.confirmTransaction(aliceWallet, txId);

        multiSigWalletScore.ensureConfirmation(result, aliceWallet.getAddress(), txId);
        multiSigWalletScore.ensureWalletOwnerAddition(result, charlieWallet.getAddress());
        multiSigWalletScore.ensureExecution(result, txId);
        LOG.infoExiting();

        // *** Replace wallet owner (charlie -> david)
        KeyWallet davidWallet = KeyWallet.create();
        // 7. tx is initiated by ownerWallet first
        LOG.infoEntering("call", "replaceWalletOwner(Charlie to David)");
        result = multiSigWalletScore.replaceWalletOwner(ownerWallet, charlieWallet.getAddress(),
                davidWallet.getAddress(), "replace wallet owner");
        txId = multiSigWalletScore.getTransactionId(result);
        LOG.infoExiting();

        // 8. Alice confirms the tx to make it executed
        LOG.infoEntering("call", "confirmTransaction() by Alice");
        result = multiSigWalletScore.confirmTransaction(aliceWallet, txId);

        multiSigWalletScore.ensureConfirmation(result, aliceWallet.getAddress(), txId);
        multiSigWalletScore.ensureWalletOwnerRemoval(result, charlieWallet.getAddress());
        multiSigWalletScore.ensureWalletOwnerAddition(result, davidWallet.getAddress());
        multiSigWalletScore.ensureExecution(result, txId);
        LOG.infoExiting();

        // *** Change requirement
        // 9. tx is initiated by ownerWallet first
        LOG.infoEntering("call", "changeRequirement(3)");
        result = multiSigWalletScore.changeRequirement(ownerWallet, 3, "change requirement to 3");
        txId = multiSigWalletScore.getTransactionId(result);
        LOG.infoExiting();

        // 10. Alice confirms the tx to make it executed
        LOG.infoEntering("call", "confirmTransaction() by Alice");
        result = multiSigWalletScore.confirmTransaction(aliceWallet, txId);

        multiSigWalletScore.ensureConfirmation(result, aliceWallet.getAddress(), txId);
        multiSigWalletScore.ensureRequirementChange(result, 3);
        multiSigWalletScore.ensureExecution(result, txId);
        LOG.infoExiting();
    }
}
