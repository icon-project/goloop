package foundation.icon.test.cases;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.*;
import org.junit.BeforeClass;

/*
sendTransaction with call
icx_call
stepUsed is bigger than specified stepLimit
 */
public class ScoreTest {
    private static IconService iconService;
    private static Env.Chain chain;
    private static KeyWallet godWallet;
    private static KeyWallet ownerWallet;

    @BeforeClass
    public static void init() throws Exception {
        Env.Node node = Env.nodes[0];
        chain = Env.nodes[0].chains[0];
        godWallet = chain.godWallet;
        iconService = new IconService(new HttpProvider(node.endpointUrl));
        ownerWallet = KeyWallet.create();
        initScore();
    }

    private static void initScore() throws Exception {
        Bytes txHash = Utils.transfer(iconService, godWallet, ownerWallet.getAddress(), 10000000);
        try {
            Utils.getTransactionResult(iconService, txHash, 5000);
        }
        catch (ResultTimeoutException ex) {
            System.out.println("Failed to transfer");
            throw ex;
        }
    }

    public void invalidParam() {
    }

    public void notEnoughStepLimit() {

    }
    public void notEnoughBalance() {

    }
    public void invalidAddress() {

    }
    public void notScoreAddress() {

    }
    public void withValue() {

    }
}
