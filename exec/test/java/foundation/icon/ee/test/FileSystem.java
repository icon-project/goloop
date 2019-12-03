package foundation.icon.ee.test;

import foundation.icon.ee.score.FileReader;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;

public class FileSystem implements FileReader {
    private Map<String, byte[]> files = new HashMap<>();

    public void writeFile(String path, byte[] data) {
        files.put(path, data.clone());
    }

    public byte[] readFile(String path) throws IOException {
        var data = files.get(path);
        if (data!=null) {
            data = data.clone();
        }
        return data;
    }
}
