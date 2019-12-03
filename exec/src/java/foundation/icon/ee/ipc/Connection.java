package foundation.icon.ee.ipc;

import java.io.IOException;
import java.io.InputStream;

public interface Connection {
    InputStream getInputStream();
    void close() throws IOException;
    void send(byte[] data) throws IOException;
}
