package foundation.icon.ee.test;

import foundation.icon.ee.logger.EELogger;
import foundation.icon.ee.score.TransactionExecutor;
import foundation.icon.ee.tooling.deploy.OptimizedJarBuilder;
import org.aion.avm.core.AvmConfiguration;
import org.aion.avm.utilities.JarBuilder;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.TestInfo;

import java.io.IOException;

public class SimpleTest {
    protected ServiceManager sm;

    private static int prevLogLevel;

    @BeforeAll
    public static void beforeAll() {
        prevLogLevel = EELogger.setLogLevel(0);
    }

    @BeforeEach
    public void setUp() {
        var pipes = Pipe.createPair();
        sm = new ServiceManager(pipes[0]);
        Thread th = new Thread(() -> {
            try {
                var conf = new AvmConfiguration();
                conf.enableContextPrintln = true;
                conf.enableVerboseContractErrors = true;
                var te = TransactionExecutor.newInstance(pipes[1],
                        "",
                        null,
                        sm.getFileIO(),
                        conf);
                te.connectAndRunLoop(sm);
            } catch (IOException e) {
                System.out.println(e);
            }
        });
        th.start();
    }

    public void createAndAcceptNewJAVAEE() {
        var pipes = Pipe.createPair();
        sm.accept(pipes[0]);
        Thread th = new Thread(() -> {
            try {
                var conf = new AvmConfiguration();
                conf.enableContextPrintln = true;
                conf.enableVerboseContractErrors = true;
                var te = TransactionExecutor.newInstance(pipes[1],
                        "",
                        null,
                        sm.getFileIO(),
                        conf);
                te.connectAndRunLoop(sm);
            } catch (IOException e) {
                System.out.println(e);
            }
        });
        th.start();
    }

    @AfterEach
    public void tearDown(TestInfo testInfo) {
        sm.close();
    }

    @AfterAll
    public static void afterAll() {
        EELogger.setLogLevel(prevLogLevel);
    }

    public byte[] makeRelJar(Class<?>... args) {
        var name = args[0].getName();
        byte[] preopt = JarBuilder.buildJarForExplicitMainAndClasses(name, args);
        return new OptimizedJarBuilder(false,
                preopt, true)
                .withUnreachableMethodRemover()
                .withRenamer().withLog(System.out).getOptimizedBytes();
    }
}
