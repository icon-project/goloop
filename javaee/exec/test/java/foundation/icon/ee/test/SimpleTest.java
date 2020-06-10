package foundation.icon.ee.test;

import foundation.icon.ee.score.TransactionExecutor;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.TestInfo;

import java.io.IOException;

public class SimpleTest {
    protected ServiceManager sm;

    @BeforeEach
    public void setUp() {
        var pipes = Pipe.createPair();
        sm = new ServiceManager(pipes[0]);
        Thread th = new Thread(() -> {
            try {
                var te = TransactionExecutor.newInstance(pipes[1],
                        "",
                        null,
                        sm.getFileReader());
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
}
