package foundation.icon.ee.ipc;

import java.io.IOException;

public interface Connector {
    Connection connect(String addr) throws IOException;
}
