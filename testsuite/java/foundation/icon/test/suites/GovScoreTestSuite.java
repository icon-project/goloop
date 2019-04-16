package foundation.icon.test.suites;

import foundation.icon.icx.KeyWallet;
import foundation.icon.test.cases.ChainScoreTest;
import foundation.icon.test.cases.DeployTest;
import foundation.icon.test.cases.ScoreTest;
import foundation.icon.test.cases.TransferTest;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import org.junit.AfterClass;
import org.junit.BeforeClass;
import org.junit.runner.RunWith;
import org.junit.runners.Suite;

import static org.junit.Assert.*;

import java.io.File;
import java.io.IOException;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.TimeUnit;

@RunWith(Suite.class)
@Suite.SuiteClasses({
        ChainScoreTest.class,
        TransferTest.class,
        DeployTest.class,
        ScoreTest.class
})
public class GovScoreTestSuite {
    private static boolean bRunChain = true;
    @BeforeClass
    public static void setUp() throws Exception {
        if("no".equals(System.getProperty("runChain"))) {
            bRunChain = false;
        }

        if(bRunChain) {
            startGoLoop();
        }

        KeyWallet god = Env.getGodWallet();
        if(god == null) {
            god = Utils.readWalletFromFile("./data/keystore_god.json", "gochain");
        }
        assertTrue(god!=null);

        Env.Chain chain = new Env.Chain(god);
        List<String> endpoint = Env.getEndpoint();
        if(endpoint==null) {
            endpoint = new LinkedList<>();
            endpoint.add("http://localhost:9080/api/v3");
        }
        Env.Node node = new Env.Node(endpoint.get(0), new Env.Chain[]{chain});
        Env.nodes = new Env.Node[]{node};

        Env.LOG.setLevel(Config.TEST_LOG_LEVEL);
    }

    @AfterClass
    public static void tearDown() {
        if(bRunChain) {
            stopGoLoop();
        }
    }

    // TODO Share the following methods in common class?
    private static Process goLoop;

    public static void startGoLoop() {
        try {
            Runtime.getRuntime().exec("rm -rf .chain");

            // TODO Make it configurable
            // TODO Consider how to print log (care for it later with docker)
            // TODO Get god wallet from config.json, not from additional file.
            ProcessBuilder pb = new ProcessBuilder(
                    "../bin/gochain", "-config=./data/govConfig.json"
                    , "-genesisStorage=./data/genesisStorage.zip");
            Map<String, String> env = pb.environment();
            // TODO how to handle with virtual env
            String separator = System.getProperties().getProperty("path.separator");
            env.put("PATH", "../.venv/bin" + separator + env.get("PATH"));
            env.put("PYTHONPATH", "../pyee");
            pb.directory(new File("."));

            if (Config.WITH_NODE_LOG) {
                pb.redirectError(ProcessBuilder.Redirect.INHERIT);
                pb.redirectOutput(ProcessBuilder.Redirect.INHERIT);
            }

            goLoop = pb.start();
            Thread.sleep(3000);
        } catch (IOException | InterruptedException ex) {
            ex.printStackTrace();
        }
    }

    public static void stopGoLoop() {
        try {
            goLoop.destroy();
            goLoop.getErrorStream().close();
            goLoop.getInputStream().close();
            goLoop.getOutputStream().close();
            goLoop.waitFor(5, TimeUnit.SECONDS);

            Env.LOG.info("Sub process is killed");
        }
        catch (Exception e) {
            e.printStackTrace();
        }
    }
}
