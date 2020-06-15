package foundation.icon.ee.score;

import java.io.IOException;

public interface FileIO {
    byte[] readFile(String path) throws IOException;
    void writeFile(String path, byte[] bytes) throws IOException;
}
