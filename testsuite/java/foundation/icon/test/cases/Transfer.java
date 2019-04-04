package foundation.icon.test.cases;

import foundation.icon.icx.*;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Env;
import org.junit.Test;

import java.math.BigInteger;

/*
test cases
1. Not enough balance.
2. Not enough stepLimit.
3. Invalid signature
4. Transfer coin. check balances of both accounts with GetBalance api.
 - Check balances in every transaction.
 - Check
 set StepPrice 0 or not.
 -
5.
 */
public class Transfer {
    private KeyWallet god;
    private KeyWallet account;
    private KeyWallet[]testWallets;
    private final IconService iconService;
    private final Env.Chain chain;

    public Transfer() throws Exception {
        this.god = god;
        testWallets = new KeyWallet[10];
        for(int i = 0; i < testWallets.length; i++){
            testWallets[i] = KeyWallet.create();
        }
        Env.Node node = Env.nodes[0];
        chain = Env.nodes[0].chains[0];
        iconService = new IconService(new HttpProvider(node.endpointUrl));
    }

    public void notEnoughBalance() throws Exception{
        KeyWallet[]testWallets = new KeyWallet[10];
        for (int i = 0; i < testWallets.length; i++) {
            testWallets[i] = KeyWallet.create();
            if (iconService.getBalance(testWallets[i].getAddress()).execute()
                    .compareTo(BigInteger.valueOf(0)) != 0) {
                throw new Exception();
            }
            // transfer from GOD to test wallets
        }

    }

    public void notEnoughSteplimit() {

    }

    public void invalidSignature() {

    }

    public void transferAndCheckBal() throws Exception {
    }
}
