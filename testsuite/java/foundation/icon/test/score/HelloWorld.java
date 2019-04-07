package foundation.icon.test.score;

import foundation.icon.icx.IconService;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Utils;

import java.io.IOException;
import java.math.BigInteger;
import java.util.concurrent.TimeoutException;

public class HelloWorld extends Score {
    private static final String PATH = Constants.SCORE_ROOT +  "helloWorld.zip";

    public static HelloWorld mustDeploy(IconService service, Wallet wallet, BigInteger nid)
            throws IOException, TransactionFailureException, TimeoutException
    {
        return new HelloWorld(
                service,
                Score.mustDeploy(service, wallet, PATH, null),
                nid
        );
    }

    public HelloWorld(IconService iconService, Address scoreAddress, BigInteger nid) {
        super(iconService, scoreAddress, nid);
    }

    public boolean invokeHello(Wallet from) throws TimeoutException{
        try {
            TransactionResult txResult = invokeAndWaitResult(from, "hello", null
                    , BigInteger.valueOf(0), BigInteger.valueOf(100));
            if (txResult == null || txResult.getStatus().compareTo(Constants.STATUS_SUCCESS) != 0) {
                System.out.println("Failed to invoke. result = " + txResult);
                throw new TimeoutException();
            }
        }
        catch (IOException exception) {
            System.out.println("Failed to invoke. Exception : " + exception);
            return false;
        }
        return true;
    }
}
