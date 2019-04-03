package foundation.icon.test.score;

import foundation.icon.icx.IconService;
import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.test.common.Constants;

import java.io.IOException;
import java.math.BigInteger;

public class HelloScore extends Score {
    private static final String PATH = Constants.SCORE_ROOT +  "helloScore.zip";

    public static HelloScore mustDeploy(IconService service, Wallet wallet, BigInteger nid)
            throws IOException, TransactionFailureException
    {
        return new HelloScore(
                service,
                Score.mustDeploy(service, wallet, PATH, null),
                nid
        );
    }

    public HelloScore(IconService iconService, Address scoreAddress, BigInteger nid) {
        super(iconService, scoreAddress, nid);
    }
}
