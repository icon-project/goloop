package foundation.icon.test.suites;

import foundation.icon.icx.KeyWallet;
import foundation.icon.test.cases.BasicScoreTest;
import foundation.icon.test.cases.MultiSigWalletTest;
import foundation.icon.test.cases.RevertTest;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Log;
import foundation.icon.test.common.Utils;
import org.junit.AfterClass;
import org.junit.BeforeClass;
import org.junit.runner.RunWith;
import org.junit.runners.Suite;

import java.io.*;
import java.math.BigInteger;
import java.util.Map;

@RunWith(Suite.class)
@Suite.SuiteClasses({
        BasicScoreTest.class,
        MultiSigWalletTest.class,
        RevertTest.class
})
public class SingleChainTestSuite {
    private static final boolean WITH_NODE_LOG = false;
    private static final int TEST_LOG_LEVEL = Log.LEVEL_INFO;

    @BeforeClass
    public static void setUp() throws Exception {
        startGoLoop();

        KeyWallet god = Utils.readWalletFromFile("./data/keystore_god.json", "gochain");
        Env.Chain chain = new Env.Chain(BigInteger.valueOf(3), god);
        Env.Node node = new Env.Node("http://localhost:9080/api/v3", new Env.Chain[]{chain});
        Env.nodes = new Env.Node[]{node};

        Env.LOG.setLevel(TEST_LOG_LEVEL);
    }

    @AfterClass
    public static void tearDown() {
        stopGoLoop();
    }

    // TODO Share the following methods in common class?
    private static Process goLoop;

    public static void startGoLoop() {
        try {
            // TODO Make it configurable
            // TODO Consider how to print log (care for it later with docker)
            // TODO Get god wallet from config.json, not from additional file.
            ProcessBuilder pb = new ProcessBuilder("../bin/gochain", "-config=./data/config.json");
            Map<String, String> env = pb.environment();
            // TODO how to handle with virtual env
            String separator = System.getProperties().getProperty("path.separator");
            env.put("PATH", "../.venv/bin" + separator + env.get("PATH"));
            env.put("PYTHONPATH", "../pyee");
            pb.directory(new File("."));

            // TODO temporary log handling. It may change based on docker.
            if (WITH_NODE_LOG) {
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
        goLoop.destroy();
        if (goLoop.isAlive()) {
            System.out.println("Failed to kill sub process");
        } else {
            System.out.println("Sub process is killed");
        }
    }
}
