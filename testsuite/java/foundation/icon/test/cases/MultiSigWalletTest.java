package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.HelloWorld;
import foundation.icon.test.score.MultiSigWalletScore;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;

import static foundation.icon.test.common.Env.LOG;

@Tag(Constants.TAG_PARALLEL)
public class MultiSigWalletTest {
    private static Env.Chain chain;
    private static IconService iconService;

    @BeforeAll
    public static void setUp() {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
    }

    @Test
    public void multiSigWalletTest() throws Exception {
        KeyWallet ownerWallet = KeyWallet.create();
        KeyWallet aliceWallet = KeyWallet.create();
        KeyWallet bobWallet = KeyWallet.create();

        LOG.info("Address of owner: " + ownerWallet.getAddress());
        LOG.info("Address of Alice: " + aliceWallet.getAddress());
        LOG.info("Address of Bob:   " + bobWallet.getAddress());

        // deploy MultiSigWallet multiSigWalletScore
        LOG.info("deploy: MultiSigWalletScore");
        Address[] walletOwners =
                new Address[] {ownerWallet.getAddress(), aliceWallet.getAddress(), bobWallet.getAddress()};
        MultiSigWalletScore multiSigWalletScore =
                MultiSigWalletScore.mustDeploy(iconService, chain, ownerWallet, walletOwners, 2);
        Address multiSigWalletAddress = multiSigWalletScore.getAddress();

        // send 3 icx to the multiSigWallet
        LOG.info("transfer: 3 icx to multiSigWallet");
        Utils.transferIcx(iconService, chain.networkId, chain.godWallet, multiSigWalletAddress, "3");
        Utils.ensureIcxBalance(iconService, multiSigWalletAddress, 0, 3);

        // *** Send 2 icx to Bob (EOA)
        // 1. tx is initiated by ownerWallet first
        LOG.infoEntering("call", "submitIcxTransaction() - send 2 icx to Bob");
        // TODO check txHash, txId
        TransactionResult result =
                multiSigWalletScore.submitIcxTransaction(ownerWallet, bobWallet.getAddress(), 2, "send 2 icx to Bob");
        BigInteger txId = multiSigWalletScore.getTransactionId(result);
        LOG.infoExiting();

        // 2. Alice confirms the tx to make the tx executedS
        LOG.infoEntering("call", "confirmTransaction(Alice)");
        result = multiSigWalletScore.confirmTransaction(aliceWallet, txId);

        multiSigWalletScore.ensureConfirmation(result, aliceWallet.getAddress(), txId);
        multiSigWalletScore.ensureIcxTransfer(result, multiSigWalletAddress, bobWallet.getAddress(), 2);
        multiSigWalletScore.ensureExecution(result, txId);

        // check icx balances
        Utils.ensureIcxBalance(iconService, multiSigWalletAddress, 3, 1);
        Utils.ensureIcxBalance(iconService, bobWallet.getAddress(), 0, 2);
        LOG.infoExiting();

        // *** Send 1 icx to Contract
        // deploy sample multiSigWalletScore to accept icx
        LOG.info("deploy: HelloWorld");
        HelloWorld helloScore = HelloWorld.install(iconService, chain, ownerWallet);

        // 3. tx is initiated by ownerWallet first
        LOG.infoEntering("call", "submitIcxTransaction() - send 1 icx to hello");
        result = multiSigWalletScore.submitIcxTransaction(ownerWallet, helloScore.getAddress(), 1, "send 1 icx to hello");
        txId = multiSigWalletScore.getTransactionId(result);
        LOG.infoExiting();

        // 4. Bob confirms the tx to make the tx executed
        LOG.infoEntering("call", "confirmTransaction(Bob)");
        result = multiSigWalletScore.confirmTransaction(bobWallet, txId);

        multiSigWalletScore.ensureConfirmation(result, bobWallet.getAddress(), txId);
        multiSigWalletScore.ensureIcxTransfer(result, multiSigWalletAddress, helloScore.getAddress(), 1);
        multiSigWalletScore.ensureExecution(result, txId);

        // check icx balances
        Utils.ensureIcxBalance(iconService, multiSigWalletAddress, 1, 0);
        Utils.ensureIcxBalance(iconService, helloScore.getAddress(), 0, 1);
        LOG.infoExiting();

        // *** Add new wallet owner (charlie)
        KeyWallet charlieWallet = KeyWallet.create();
        // 5. tx is initiated by ownerWallet first
        LOG.infoEntering("call", "addWalletOwner(Charlie)");
        result = multiSigWalletScore.addWalletOwner(ownerWallet, charlieWallet.getAddress(), "add new wallet owner");
        txId = multiSigWalletScore.getTransactionId(result);
        LOG.infoExiting();

        // 6. Alice confirms the tx to make the tx executed
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

        // 8. Alice confirms the tx to make the tx executed
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

        // 10. Alice confirms the tx to make the tx executed
        LOG.infoEntering("call", "confirmTransaction() by Alice");
        result = multiSigWalletScore.confirmTransaction(aliceWallet, txId);

        multiSigWalletScore.ensureConfirmation(result, aliceWallet.getAddress(), txId);
        multiSigWalletScore.ensureRequirementChange(result, 3);
        multiSigWalletScore.ensureExecution(result, txId);
        LOG.infoExiting();
    }
}
