package foundation.icon.ee.score;

import java.io.IOException;

public interface FileReader {
    byte[] readFile(String path) throws IOException;
}
