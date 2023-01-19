package foundation.icon.ee.test;

import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.TestInfo;

import java.io.BufferedReader;
import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.InputStreamReader;
import java.io.PrintStream;
import java.nio.file.Files;
import java.nio.file.NoSuchFileException;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class GoldenTest extends NoDebugTest {
    private ByteArrayOutputStream outContent;
    private PrintStream prevOut;

    protected Path getGoldenFilePath(TestInfo testInfo) {
        String cls = this.getClass().getName().replace('.', '/');
        String method = testInfo.getTestMethod().get().getName();
        return Paths.get("test", "resources", "out", cls + "-" + method + ".txt");
    }

    protected Path getActualFilePath(TestInfo testInfo) {
        String cls = this.getClass().getName().replace('.', '/');
        String method = testInfo.getTestMethod().get().getName();
        return Paths.get("build", "test-results", "out", cls + "-" + method + ".txt");
    }

    @BeforeEach
    public void setUp() {
        outContent = new ByteArrayOutputStream();
        prevOut = System.out;
        System.setOut(new PrintStream(new TeeOutputStream(prevOut, outContent)));

        super.setUp();
    }

    private void mkdirs(Path path) {
        path.toFile().getParentFile().mkdirs();
    }

    @AfterEach
    public void tearDown(TestInfo testInfo) {
        super.tearDown(testInfo);
        System.out.flush();
        System.setOut(prevOut);
        var bis = new ByteArrayInputStream(outContent.toByteArray());
        var r = new BufferedReader(new InputStreamReader(bis));
        var path = getGoldenFilePath(testInfo);
        mkdirs(path);
        List<String> expected;
        try {
            expected = Files.readAllLines(path);
        } catch (NoSuchFileException e) {
            expected = new ArrayList<>();
        } catch (Exception e) {
            throw new AssertionError(e);
        }
        List<String> actual = new ArrayList<>();
        try {
            while (r.ready()) {
                actual.add(r.readLine());
            }
            var actualPath = getActualFilePath(testInfo);
            mkdirs(actualPath);
            Files.write(getActualFilePath(testInfo), actual);
        } catch (Exception e) {
            throw new AssertionError(e);
        }
        assertEquals(expected, actual);
    }
}
